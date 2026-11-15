// Package iamfake is an in-memory awsapi.IamApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package iamfake

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type Api struct {
	awsapi.IamApi

	mtx      sync.Mutex
	roles    []types.Role
	role     *types.Role
	users    []types.User
	user     *types.User
	policies []types.Policy
	policy   *types.Policy
	err      error

	principalPolicies []awsapi.PrincipalPolicy
	// documents are the policy documents by policy name, or by ARN for the
	// managed policies of the policies page
	documents  map[string]string
	groups     []types.Group
	accessKeys []types.AccessKeyMetadata
	entities   []awsapi.PolicyEntity
	tags       []types.Tag

	// documentDelays hold back the documents of the policies named
	documentDelays map[string]time.Duration
	// rolesByName and roleDelays give and hold back GetRole by name
	rolesByName map[string]*types.Role
	roleDelays  map[string]time.Duration

	calls []string
}

type Option func(*Api)

// WithRoles sets the roles ListRoles gives
func WithRoles(roles []types.Role) Option { return func(f *Api) { f.roles = roles } }

// WithRole sets the role GetRole gives for a name that WithRolesByName does not
func WithRole(role *types.Role) Option { return func(f *Api) { f.role = role } }

// WithRolesByName sets the role GetRole gives for each name
func WithRolesByName(roles map[string]*types.Role) Option {
	return func(f *Api) { f.rolesByName = maps.Clone(roles) }
}

// WithUsers sets the users ListUsers gives
func WithUsers(users []types.User) Option { return func(f *Api) { f.users = users } }

// WithUser sets the user GetUser gives, for any name
func WithUser(user *types.User) Option { return func(f *Api) { f.user = user } }

// WithPolicies sets the policies ListPolicies gives
func WithPolicies(policies []types.Policy) Option { return func(f *Api) { f.policies = policies } }

// WithPolicy sets the policy GetPolicy gives, for any ARN
func WithPolicy(policy *types.Policy) Option { return func(f *Api) { f.policy = policy } }

// WithPrincipalPolicies sets the policies of every role and user
func WithPrincipalPolicies(policies []awsapi.PrincipalPolicy) Option {
	return func(f *Api) { f.principalPolicies = policies }
}

// WithDocuments sets the policy documents, by the name of the policy of a role
// or user, or by the ARN of a managed policy
func WithDocuments(documents map[string]string) Option {
	return func(f *Api) { f.documents = maps.Clone(documents) }
}

// WithGroups sets the groups of every user
func WithGroups(groups []types.Group) Option { return func(f *Api) { f.groups = groups } }

// WithAccessKeys sets the access keys of every user
func WithAccessKeys(keys []types.AccessKeyMetadata) Option {
	return func(f *Api) { f.accessKeys = keys }
}

// WithEntities sets the entities every managed policy is attached to
func WithEntities(entities []awsapi.PolicyEntity) Option {
	return func(f *Api) { f.entities = entities }
}

// WithTags sets the tags of every role, user and policy
func WithTags(tags []types.Tag) Option { return func(f *Api) { f.tags = tags } }

// WithDocumentDelays holds back the document of each policy of a role or
// user named, by the time given, as a slow api would. Use it in
// uitest.RunInBubble, where the wait takes no real time.
func WithDocumentDelays(delays map[string]time.Duration) Option {
	return func(f *Api) { f.documentDelays = maps.Clone(delays) }
}

// WithRoleDelays holds back GetRole for each role named, by the time given, as
// a slow api would. Use it in uitest.RunInBubble, where the wait takes no real
// time.
func WithRoleDelays(delays map[string]time.Duration) Option {
	return func(f *Api) { f.roleDelays = maps.Clone(delays) }
}

// WithErr makes every call fail
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call fail. Safe while background loads run.
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// Calls returns every call made so far, formatted as "<method> <args...>".
func (f *Api) Calls() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

// CallsWithPrefix returns the calls whose text starts with prefix, e.g. "roles".
func (f *Api) CallsWithPrefix(prefix string) []string {
	return slices.DeleteFunc(f.Calls(), func(call string) bool { return !strings.HasPrefix(call, prefix) })
}

// record notes the call, and gives the error calls fail with. The caller must
// hold the lock.
func (f *Api) record(call string) error {
	f.calls = append(f.calls, call)
	return f.err
}

// delay is how long the call for the name is held back
func (f *Api) delay(delays map[string]time.Duration, name string) time.Duration {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return delays[name]
}

func copyOf[T any](value *T) *T {
	if value == nil {
		return nil
	}
	var copied = *value
	return &copied
}

func (f *Api) ListRoles(ctx context.Context, force bool) ([]types.Role, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.roles), f.record("roles")
}

func (f *Api) GetRole(ctx context.Context, roleName string) (*types.Role, error) {
	time.Sleep(f.delay(f.roleDelays, roleName)) // sleep-ok: simulated api latency
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var err = f.record("role " + roleName)
	if role, ok := f.rolesByName[roleName]; ok {
		return copyOf(role), err
	}
	return copyOf(f.role), err
}

// principalDocument is the document of a policy of a role or user
func (f *Api) principalDocument(call string, policy awsapi.PrincipalPolicy) (string, error) {
	time.Sleep(f.delay(f.documentDelays, policy.Name)) // sleep-ok: simulated api latency
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.documents[policy.Name], f.record(call + " " + policy.Name)
}

func (f *Api) ListRolePolicies(ctx context.Context, roleName string) ([]awsapi.PrincipalPolicy, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.principalPolicies), f.record("role policies " + roleName)
}

func (f *Api) GetRolePolicyDocument(
	ctx context.Context, roleName string, policy awsapi.PrincipalPolicy,
) (string, error) {
	return f.principalDocument("role document "+roleName, policy)
}

func (f *Api) ListRoleTags(ctx context.Context, roleName string) ([]types.Tag, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.tags), f.record("role tags " + roleName)
}

func (f *Api) ListUsers(ctx context.Context, force bool) ([]types.User, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.users), f.record("users")
}

func (f *Api) GetUser(ctx context.Context, userName string) (*types.User, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return copyOf(f.user), f.record("user " + userName)
}

func (f *Api) ListUserPolicies(ctx context.Context, userName string) ([]awsapi.PrincipalPolicy, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.principalPolicies), f.record("user policies " + userName)
}

func (f *Api) GetUserPolicyDocument(
	ctx context.Context, userName string, policy awsapi.PrincipalPolicy,
) (string, error) {
	return f.principalDocument("user document "+userName, policy)
}

func (f *Api) ListUserGroups(ctx context.Context, userName string) ([]types.Group, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.groups), f.record("user groups " + userName)
}

func (f *Api) ListUserAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.accessKeys), f.record("user access keys " + userName)
}

func (f *Api) ListUserTags(ctx context.Context, userName string) ([]types.Tag, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.tags), f.record("user tags " + userName)
}

func (f *Api) ListPolicies(ctx context.Context, force bool) ([]types.Policy, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.policies), f.record("policies")
}

func (f *Api) GetPolicy(ctx context.Context, policyArn string) (*types.Policy, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return copyOf(f.policy), f.record("policy " + policyArn)
}

func (f *Api) GetPolicyDocument(ctx context.Context, policyArn string) (string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.documents[policyArn], f.record("policy document " + policyArn)
}

func (f *Api) ListPolicyEntities(ctx context.Context, policyArn string) ([]awsapi.PolicyEntity, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.entities), f.record("policy entities " + policyArn)
}

func (f *Api) ListPolicyTags(ctx context.Context, policyArn string) ([]types.Tag, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.tags), f.record("policy tags " + policyArn)
}
