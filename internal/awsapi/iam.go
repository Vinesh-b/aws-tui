package awsapi

import (
	"context"
	"log"
	"net/url"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// RolePolicy is a policy of a role, either a managed policy attached to it or
// one of its inline policies, which have no ARN.
type RolePolicy struct {
	Name string
	Arn  string
}

func (inst RolePolicy) IsInline() bool {
	return len(inst.Arn) == 0
}

type IamApi interface {
	ListRoles(ctx context.Context, force bool) ([]types.Role, error)
	// GetRole gives the role with its tags and last use, which ListRoles
	// leaves out. The trust policy document is decoded to json.
	GetRole(ctx context.Context, roleName string) (*types.Role, error)
	// ListRolePolicies gives the managed policies attached to the role and then
	// its inline policies, each sorted by name.
	ListRolePolicies(ctx context.Context, roleName string) ([]RolePolicy, error)
	// GetRolePolicyDocument gives the json document of the policy of the role,
	// which for a managed policy is its default version.
	GetRolePolicyDocument(ctx context.Context, roleName string, policy RolePolicy) (string, error)
	ListRoleTags(ctx context.Context, roleName string) ([]types.Tag, error)
}

type iamApi struct {
	baseApi
}

func NewIamApi(
	logger *log.Logger,
) IamApi {
	return &iamApi{
		baseApi: baseApi{logger: logger},
	}
}

// decodePolicyDocument turns the url encoded documents the api gives into json
func decodePolicyDocument(document string) (string, error) {
	return url.QueryUnescape(document)
}

func (inst *iamApi) ListRoles(ctx context.Context, force bool) ([]types.Role, error) {
	var paginator = iam.NewListRolesPaginator(
		GetAwsApiClients().iam,
		&iam.ListRolesInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.Role, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Roles, nil
	})

	sortByKey(result, func(r types.Role) *string { return r.RoleName })

	return result, inst.logErr(err)
}

func (inst *iamApi) GetRole(ctx context.Context, roleName string) (*types.Role, error) {
	if err := requireArg("role name", roleName); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().iam.GetRole(ctx, &iam.GetRoleInput{RoleName: &roleName})
	if err != nil {
		return nil, inst.logErr(err)
	}

	var role = output.Role
	if role.AssumeRolePolicyDocument != nil {
		var document, err = decodePolicyDocument(*role.AssumeRolePolicyDocument)
		if err != nil {
			return nil, inst.logErr(err)
		}
		role.AssumeRolePolicyDocument = &document
	}

	return role, nil
}

func (inst *iamApi) ListRolePolicies(ctx context.Context, roleName string) ([]RolePolicy, error) {
	if err := requireArg("role name", roleName); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().iam
	var attachedPaginator = iam.NewListAttachedRolePoliciesPaginator(client,
		&iam.ListAttachedRolePoliciesInput{RoleName: &roleName},
	)
	var attached, err = collectPaginator(attachedPaginator.HasMorePages, func() ([]RolePolicy, error) {
		var output, err = attachedPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var policies = make([]RolePolicy, 0, len(output.AttachedPolicies))
		for _, policy := range output.AttachedPolicies {
			policies = append(policies, RolePolicy{
				Name: aws.ToString(policy.PolicyName),
				Arn:  aws.ToString(policy.PolicyArn),
			})
		}
		return policies, nil
	})
	if err != nil {
		return attached, inst.logErr(err)
	}

	var inlinePaginator = iam.NewListRolePoliciesPaginator(client,
		&iam.ListRolePoliciesInput{RoleName: &roleName},
	)
	var inline []RolePolicy
	inline, err = collectPaginator(inlinePaginator.HasMorePages, func() ([]RolePolicy, error) {
		var output, err = inlinePaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var policies = make([]RolePolicy, 0, len(output.PolicyNames))
		for _, name := range output.PolicyNames {
			policies = append(policies, RolePolicy{Name: name})
		}
		return policies, nil
	})

	for _, policies := range [][]RolePolicy{attached, inline} {
		sort.Slice(policies, func(i, j int) bool { return policies[i].Name < policies[j].Name })
	}

	return append(attached, inline...), inst.logErr(err)
}

func (inst *iamApi) GetRolePolicyDocument(
	ctx context.Context, roleName string, policy RolePolicy,
) (string, error) {
	if err := requireArg("role name", roleName); err != nil {
		return "", err
	}
	if err := requireArg("policy name", policy.Name); err != nil {
		return "", err
	}

	var client = GetAwsApiClients().iam
	var document string

	if policy.IsInline() {
		var output, err = client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
			RoleName:   &roleName,
			PolicyName: &policy.Name,
		})
		if err != nil {
			return "", inst.logErr(err)
		}
		document = aws.ToString(output.PolicyDocument)
	} else {
		var managed, err = client.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &policy.Arn})
		if err != nil {
			return "", inst.logErr(err)
		}

		var version *iam.GetPolicyVersionOutput
		version, err = client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
			PolicyArn: &policy.Arn,
			VersionId: managed.Policy.DefaultVersionId,
		})
		if err != nil {
			return "", inst.logErr(err)
		}
		document = aws.ToString(version.PolicyVersion.Document)
	}

	var decoded, err = decodePolicyDocument(document)
	return decoded, inst.logErr(err)
}

func (inst *iamApi) ListRoleTags(ctx context.Context, roleName string) ([]types.Tag, error) {
	if err := requireArg("role name", roleName); err != nil {
		return nil, err
	}

	var paginator = iam.NewListRoleTagsPaginator(
		GetAwsApiClients().iam,
		&iam.ListRoleTagsInput{RoleName: &roleName},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.Tag, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Tags, nil
	})

	return result, inst.logErr(err)
}
