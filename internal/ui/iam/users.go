package iam

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func userName(user types.User) string {
	return aws.ToString(user.UserName)
}

func NewUserListTable(serviceCtx *core.ServiceContext[awsapi.IamApi]) *IamListTable[types.User] {
	return NewIamListTable(serviceCtx, "Users", core.TableRow{"Name", "Path"}, "User",
		userName, userName,
		func(user types.User) core.TableRow {
			return core.TableRow{userName(user), aws.ToString(user.Path)}
		},
		func(ctx context.Context) ([]types.User, error) {
			return serviceCtx.Api.ListUsers(ctx, true)
		},
	)
}

func userDetailsRows(user *types.User) []core.TableRow {
	var rows = []core.TableRow{
		{"Name", aws.ToString(user.UserName)},
		{"ARN", aws.ToString(user.Arn)},
		{"User ID", aws.ToString(user.UserId)},
		{"Path", aws.ToString(user.Path)},
		{"Created", core.FormatTime(user.CreateDate)},
		// Unset for users that have no password or never signed in with it
		{"Password Last Used", core.FormatTime(user.PasswordLastUsed)},
	}

	return append(rows, permissionsBoundaryRows(user.PermissionsBoundary)...)
}

func NewUserDetailsTable(serviceCtx *core.ServiceContext[awsapi.IamApi]) *IamDetailsTable[types.User] {
	return NewIamDetailsTable(serviceCtx, "User Details",
		func(ctx context.Context, key string) (*types.User, error) {
			return serviceCtx.Api.GetUser(ctx, key)
		},
		userDetailsRows,
		func(user *types.User) string { return aws.ToString(user.UserName) },
	)
}

type IamUsersPageView struct {
	*core.ServicePageView
	UserListTable    *IamListTable[types.User]
	UserDetailsTable *IamDetailsTable[types.User]
	PoliciesView     *PrincipalPoliciesView
	GroupsTable      *IamChildTable[types.Group]
	AccessKeysTable  *IamChildTable[types.AccessKeyMetadata]
	UserTagsTable    *core.TagsTable[types.Tag, awsapi.IamApi]
	TabView          *core.TabViewHorizontal
	serviceCtx       *core.ServiceContext[awsapi.IamApi]
}

func NewIamUsersPageView(
	userListTable *IamListTable[types.User],
	userDetailsTable *IamDetailsTable[types.User],
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *IamUsersPageView {
	var tagsTable = core.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() func() ([]types.Tag, error) {
			var userName = userListTable.GetSelectedKey()
			return func() ([]types.Tag, error) {
				return serviceCtx.Api.ListUserTags(serviceCtx.Ctx, userName)
			}
		})

	var policiesView = NewPrincipalPoliciesView(serviceCtx,
		func(ctx context.Context, key string) ([]awsapi.PrincipalPolicy, error) {
			return serviceCtx.Api.ListUserPolicies(ctx, key)
		},
		func(ctx context.Context, principal string, policy awsapi.PrincipalPolicy) (string, error) {
			return serviceCtx.Api.GetUserPolicyDocument(ctx, principal, policy)
		},
	)

	var groupsTable = NewIamChildTable(serviceCtx, "Groups", core.TableRow{"Name", "Path", "Created"},
		func(group types.Group) core.TableRow {
			return core.TableRow{
				aws.ToString(group.GroupName),
				aws.ToString(group.Path),
				core.FormatTime(group.CreateDate),
			}
		},
		func(ctx context.Context, key string) ([]types.Group, error) {
			return serviceCtx.Api.ListUserGroups(ctx, key)
		},
	)

	var accessKeysTable = NewIamChildTable(serviceCtx, "Access Keys", core.TableRow{"Access Key ID", "Status", "Created"},
		func(key types.AccessKeyMetadata) core.TableRow {
			return core.TableRow{
				aws.ToString(key.AccessKeyId),
				string(key.Status),
				core.FormatTime(key.CreateDate),
			}
		},
		func(ctx context.Context, key string) ([]types.AccessKeyMetadata, error) {
			return serviceCtx.Api.ListUserAccessKeys(ctx, key)
		},
	)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(IamTabNameDetails, userDetailsTable, 0, 1, true).
		AddTab(IamTabNamePolicies, policiesView, 0, 1, true).
		AddTab(IamTabNameGroups, groupsTable, 0, 1, true).
		AddTab(IamTabNameAccessKeys, accessKeysTable, 0, 1, true).
		AddTab(IamTabNameTags, tagsTable, 0, 1, true)

	var view = &IamUsersPageView{
		ServicePageView: newIamPageView(serviceCtx.AppContext, tabView, userListTable,
			&userListTable.ErrorMessageCallback,
			&userDetailsTable.ErrorMessageCallback,
			&groupsTable.ErrorMessageCallback,
			&accessKeysTable.ErrorMessageCallback,
			&tagsTable.ErrorMessageCallback,
		),
		UserListTable:    userListTable,
		UserDetailsTable: userDetailsTable,
		PoliciesView:     policiesView,
		GroupsTable:      groupsTable,
		AccessKeysTable:  accessKeysTable,
		UserTagsTable:    tagsTable,
		TabView:          tabView,
		serviceCtx:       serviceCtx,
	}
	policiesView.SetErrorMessageCallback(userListTable.ErrorMessageCallback)

	userListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		var userName = userListTable.GetSelectedKey()
		if userName == "" {
			return
		}
		userDetailsTable.RefreshDetails(userName)
		policiesView.Refresh(userName)
		groupsTable.Refresh(userName)
		accessKeysTable.Refresh(userName)
		tagsTable.RefreshDetails()
	})

	return view
}
