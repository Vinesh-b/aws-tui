// Package cognitofake is an in-memory awsapi.CognitoApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package cognitofake

import (
	"context"
	"maps"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// UsersCall is a call of ListUsers
type UsersCall struct {
	UserPoolId string
	Reset      bool
}

type Api struct {
	awsapi.CognitoApi

	mtx       sync.Mutex
	userPools []idpTypes.UserPoolDescriptionType
	// userPoolDetails are what DescribeUserPool gives, by id
	userPoolDetails map[string]idpTypes.UserPoolType
	// userPages are the pages ListUsers gives in turn, by user pool id
	userPages map[string][][]idpTypes.UserType
	// usersOf and nextUserPage are the pool and page ListUsers gives next
	// without a reset
	usersOf      string
	nextUserPage int
	// clients are what ListUserPoolClients gives, by user pool id
	clients       map[string][]idpTypes.UserPoolClientDescription
	identityPools []idTypes.IdentityPoolShortDescription
	// identityPoolDetails are what DescribeIdentityPool gives, by id
	identityPoolDetails map[string]awsapi.IdentityPool
	// roles are what GetIdentityPoolRoles gives, by identity pool id
	roles map[string]awsapi.IdentityPoolRoles
	err   error

	userPoolsAskedFor     []string
	usersCalls            []UsersCall
	clientsAskedFor       []string
	identityPoolsAskedFor []string
	rolesAskedFor         []string
}

type Option func(*Api)

func WithUserPools(pools []idpTypes.UserPoolDescriptionType) Option {
	return func(f *Api) { f.userPools = slices.Clone(pools) }
}

// WithUserPool gives what DescribeUserPool returns for the id
func WithUserPool(userPoolId string, pool idpTypes.UserPoolType) Option {
	return func(f *Api) { f.userPoolDetails[userPoolId] = cloneUserPool(pool) }
}

// WithUserPages gives the pages ListUsers returns in turn for the user pool,
// after which it returns none
func WithUserPages(userPoolId string, pages ...[]idpTypes.UserType) Option {
	return func(f *Api) {
		var copied [][]idpTypes.UserType
		for _, page := range pages {
			copied = append(copied, cloneUsers(page))
		}
		f.userPages[userPoolId] = copied
	}
}

// WithClients gives what ListUserPoolClients returns for the user pool
func WithClients(userPoolId string, clients []idpTypes.UserPoolClientDescription) Option {
	return func(f *Api) { f.clients[userPoolId] = slices.Clone(clients) }
}

func WithIdentityPools(pools []idTypes.IdentityPoolShortDescription) Option {
	return func(f *Api) { f.identityPools = slices.Clone(pools) }
}

// WithIdentityPool gives what DescribeIdentityPool returns for the id
func WithIdentityPool(identityPoolId string, pool awsapi.IdentityPool) Option {
	return func(f *Api) { f.identityPoolDetails[identityPoolId] = cloneIdentityPool(pool) }
}

// WithRoles gives what GetIdentityPoolRoles returns for the identity pool
func WithRoles(identityPoolId string, roles awsapi.IdentityPoolRoles) Option {
	return func(f *Api) { f.roles[identityPoolId] = cloneRoles(roles) }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{
		userPoolDetails:     map[string]idpTypes.UserPoolType{},
		userPages:           map[string][][]idpTypes.UserType{},
		clients:             map[string][]idpTypes.UserPoolClientDescription{},
		identityPoolDetails: map[string]awsapi.IdentityPool{},
		roles:               map[string]awsapi.IdentityPoolRoles{},
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// cloneUserPool copies the user pool with the settings the UI shows, so the
// test and the UI never share them
func cloneUserPool(pool idpTypes.UserPoolType) idpTypes.UserPoolType {
	pool.UsernameAttributes = slices.Clone(pool.UsernameAttributes)
	pool.AliasAttributes = slices.Clone(pool.AliasAttributes)
	pool.AutoVerifiedAttributes = slices.Clone(pool.AutoVerifiedAttributes)
	pool.UserPoolTags = maps.Clone(pool.UserPoolTags)
	if pool.Policies != nil {
		var policies = *pool.Policies
		if policies.PasswordPolicy != nil {
			var password = *policies.PasswordPolicy
			policies.PasswordPolicy = &password
		}
		pool.Policies = &policies
	}
	if pool.LambdaConfig != nil {
		var config = *pool.LambdaConfig
		pool.LambdaConfig = &config
	}
	if pool.AdminCreateUserConfig != nil {
		var config = *pool.AdminCreateUserConfig
		pool.AdminCreateUserConfig = &config
	}
	if pool.EmailConfiguration != nil {
		var config = *pool.EmailConfiguration
		pool.EmailConfiguration = &config
	}
	return pool
}

func cloneUsers(users []idpTypes.UserType) []idpTypes.UserType {
	var result = slices.Clone(users)
	for i := range result {
		result[i].Attributes = slices.Clone(result[i].Attributes)
	}
	return result
}

func cloneIdentityPool(pool awsapi.IdentityPool) awsapi.IdentityPool {
	pool.CognitoIdentityProviders = slices.Clone(pool.CognitoIdentityProviders)
	pool.SupportedLoginProviders = maps.Clone(pool.SupportedLoginProviders)
	pool.OpenIdConnectProviders = slices.Clone(pool.OpenIdConnectProviders)
	pool.SamlProviders = slices.Clone(pool.SamlProviders)
	pool.Tags = maps.Clone(pool.Tags)
	return pool
}

func cloneRoles(roles awsapi.IdentityPoolRoles) awsapi.IdentityPoolRoles {
	var mappings = map[string]idTypes.RoleMapping{}
	for provider, mapping := range roles.RoleMappings {
		if mapping.RulesConfiguration != nil {
			var config = *mapping.RulesConfiguration
			config.Rules = slices.Clone(config.Rules)
			mapping.RulesConfiguration = &config
		}
		mappings[provider] = mapping
	}
	return awsapi.IdentityPoolRoles{Roles: maps.Clone(roles.Roles), RoleMappings: mappings}
}

// SetErr makes every following call fail
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

func (f *Api) ListUserPools(ctx context.Context) ([]idpTypes.UserPoolDescriptionType, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.userPools), nil
}

// DescribeUserPool gives the user pool of the id, or one with only its id
// when it wasn't given
func (f *Api) DescribeUserPool(ctx context.Context, userPoolId string) (*idpTypes.UserPoolType, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.userPoolsAskedFor = append(f.userPoolsAskedFor, userPoolId)
	if f.err != nil {
		return nil, f.err
	}
	var pool, ok = f.userPoolDetails[userPoolId]
	if !ok {
		pool = idpTypes.UserPoolType{Id: &userPoolId}
	}
	pool = cloneUserPool(pool)
	return &pool, nil
}

// UserPoolsAskedFor is the ids DescribeUserPool was called with, in order
func (f *Api) UserPoolsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.userPoolsAskedFor)
}

// ListUsers gives the pages of the user pool in turn, from the first again
// after a reset or for another pool
func (f *Api) ListUsers(ctx context.Context, userPoolId string, reset bool) ([]idpTypes.UserType, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.usersCalls = append(f.usersCalls, UsersCall{UserPoolId: userPoolId, Reset: reset})
	if f.err != nil {
		return []idpTypes.UserType{}, f.err
	}
	if reset || f.usersOf != userPoolId {
		f.usersOf = userPoolId
		f.nextUserPage = 0
	}
	var pages = f.userPages[userPoolId]
	if f.nextUserPage >= len(pages) {
		return []idpTypes.UserType{}, nil
	}
	var page = cloneUsers(pages[f.nextUserPage])
	f.nextUserPage++
	return page, nil
}

// UsersCalls is the calls of ListUsers, in order
func (f *Api) UsersCalls() []UsersCall {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.usersCalls)
}

func (f *Api) ListUserPoolClients(
	ctx context.Context, userPoolId string,
) ([]idpTypes.UserPoolClientDescription, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.clientsAskedFor = append(f.clientsAskedFor, userPoolId)
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.clients[userPoolId]), nil
}

// ClientsAskedFor is the ids ListUserPoolClients was called with, in order
func (f *Api) ClientsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.clientsAskedFor)
}

func (f *Api) ListIdentityPools(ctx context.Context) ([]idTypes.IdentityPoolShortDescription, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.identityPools), nil
}

// DescribeIdentityPool gives the identity pool of the id, or one with only its
// id when it wasn't given
func (f *Api) DescribeIdentityPool(ctx context.Context, identityPoolId string) (*awsapi.IdentityPool, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.identityPoolsAskedFor = append(f.identityPoolsAskedFor, identityPoolId)
	if f.err != nil {
		return nil, f.err
	}
	var pool, ok = f.identityPoolDetails[identityPoolId]
	if !ok {
		pool = awsapi.IdentityPool{Id: identityPoolId}
	}
	pool = cloneIdentityPool(pool)
	return &pool, nil
}

// IdentityPoolsAskedFor is the ids DescribeIdentityPool was called with, in
// order
func (f *Api) IdentityPoolsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.identityPoolsAskedFor)
}

func (f *Api) GetIdentityPoolRoles(ctx context.Context, identityPoolId string) (*awsapi.IdentityPoolRoles, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.rolesAskedFor = append(f.rolesAskedFor, identityPoolId)
	if f.err != nil {
		return nil, f.err
	}
	var roles = cloneRoles(f.roles[identityPoolId])
	return &roles, nil
}

// RolesAskedFor is the ids GetIdentityPoolRoles was called with, in order
func (f *Api) RolesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.rolesAskedFor)
}
