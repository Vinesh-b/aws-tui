package iam

import (
	"context"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const testRoleArn = "arn:aws:iam::123456789012:role/service-role/orders"

const testPolicyArn = "arn:aws:iam::123456789012:policy/orders-queue"

func newRole(name string) types.Role {
	return types.Role{RoleName: aws.String(name), Path: aws.String("/service-role/")}
}

func newUser(name string) types.User {
	return types.User{UserName: aws.String(name), Path: aws.String("/developers/")}
}

func newPolicy(name string, arn string) types.Policy {
	return types.Policy{PolicyName: aws.String(name), Arn: aws.String(arn), AttachmentCount: aws.Int32(2)}
}

type fakeIamApi struct {
	awsapi.IamApi
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

	// documentDelay holds back the documents of the policies named
	documentDelay map[string]time.Duration
	// rolesByName and roleDelay give and hold back GetRole by name
	rolesByName map[string]*types.Role
	roleDelay   map[string]time.Duration

	mu        sync.Mutex
	listCalls int
}

func (f *fakeIamApi) countList() {
	f.mu.Lock()
	f.listCalls++
	f.mu.Unlock()
}

func (f *fakeIamApi) listRolesCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listCalls
}

func (f *fakeIamApi) ListRoles(ctx context.Context, force bool) ([]types.Role, error) {
	f.countList()
	return f.roles, f.err
}

func (f *fakeIamApi) GetRole(ctx context.Context, roleName string) (*types.Role, error) {
	time.Sleep(f.roleDelay[roleName]) // sleep-ok: simulated api latency
	if role, ok := f.rolesByName[roleName]; ok {
		return role, f.err
	}
	return f.role, f.err
}

func (f *fakeIamApi) principalDocument(policy awsapi.PrincipalPolicy) (string, error) {
	time.Sleep(f.documentDelay[policy.Name]) // sleep-ok: simulated api latency
	return f.documents[policy.Name], f.err
}

func (f *fakeIamApi) ListRolePolicies(ctx context.Context, roleName string) ([]awsapi.PrincipalPolicy, error) {
	return f.principalPolicies, f.err
}

func (f *fakeIamApi) GetRolePolicyDocument(
	ctx context.Context, roleName string, policy awsapi.PrincipalPolicy,
) (string, error) {
	return f.principalDocument(policy)
}

func (f *fakeIamApi) ListRoleTags(ctx context.Context, roleName string) ([]types.Tag, error) {
	return f.tags, f.err
}

func (f *fakeIamApi) ListUsers(ctx context.Context, force bool) ([]types.User, error) {
	return f.users, f.err
}

func (f *fakeIamApi) GetUser(ctx context.Context, userName string) (*types.User, error) {
	return f.user, f.err
}

func (f *fakeIamApi) ListUserPolicies(ctx context.Context, userName string) ([]awsapi.PrincipalPolicy, error) {
	return f.principalPolicies, f.err
}

func (f *fakeIamApi) GetUserPolicyDocument(
	ctx context.Context, userName string, policy awsapi.PrincipalPolicy,
) (string, error) {
	return f.principalDocument(policy)
}

func (f *fakeIamApi) ListUserGroups(ctx context.Context, userName string) ([]types.Group, error) {
	return f.groups, f.err
}

func (f *fakeIamApi) ListUserAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error) {
	return f.accessKeys, f.err
}

func (f *fakeIamApi) ListUserTags(ctx context.Context, userName string) ([]types.Tag, error) {
	return f.tags, f.err
}

func (f *fakeIamApi) ListPolicies(ctx context.Context, force bool) ([]types.Policy, error) {
	return f.policies, f.err
}

func (f *fakeIamApi) GetPolicy(ctx context.Context, policyArn string) (*types.Policy, error) {
	return f.policy, f.err
}

func (f *fakeIamApi) GetPolicyDocument(ctx context.Context, policyArn string) (string, error) {
	return f.documents[policyArn], f.err
}

func (f *fakeIamApi) ListPolicyEntities(ctx context.Context, policyArn string) ([]awsapi.PolicyEntity, error) {
	return f.entities, f.err
}

func (f *fakeIamApi) ListPolicyTags(ctx context.Context, policyArn string) ([]types.Tag, error) {
	return f.tags, f.err
}
