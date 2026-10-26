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

// PrincipalPolicy is a policy of a role or user, either a managed policy
// attached to it or one of its inline policies, which have no ARN.
type PrincipalPolicy struct {
	Name string
	Arn  string
}

func (inst PrincipalPolicy) IsInline() bool {
	return len(inst.Arn) == 0
}

// PolicyEntity is a role, user or group a managed policy is attached to
type PolicyEntity struct {
	Name string
	// Type is "Role", "User" or "Group"
	Type string
}

type IamApi interface {
	ListRoles(ctx context.Context, force bool) ([]types.Role, error)
	// GetRole gives the role with its tags and last use, which ListRoles
	// leaves out. The trust policy document is decoded to json.
	GetRole(ctx context.Context, roleName string) (*types.Role, error)
	// ListRolePolicies gives the managed policies attached to the role and then
	// its inline policies, each sorted by name.
	ListRolePolicies(ctx context.Context, roleName string) ([]PrincipalPolicy, error)
	// GetRolePolicyDocument gives the json document of the policy of the role,
	// which for a managed policy is its default version.
	GetRolePolicyDocument(ctx context.Context, roleName string, policy PrincipalPolicy) (string, error)
	ListRoleTags(ctx context.Context, roleName string) ([]types.Tag, error)

	ListUsers(ctx context.Context, force bool) ([]types.User, error)
	// GetUser gives the user with its tags, which ListUsers leaves out
	GetUser(ctx context.Context, userName string) (*types.User, error)
	// ListUserPolicies gives the managed policies attached to the user and then
	// its inline policies, each sorted by name. Policies the user has from its
	// groups are not included.
	ListUserPolicies(ctx context.Context, userName string) ([]PrincipalPolicy, error)
	// GetUserPolicyDocument gives the json document of the policy of the user,
	// which for a managed policy is its default version.
	GetUserPolicyDocument(ctx context.Context, userName string, policy PrincipalPolicy) (string, error)
	ListUserGroups(ctx context.Context, userName string) ([]types.Group, error)
	ListUserAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error)
	ListUserTags(ctx context.Context, userName string) ([]types.Tag, error)

	// ListPolicies gives the customer managed policies and then the AWS managed
	// policies that are attached to something, each sorted by name. The other
	// AWS managed policies, of which there are over a thousand, are left out.
	ListPolicies(ctx context.Context, force bool) ([]types.Policy, error)
	GetPolicy(ctx context.Context, policyArn string) (*types.Policy, error)
	// GetPolicyDocument gives the json document of the default version of the
	// managed policy
	GetPolicyDocument(ctx context.Context, policyArn string) (string, error)
	// ListPolicyEntities gives the roles, users and groups the managed policy is
	// attached to, in that order and each sorted by name.
	ListPolicyEntities(ctx context.Context, policyArn string) ([]PolicyEntity, error)
	ListPolicyTags(ctx context.Context, policyArn string) ([]types.Tag, error)
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

func sortPolicies(policies []PrincipalPolicy) {
	sort.Slice(policies, func(i, j int) bool { return policies[i].Name < policies[j].Name })
}

func attachedPolicies(attached []types.AttachedPolicy) []PrincipalPolicy {
	var policies = make([]PrincipalPolicy, 0, len(attached))
	for _, policy := range attached {
		policies = append(policies, PrincipalPolicy{
			Name: aws.ToString(policy.PolicyName),
			Arn:  aws.ToString(policy.PolicyArn),
		})
	}
	return policies
}

func inlinePolicies(names []string) []PrincipalPolicy {
	var policies = make([]PrincipalPolicy, 0, len(names))
	for _, name := range names {
		policies = append(policies, PrincipalPolicy{Name: name})
	}
	return policies
}

// managedPolicyDocument is the decoded document of the default version of the
// managed policy
func (inst *iamApi) managedPolicyDocument(ctx context.Context, policyArn string) (string, error) {
	var client = GetAwsApiClients().iam
	var managed, err = client.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &policyArn})
	if err != nil {
		return "", inst.logErr(err)
	}

	var version *iam.GetPolicyVersionOutput
	version, err = client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: &policyArn,
		VersionId: managed.Policy.DefaultVersionId,
	})
	if err != nil {
		return "", inst.logErr(err)
	}

	var decoded string
	decoded, err = decodePolicyDocument(aws.ToString(version.PolicyVersion.Document))
	return decoded, inst.logErr(err)
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

func (inst *iamApi) ListRolePolicies(ctx context.Context, roleName string) ([]PrincipalPolicy, error) {
	if err := requireArg("role name", roleName); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().iam
	var attachedPaginator = iam.NewListAttachedRolePoliciesPaginator(client,
		&iam.ListAttachedRolePoliciesInput{RoleName: &roleName},
	)
	var attached, err = collectPaginator(attachedPaginator.HasMorePages, func() ([]PrincipalPolicy, error) {
		var output, err = attachedPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return attachedPolicies(output.AttachedPolicies), nil
	})
	if err != nil {
		return attached, inst.logErr(err)
	}

	var inlinePaginator = iam.NewListRolePoliciesPaginator(client,
		&iam.ListRolePoliciesInput{RoleName: &roleName},
	)
	var inline []PrincipalPolicy
	inline, err = collectPaginator(inlinePaginator.HasMorePages, func() ([]PrincipalPolicy, error) {
		var output, err = inlinePaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return inlinePolicies(output.PolicyNames), nil
	})

	sortPolicies(attached)
	sortPolicies(inline)

	return append(attached, inline...), inst.logErr(err)
}

func (inst *iamApi) GetRolePolicyDocument(
	ctx context.Context, roleName string, policy PrincipalPolicy,
) (string, error) {
	if err := requireArg("role name", roleName); err != nil {
		return "", err
	}
	if err := requireArg("policy name", policy.Name); err != nil {
		return "", err
	}

	if !policy.IsInline() {
		return inst.managedPolicyDocument(ctx, policy.Arn)
	}

	var output, err = GetAwsApiClients().iam.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
		RoleName:   &roleName,
		PolicyName: &policy.Name,
	})
	if err != nil {
		return "", inst.logErr(err)
	}

	var decoded string
	decoded, err = decodePolicyDocument(aws.ToString(output.PolicyDocument))
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

func (inst *iamApi) ListUsers(ctx context.Context, force bool) ([]types.User, error) {
	var paginator = iam.NewListUsersPaginator(
		GetAwsApiClients().iam,
		&iam.ListUsersInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.User, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Users, nil
	})

	sortByKey(result, func(u types.User) *string { return u.UserName })

	return result, inst.logErr(err)
}

func (inst *iamApi) GetUser(ctx context.Context, userName string) (*types.User, error) {
	if err := requireArg("user name", userName); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().iam.GetUser(ctx, &iam.GetUserInput{UserName: &userName})
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.User, nil
}

func (inst *iamApi) ListUserPolicies(ctx context.Context, userName string) ([]PrincipalPolicy, error) {
	if err := requireArg("user name", userName); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().iam
	var attachedPaginator = iam.NewListAttachedUserPoliciesPaginator(client,
		&iam.ListAttachedUserPoliciesInput{UserName: &userName},
	)
	var attached, err = collectPaginator(attachedPaginator.HasMorePages, func() ([]PrincipalPolicy, error) {
		var output, err = attachedPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return attachedPolicies(output.AttachedPolicies), nil
	})
	if err != nil {
		return attached, inst.logErr(err)
	}

	var inlinePaginator = iam.NewListUserPoliciesPaginator(client,
		&iam.ListUserPoliciesInput{UserName: &userName},
	)
	var inline []PrincipalPolicy
	inline, err = collectPaginator(inlinePaginator.HasMorePages, func() ([]PrincipalPolicy, error) {
		var output, err = inlinePaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return inlinePolicies(output.PolicyNames), nil
	})

	sortPolicies(attached)
	sortPolicies(inline)

	return append(attached, inline...), inst.logErr(err)
}

func (inst *iamApi) GetUserPolicyDocument(
	ctx context.Context, userName string, policy PrincipalPolicy,
) (string, error) {
	if err := requireArg("user name", userName); err != nil {
		return "", err
	}
	if err := requireArg("policy name", policy.Name); err != nil {
		return "", err
	}

	if !policy.IsInline() {
		return inst.managedPolicyDocument(ctx, policy.Arn)
	}

	var output, err = GetAwsApiClients().iam.GetUserPolicy(ctx, &iam.GetUserPolicyInput{
		UserName:   &userName,
		PolicyName: &policy.Name,
	})
	if err != nil {
		return "", inst.logErr(err)
	}

	var decoded string
	decoded, err = decodePolicyDocument(aws.ToString(output.PolicyDocument))
	return decoded, inst.logErr(err)
}

func (inst *iamApi) ListUserGroups(ctx context.Context, userName string) ([]types.Group, error) {
	if err := requireArg("user name", userName); err != nil {
		return nil, err
	}

	var paginator = iam.NewListGroupsForUserPaginator(
		GetAwsApiClients().iam,
		&iam.ListGroupsForUserInput{UserName: &userName},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.Group, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Groups, nil
	})

	sortByKey(result, func(g types.Group) *string { return g.GroupName })

	return result, inst.logErr(err)
}

func (inst *iamApi) ListUserAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error) {
	if err := requireArg("user name", userName); err != nil {
		return nil, err
	}

	var paginator = iam.NewListAccessKeysPaginator(
		GetAwsApiClients().iam,
		&iam.ListAccessKeysInput{UserName: &userName},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.AccessKeyMetadata, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.AccessKeyMetadata, nil
	})

	return result, inst.logErr(err)
}

func (inst *iamApi) ListUserTags(ctx context.Context, userName string) ([]types.Tag, error) {
	if err := requireArg("user name", userName); err != nil {
		return nil, err
	}

	var paginator = iam.NewListUserTagsPaginator(
		GetAwsApiClients().iam,
		&iam.ListUserTagsInput{UserName: &userName},
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

func (inst *iamApi) listPolicies(ctx context.Context, input *iam.ListPoliciesInput) ([]types.Policy, error) {
	var paginator = iam.NewListPoliciesPaginator(GetAwsApiClients().iam, input)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.Policy, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Policies, nil
	})

	sortByKey(result, func(p types.Policy) *string { return p.PolicyName })

	return result, err
}

func (inst *iamApi) ListPolicies(ctx context.Context, force bool) ([]types.Policy, error) {
	var customer, err = inst.listPolicies(ctx, &iam.ListPoliciesInput{Scope: types.PolicyScopeTypeLocal})
	if err != nil {
		return customer, inst.logErr(err)
	}

	var awsManaged []types.Policy
	awsManaged, err = inst.listPolicies(ctx, &iam.ListPoliciesInput{
		Scope:        types.PolicyScopeTypeAws,
		OnlyAttached: true,
	})

	return append(customer, awsManaged...), inst.logErr(err)
}

func (inst *iamApi) GetPolicy(ctx context.Context, policyArn string) (*types.Policy, error) {
	if err := requireArg("policy ARN", policyArn); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().iam.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &policyArn})
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.Policy, nil
}

func (inst *iamApi) GetPolicyDocument(ctx context.Context, policyArn string) (string, error) {
	if err := requireArg("policy ARN", policyArn); err != nil {
		return "", err
	}
	return inst.managedPolicyDocument(ctx, policyArn)
}

func (inst *iamApi) ListPolicyEntities(ctx context.Context, policyArn string) ([]PolicyEntity, error) {
	if err := requireArg("policy ARN", policyArn); err != nil {
		return nil, err
	}

	var paginator = iam.NewListEntitiesForPolicyPaginator(
		GetAwsApiClients().iam,
		&iam.ListEntitiesForPolicyInput{PolicyArn: &policyArn},
	)

	var roles, users, groups []PolicyEntity
	var _, err = collectPaginator(paginator.HasMorePages, func() ([]PolicyEntity, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, role := range output.PolicyRoles {
			roles = append(roles, PolicyEntity{Name: aws.ToString(role.RoleName), Type: "Role"})
		}
		for _, user := range output.PolicyUsers {
			users = append(users, PolicyEntity{Name: aws.ToString(user.UserName), Type: "User"})
		}
		for _, group := range output.PolicyGroups {
			groups = append(groups, PolicyEntity{Name: aws.ToString(group.GroupName), Type: "Group"})
		}
		return nil, nil
	})

	var result []PolicyEntity
	for _, entities := range [][]PolicyEntity{roles, users, groups} {
		sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
		result = append(result, entities...)
	}

	return result, inst.logErr(err)
}

func (inst *iamApi) ListPolicyTags(ctx context.Context, policyArn string) ([]types.Tag, error) {
	if err := requireArg("policy ARN", policyArn); err != nil {
		return nil, err
	}

	var paginator = iam.NewListPolicyTagsPaginator(
		GetAwsApiClients().iam,
		&iam.ListPolicyTagsInput{PolicyArn: &policyArn},
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
