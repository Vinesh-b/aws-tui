package iam

import (
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func newRunningRoleList(t *testing.T, api *fakeIamApi) *IamListTable[types.Role] {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](api)
	var table = NewRoleListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return table
}

func TestRoleListTable__RefreshLoadsRolesFromApi(t *testing.T) {
	var table = newRunningRoleList(t, &fakeIamApi{roles: []types.Role{newRole("orders"), newRole("billing")}})

	table.Refresh(true)

	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{
		{"Name", "Path"},
		{"orders", "/service-role/"},
		{"billing", "/service-role/"},
	})
}

func TestRoleListTable__RefreshReportsApiError(t *testing.T) {
	var table = newRunningRoleList(t, &fakeIamApi{err: errors.New("access denied")})
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.Refresh(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q", got)
	}
}

func TestRoleListTable__FilterByNameShowsMatchingRoles(t *testing.T) {
	var table = newRunningRoleList(t, &fakeIamApi{roles: []types.Role{newRole("orders"), newRole("billing")}})
	table.Refresh(true)
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterByName("bill")

	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{{"Name", "Path"}, {"billing", "/service-role/"}})
}

func TestRoleListTable__OpenRoleLoadsListToSelectIt(t *testing.T) {
	var api = &fakeIamApi{roles: []types.Role{newRole("billing"), newRole("orders")}}
	var table = newRunningRoleList(t, api)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.Open("orders") })

	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetSelectedKey() == "orders" })

	uitest.OnUI(t, table.serviceCtx.App, func() { table.Open("billing") })
	uitest.OnUI(t, table.serviceCtx.App, func() {
		if got := table.GetSelectedKey(); got != "billing" {
			t.Errorf("selected = %q, want billing", got)
		}
	})
	if got := api.listRolesCalls(); got != 1 {
		t.Errorf("list calls = %d, want 1 as the second role was already loaded", got)
	}
}

func TestRoleListTable__OpenMissingRoleReportsIt(t *testing.T) {
	var table = newRunningRoleList(t, &fakeIamApi{roles: []types.Role{newRole("orders")}})
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	uitest.OnUI(t, table.serviceCtx.App, func() { table.Open("billing") })

	if got := uitest.AwaitError(t, received); got != "Role billing not found" {
		t.Errorf("message = %q", got)
	}
}

func newDetailsRole() *types.Role {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	var used = time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	return &types.Role{
		RoleName:                 aws.String("orders"),
		Arn:                      aws.String(testRoleArn),
		RoleId:                   aws.String("AROAEXAMPLE"),
		Path:                     aws.String("/service-role/"),
		Description:              aws.String("Runs the orders lambda"),
		CreateDate:               &created,
		MaxSessionDuration:       aws.Int32(3600),
		AssumeRolePolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[]}`),
		RoleLastUsed:             &types.RoleLastUsed{LastUsedDate: &used, Region: aws.String("us-east-1")},
		PermissionsBoundary: &types.AttachedPermissionsBoundary{
			PermissionsBoundaryType: types.PermissionsBoundaryAttachmentTypePolicy,
			PermissionsBoundaryArn:  aws.String("arn:aws:iam::123456789012:policy/boundary"),
		},
	}
}

func newRoleDetails(role *types.Role) *IamDetailsTable[types.Role] {
	var table = NewRoleDetailsTable(uitest.NewServiceContext[awsapi.IamApi]())
	table.data = role
	table.populateDetailsTable()
	return table
}

func TestRoleDetailsTable__NoRoleShowsNothing(t *testing.T) {
	var table = newRoleDetails(nil)

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestRoleDetailsTable__ShowsRole(t *testing.T) {
	var table = newRoleDetails(newDetailsRole())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"ARN", testRoleArn},
		{"Role ID", "AROAEXAMPLE"},
		{"Path", "/service-role/"},
		{"Description", "Runs the orders lambda"},
		{"Created", "2024-01-02 03:04:05"},
		{"Max Session Duration (s)", "3600"},
		{"Last Used"},
		{"Date", "2024-02-03 04:05:06"},
		{"Region", "us-east-1"},
		{"Permissions Boundary"},
		{"Type", "PermissionsBoundaryPolicy"},
		{"Policy ARN", "arn:aws:iam::123456789012:policy/boundary"},
	})
	if got := table.GetTitle(); got != "Role Details ❬orders❭" {
		t.Errorf("title = %q", got)
	}
	if link, _ := table.GetLink(12); link != "arn:aws:iam::123456789012:policy/boundary" {
		t.Errorf("boundary link = %q, want the policy", link)
	}
}

func TestRoleDetailsTable__RoleWithoutLastUseOrBoundaryLeavesSectionsOut(t *testing.T) {
	var role = newDetailsRole()
	role.RoleLastUsed = nil
	role.PermissionsBoundary = nil

	var table = newRoleDetails(role)

	if got := table.GetCellText(7, 0); got != "" {
		t.Errorf("unexpected row %q after the general rows", got)
	}
}

func TestRoleDetailsTable__SlowerEarlierRoleDoesNotReplaceLatest(t *testing.T) {
	var billing = newDetailsRole()
	billing.RoleName = aws.String("billing")
	var api = &fakeIamApi{
		rolesByName: map[string]*types.Role{"billing": billing, "orders": newDetailsRole()},
		roleDelay:   map[string]time.Duration{"billing": 300 * time.Millisecond},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](api)
	var table = NewRoleDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() {
		table.RefreshDetails("billing")
		table.RefreshDetails("orders")
	})
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(0, 1) == "orders" })
	time.Sleep(500 * time.Millisecond)

	uitest.OnUI(t, serviceCtx.App, func() {
		if got := table.GetCellText(0, 1); got != "orders" {
			t.Errorf("name = %q, want orders", got)
		}
		if got := table.GetTitle(); got != "Role Details ❬orders❭" {
			t.Errorf("title = %q, want the latest role's", got)
		}
	})
}

func newRolesPage(t *testing.T, api *fakeIamApi) *IamRolesPageView {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](api)
	var view = NewIamRolesPageView(NewRoleListTable(serviceCtx), NewRoleDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestRolesPage__SelectingRoleLoadsItsDetailsTrustPolicyAndPolicies(t *testing.T) {
	var api = &fakeIamApi{
		roles: []types.Role{newRole("orders")},
		role:  newDetailsRole(),
		principalPolicies: []awsapi.PrincipalPolicy{
			{Name: "Logs", Arn: "arn:aws:iam::1:policy/Logs"}, {Name: "queue-access"},
		},
		documents: map[string]string{"Logs": `{"Version":"2012-10-17"}`},
		tags:      []types.Tag{{Key: aws.String("team"), Value: aws.String("orders")}},
	}
	var view = newRolesPage(t, api)
	var app = view.serviceCtx.App

	view.RoleListTable.Refresh(true)

	uitest.Eventually(t, app, func() bool { return view.RoleDetailsTable.GetCellText(0, 1) == "orders" })
	uitest.Eventually(t, app, func() bool {
		return view.TrustPolicyView.GetText() == "{\n  \"Statement\": [],\n  \"Version\": \"2012-10-17\"\n}"
	})
	uitest.AwaitRows(t, app, view.PoliciesView.PoliciesTable, []core.TableRow{
		{"Name", "Type"},
		{"Logs", "Managed"},
		{"queue-access", "Inline"},
	})
	uitest.Eventually(t, app, func() bool {
		return view.PoliciesView.DocumentView.GetText() == "{\n  \"Version\": \"2012-10-17\"\n}"
	})
	uitest.Eventually(t, app, func() bool { return view.RoleTagsTable.GetCellText(0, 1) == "orders" })
}

func TestPrincipalPoliciesView__SlowDocumentOfPreviousPolicyIsDropped(t *testing.T) {
	var api = &fakeIamApi{
		principalPolicies: []awsapi.PrincipalPolicy{{Name: "slow"}, {Name: "fast"}},
		documents:         map[string]string{"slow": `{"Name":"slow"}`, "fast": `{"Name":"fast"}`},
		documentDelay:     map[string]time.Duration{"slow": 200 * time.Millisecond},
	}
	var view = newRolesPage(t, api)
	var app = view.serviceCtx.App
	var policies = view.PoliciesView

	uitest.OnUI(t, app, func() { policies.Refresh("orders") })
	uitest.Eventually(t, app, func() bool { return policies.PoliciesTable.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, app, func() { policies.PoliciesTable.Select(2, 0) })

	uitest.Eventually(t, app, func() bool { return policies.DocumentView.GetText() == "{\n  \"Name\": \"fast\"\n}" })
	time.Sleep(300 * time.Millisecond)
	uitest.OnUI(t, app, func() {
		if got := policies.DocumentView.GetText(); got != "{\n  \"Name\": \"fast\"\n}" {
			t.Errorf("document = %q, want the selected policy's", got)
		}
		if got := policies.DocumentView.GetTitle(); got != "Policy Document ❬fast❭" {
			t.Errorf("document title = %q, want the selected policy's", got)
		}
	})
}
