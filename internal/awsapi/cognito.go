package awsapi

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// cognitoPageSize is the most items a Cognito list call can give in one page
const cognitoPageSize = 60

// CognitoArn is what a Cognito ARN names, whose resource is
// "<kind>/<id>", such as "userpool/us-east-1_aBc123" or
// "identitypool/us-east-1:1a2b3c4d-..."
type CognitoArn struct {
	// Service is "cognito-idp" for user pools, or "cognito-identity" for
	// identity pools
	Service string
	// Kind is the kind of resource, such as "userpool" or "identitypool"
	Kind string
	Id   string
}

// ParseCognitoArn is what the Cognito ARN names
func ParseCognitoArn(resourceArn string) (CognitoArn, error) {
	var parsed, err = arn.Parse(resourceArn)
	if err != nil {
		return CognitoArn{}, fmt.Errorf("invalid Cognito ARN %q: %w", resourceArn, err)
	}

	var kind, id, found = strings.Cut(parsed.Resource, "/")
	var known = parsed.Service == "cognito-idp" || parsed.Service == "cognito-identity"
	if !known || !found || len(kind) == 0 || len(id) == 0 || strings.Contains(id, "/") {
		return CognitoArn{}, fmt.Errorf("invalid Cognito ARN %q", resourceArn)
	}
	return CognitoArn{Service: parsed.Service, Kind: kind, Id: id}, nil
}

// IdentityPool is an identity pool with the identity providers it trusts
type IdentityPool struct {
	Id                       string
	Name                     string
	AllowUnauthenticated     bool
	AllowClassicFlow         bool
	DeveloperProviderName    string
	CognitoIdentityProviders []idTypes.CognitoIdentityProvider
	// SupportedLoginProviders are the app ids of the public providers, by
	// their domain, such as "accounts.google.com"
	SupportedLoginProviders map[string]string
	OpenIdConnectProviders  []string
	SamlProviders           []string
	Tags                    map[string]string
}

// IdentityPoolRoles are the roles an identity pool gives its identities
type IdentityPoolRoles struct {
	// Roles are the default role ARNs, by "authenticated" or "unauthenticated"
	Roles map[string]string
	// RoleMappings are how the role is chosen for the users of a provider,
	// by the provider, such as "cognito-idp.us-east-1.amazonaws.com/<pool
	// id>:<client id>"
	RoleMappings map[string]idTypes.RoleMapping
}

// CognitoApi browses the user pools and identity pools of the region
type CognitoApi interface {
	// ListUserPools returns the user pools, sorted by name
	ListUserPools(ctx context.Context) ([]idpTypes.UserPoolDescriptionType, error)
	// DescribeUserPool returns the user pool of the id with all its settings
	DescribeUserPool(ctx context.Context, userPoolId string) (*idpTypes.UserPoolType, error)
	// ListUsers returns the next page of the users of the user pool. A pool
	// can have millions, so they are only loaded a page at a time. Reset
	// starts again from the first, which it does for another pool too. An
	// empty result without an error is the end of the users.
	ListUsers(ctx context.Context, userPoolId string, reset bool) ([]idpTypes.UserType, error)
	// ListUserPoolClients returns the app clients of the user pool, sorted by
	// name
	ListUserPoolClients(ctx context.Context, userPoolId string) ([]idpTypes.UserPoolClientDescription, error)
	// ListIdentityPools returns the identity pools, sorted by name
	ListIdentityPools(ctx context.Context) ([]idTypes.IdentityPoolShortDescription, error)
	// DescribeIdentityPool returns the identity pool of the id with the
	// providers it trusts
	DescribeIdentityPool(ctx context.Context, identityPoolId string) (*IdentityPool, error)
	// GetIdentityPoolRoles returns the roles the identity pool gives
	GetIdentityPoolRoles(ctx context.Context, identityPoolId string) (*IdentityPoolRoles, error)
}

type cognitoApi struct {
	baseApi
	// usersMtx keeps the pages in order when a reset comes while a page is
	// loading
	usersMtx sync.Mutex
	users    *cognitoidentityprovider.ListUsersPaginator
	// usersOf is the user pool the users paginator is for
	usersOf string
}

func NewCognitoApi(
	clients *AwsApiClients,
	logger *log.Logger,
) CognitoApi {
	return &cognitoApi{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

func (inst *cognitoApi) ListUserPools(ctx context.Context) ([]idpTypes.UserPoolDescriptionType, error) {
	var paginator = cognitoidentityprovider.NewListUserPoolsPaginator(
		inst.clients.cognitoidp,
		&cognitoidentityprovider.ListUserPoolsInput{MaxResults: aws.Int32(cognitoPageSize)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]idpTypes.UserPoolDescriptionType, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.UserPools, nil
	})

	sortByKey(result, func(pool idpTypes.UserPoolDescriptionType) *string { return pool.Name })

	return result, inst.logErr(err)
}

func (inst *cognitoApi) DescribeUserPool(ctx context.Context, userPoolId string) (*idpTypes.UserPoolType, error) {
	if err := requireArg("user pool id", userPoolId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.cognitoidp.DescribeUserPool(ctx,
		&cognitoidentityprovider.DescribeUserPoolInput{UserPoolId: aws.String(userPoolId)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.UserPool, nil
}

func (inst *cognitoApi) ListUsers(ctx context.Context, userPoolId string, reset bool) ([]idpTypes.UserType, error) {
	if err := requireArg("user pool id", userPoolId); err != nil {
		return nil, err
	}

	inst.usersMtx.Lock()
	defer inst.usersMtx.Unlock()

	if inst.users == nil || reset || inst.usersOf != userPoolId {
		inst.users = cognitoidentityprovider.NewListUsersPaginator(
			inst.clients.cognitoidp,
			&cognitoidentityprovider.ListUsersInput{
				UserPoolId: aws.String(userPoolId),
				Limit:      aws.Int32(cognitoPageSize),
			},
		)
		inst.usersOf = userPoolId
	}

	// A page can be empty with more after it, which would look like the end
	for inst.users.HasMorePages() {
		var output, err = inst.users.NextPage(ctx)
		if err != nil {
			return []idpTypes.UserType{}, inst.logErr(err)
		}
		if len(output.Users) > 0 {
			return output.Users, nil
		}
	}
	return []idpTypes.UserType{}, nil
}

func (inst *cognitoApi) ListUserPoolClients(
	ctx context.Context, userPoolId string,
) ([]idpTypes.UserPoolClientDescription, error) {
	if err := requireArg("user pool id", userPoolId); err != nil {
		return nil, err
	}

	var paginator = cognitoidentityprovider.NewListUserPoolClientsPaginator(
		inst.clients.cognitoidp,
		&cognitoidentityprovider.ListUserPoolClientsInput{
			UserPoolId: aws.String(userPoolId),
			MaxResults: aws.Int32(cognitoPageSize),
		},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]idpTypes.UserPoolClientDescription, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.UserPoolClients, nil
	})

	sortByKey(result, func(client idpTypes.UserPoolClientDescription) *string { return client.ClientName })

	return result, inst.logErr(err)
}

func (inst *cognitoApi) ListIdentityPools(ctx context.Context) ([]idTypes.IdentityPoolShortDescription, error) {
	var paginator = cognitoidentity.NewListIdentityPoolsPaginator(
		inst.clients.cognitoid,
		&cognitoidentity.ListIdentityPoolsInput{MaxResults: aws.Int32(cognitoPageSize)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]idTypes.IdentityPoolShortDescription, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.IdentityPools, nil
	})

	sortByKey(result, func(pool idTypes.IdentityPoolShortDescription) *string { return pool.IdentityPoolName })

	return result, inst.logErr(err)
}

func (inst *cognitoApi) DescribeIdentityPool(ctx context.Context, identityPoolId string) (*IdentityPool, error) {
	if err := requireArg("identity pool id", identityPoolId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.cognitoid.DescribeIdentityPool(ctx,
		&cognitoidentity.DescribeIdentityPoolInput{IdentityPoolId: aws.String(identityPoolId)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	var providers = output.CognitoIdentityProviders
	slices.SortStableFunc(providers, func(a, b idTypes.CognitoIdentityProvider) int {
		return cmp.Or(
			cmp.Compare(aws.ToString(a.ProviderName), aws.ToString(b.ProviderName)),
			cmp.Compare(aws.ToString(a.ClientId), aws.ToString(b.ClientId)),
		)
	})

	return &IdentityPool{
		Id:                       aws.ToString(output.IdentityPoolId),
		Name:                     aws.ToString(output.IdentityPoolName),
		AllowUnauthenticated:     output.AllowUnauthenticatedIdentities,
		AllowClassicFlow:         aws.ToBool(output.AllowClassicFlow),
		DeveloperProviderName:    aws.ToString(output.DeveloperProviderName),
		CognitoIdentityProviders: providers,
		SupportedLoginProviders:  output.SupportedLoginProviders,
		OpenIdConnectProviders:   slices.Sorted(slices.Values(output.OpenIdConnectProviderARNs)),
		SamlProviders:            slices.Sorted(slices.Values(output.SamlProviderARNs)),
		Tags:                     output.IdentityPoolTags,
	}, nil
}

func (inst *cognitoApi) GetIdentityPoolRoles(ctx context.Context, identityPoolId string) (*IdentityPoolRoles, error) {
	if err := requireArg("identity pool id", identityPoolId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.cognitoid.GetIdentityPoolRoles(ctx,
		&cognitoidentity.GetIdentityPoolRolesInput{IdentityPoolId: aws.String(identityPoolId)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}
	return &IdentityPoolRoles{Roles: output.Roles, RoleMappings: output.RoleMappings}, nil
}
