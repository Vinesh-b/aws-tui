package iam

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const testUserArn = "arn:aws:iam::123456789012:user/developers/alice"

func newDetailsUser() *types.User {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return &types.User{
		UserName:   aws.String("alice"),
		Arn:        aws.String(testUserArn),
		UserId:     aws.String("AIDAEXAMPLE"),
		Path:       aws.String("/developers/"),
		CreateDate: &created,
	}
}

func TestUserDetailsTable__ShowsUser(t *testing.T) {
	var table = NewUserDetailsTable(uitest.NewServiceContext[awsapi.IamApi]())
	table.data = newDetailsUser()

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "alice"},
		{"ARN", testUserArn},
		{"User ID", "AIDAEXAMPLE"},
		{"Path", "/developers/"},
		{"Created", "2024-01-02 03:04:05"},
		{"Password Last Used", ""},
	})
	if got := table.GetTitle(); got != "User Details ❬alice❭" {
		t.Errorf("title = %q", got)
	}
}

func TestUserListTable__ShowsUsers(t *testing.T) {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](&fakeIamApi{users: []types.User{newUser("alice"), newUser("bob")}})
	var table = NewUserListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.Refresh(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Name", "Path"},
		{"alice", "/developers/"},
		{"bob", "/developers/"},
	})
}

func TestUsersPage__SelectingUserLoadsItsTabs(t *testing.T) {
	var keyCreated = time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	var api = &fakeIamApi{
		users:             []types.User{newUser("alice")},
		user:              newDetailsUser(),
		principalPolicies: []awsapi.PrincipalPolicy{{Name: "deploy"}},
		documents:         map[string]string{"deploy": `{"Statement":[]}`},
		groups:            []types.Group{{GroupName: aws.String("developers"), Path: aws.String("/")}},
		accessKeys: []types.AccessKeyMetadata{{
			AccessKeyId: aws.String("AKIAEXAMPLE"), Status: types.StatusTypeActive, CreateDate: &keyCreated,
		}},
		tags: []types.Tag{{Key: aws.String("team"), Value: aws.String("platform")}},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](api)
	var view = NewIamUsersPageView(NewUserListTable(serviceCtx), NewUserDetailsTable(serviceCtx), serviceCtx)
	var app = serviceCtx.App
	uitest.RunApp(t, app)

	view.UserListTable.Refresh(true)

	uitest.Eventually(t, app, func() bool { return view.UserDetailsTable.GetCellText(0, 1) == "alice" })
	uitest.AwaitRows(t, app, view.PoliciesView.PoliciesTable, []core.TableRow{{"Name", "Type"}, {"deploy", "Inline"}})
	uitest.Eventually(t, app, func() bool {
		return view.PoliciesView.DocumentView.GetText() == "{\n  \"Statement\": []\n}"
	})
	uitest.AwaitRows(t, app, view.GroupsTable, []core.TableRow{{"Name", "Path", "Created"}, {"developers", "/", ""}})
	uitest.AwaitRows(t, app, view.AccessKeysTable, []core.TableRow{
		{"Access Key ID", "Status", "Created"},
		{"AKIAEXAMPLE", "Active", "2024-03-04 05:06:07"},
	})
	uitest.Eventually(t, app, func() bool { return view.UserTagsTable.GetCellText(0, 1) == "platform" })
	uitest.OnUI(t, app, func() {
		if got := view.GroupsTable.GetTitle(); got != "Groups ❬1❭ ❬alice❭" {
			t.Errorf("groups title = %q", got)
		}
	})
}
