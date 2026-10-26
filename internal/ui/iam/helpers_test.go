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

func newRole(name string) types.Role {
	return types.Role{RoleName: aws.String(name), Path: aws.String("/service-role/")}
}

type fakeIamApi struct {
	awsapi.IamApi
	roles     []types.Role
	role      *types.Role
	policies  []awsapi.RolePolicy
	documents map[string]string
	tags      []types.Tag
	err       error
	// documentDelay holds back the documents of the policies named
	documentDelay map[string]time.Duration
	// rolesByName and roleDelay give and hold back GetRole by name
	rolesByName map[string]*types.Role
	roleDelay   map[string]time.Duration

	mu        sync.Mutex
	listCalls int
}

func (f *fakeIamApi) ListRoles(ctx context.Context, force bool) ([]types.Role, error) {
	f.mu.Lock()
	f.listCalls++
	f.mu.Unlock()
	return f.roles, f.err
}

func (f *fakeIamApi) listRolesCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listCalls
}

func (f *fakeIamApi) GetRole(ctx context.Context, roleName string) (*types.Role, error) {
	time.Sleep(f.roleDelay[roleName])
	if role, ok := f.rolesByName[roleName]; ok {
		return role, f.err
	}
	return f.role, f.err
}

func (f *fakeIamApi) ListRolePolicies(ctx context.Context, roleName string) ([]awsapi.RolePolicy, error) {
	return f.policies, f.err
}

func (f *fakeIamApi) GetRolePolicyDocument(
	ctx context.Context, roleName string, policy awsapi.RolePolicy,
) (string, error) {
	time.Sleep(f.documentDelay[policy.Name])
	return f.documents[policy.Name], f.err
}

func (f *fakeIamApi) ListRoleTags(ctx context.Context, roleName string) ([]types.Tag, error) {
	return f.tags, f.err
}
