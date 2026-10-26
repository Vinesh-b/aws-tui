package iam

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func TestPrincipalNameFromArn(t *testing.T) {
	var tests = []struct {
		arn          string
		resourceType string
		want         string
		wantOk       bool
	}{
		{"arn:aws:iam::123456789012:role/orders", "role", "orders", true},
		{testRoleArn, "role", "orders", true},
		{testUserArn, "user", "alice", true},
		{testUserArn, "role", "", false},
		{"arn:aws:iam::123456789012:role/", "role", "", false},
	}

	for _, tt := range tests {
		var parsed, _ = arn.Parse(tt.arn)
		var got, ok = principalNameFromArn(parsed, tt.resourceType)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("%s as %s: got %q, %v, want %q, %v", tt.arn, tt.resourceType, got, ok, tt.want, tt.wantOk)
		}
	}
}

func TestHomeView__OpensRolesUsersAndPolicies(t *testing.T) {
	var tests = []struct {
		name string
		arn  string
		want string
	}{
		{"role", testRoleArn, "Role Details ❬orders❭"},
		{"user", testUserArn, "User Details ❬alice❭"},
		{"policy", testPolicyArn, "Policy Details ❬orders-queue❭"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var appCtx = uitest.NewAppContext()
			var api = &fakeIamApi{
				roles:    []types.Role{newRole("billing"), newRole("orders")},
				role:     newDetailsRole(),
				users:    []types.User{newUser("alice")},
				user:     newDetailsUser(),
				policies: []types.Policy{newPolicy("orders-queue", testPolicyArn)},
				policy:   &types.Policy{PolicyName: newPolicy("orders-queue", testPolicyArn).PolicyName},
			}
			var page = NewIamHomeView(appCtx, api)
			var screen = uitest.ShowPage(t, appCtx.App, page)
			var opener, ok = page.(core.ResourceOpener)
			if !ok || !opener.OpensArnService("iam") {
				t.Fatal("iam page should open iam resources")
			}
			var resource, _ = arn.Parse(tt.arn)

			uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

			uitest.AwaitScreenText(t, appCtx.App, screen, tt.want)
		})
	}
}
