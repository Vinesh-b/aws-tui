package iam

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type IamTabName = string

const (
	IamTabNameDetails     IamTabName = "Details"
	IamTabNameTrustPolicy IamTabName = "Trust Policy"
	IamTabNamePolicies    IamTabName = "Policies"
	IamTabNameGroups      IamTabName = "Groups"
	IamTabNameAccessKeys  IamTabName = "Access Keys"
	IamTabNameDocument    IamTabName = "Document"
	IamTabNameAttachedTo  IamTabName = "Attached To"
)

func roleName(role types.Role) string {
	return aws.ToString(role.RoleName)
}

func NewRoleListTable(serviceCtx *core.ServiceContext[awsapi.IamApi]) *IamListTable[types.Role] {
	return NewIamListTable(serviceCtx, "Roles", core.TableRow{"Name", "Path"}, "Role",
		roleName,
		func(role types.Role) core.TableRow {
			return core.TableRow{roleName(role), aws.ToString(role.Path)}
		},
		func(ctx context.Context) ([]types.Role, error) {
			return serviceCtx.Api.ListRoles(ctx, true)
		},
	)
}

func permissionsBoundaryRows(boundary *types.AttachedPermissionsBoundary) []core.TableRow {
	if boundary == nil {
		return nil
	}
	return core.DetailsSection("Permissions Boundary",
		core.TableRow{"Type", string(boundary.PermissionsBoundaryType)},
		core.TableRow{"Policy ARN", aws.ToString(boundary.PermissionsBoundaryArn)},
	)
}

func roleDetailsRows(role *types.Role) []core.TableRow {
	var rows = []core.TableRow{
		{"Name", aws.ToString(role.RoleName)},
		{"ARN", aws.ToString(role.Arn)},
		{"Role ID", aws.ToString(role.RoleId)},
		{"Path", aws.ToString(role.Path)},
		{"Description", aws.ToString(role.Description)},
		{"Created", core.FormatTime(role.CreateDate)},
		{"Max Session Duration (s)", core.FormatOptionalInt(role.MaxSessionDuration)},
	}

	if lastUsed := role.RoleLastUsed; lastUsed != nil {
		rows = append(rows, core.DetailsSection("Last Used",
			core.TableRow{"Date", core.FormatTime(lastUsed.LastUsedDate)},
			core.TableRow{"Region", aws.ToString(lastUsed.Region)},
		)...)
	}

	return append(rows, permissionsBoundaryRows(role.PermissionsBoundary)...)
}

func NewRoleDetailsTable(serviceCtx *core.ServiceContext[awsapi.IamApi]) *IamDetailsTable[types.Role] {
	return NewIamDetailsTable(serviceCtx, "Role Details",
		func(ctx context.Context, key string) (*types.Role, error) {
			return serviceCtx.Api.GetRole(ctx, key)
		},
		roleDetailsRows,
		func(role *types.Role) string { return aws.ToString(role.RoleName) },
		func(ctx context.Context, key string) ([]types.Tag, error) {
			return serviceCtx.Api.ListRoleTags(ctx, key)
		},
	)
}

type IamRolesPageView struct {
	*core.ServicePageView
	RoleListTable    *IamListTable[types.Role]
	RoleDetailsTable *IamDetailsTable[types.Role]
	TrustPolicyView  *core.SearchableTextView
	PoliciesView     *PrincipalPoliciesView
	TabView          *core.TabViewHorizontal
	serviceCtx       *core.ServiceContext[awsapi.IamApi]
}

func NewIamRolesPageView(
	roleListTable *IamListTable[types.Role],
	roleDetailsTable *IamDetailsTable[types.Role],
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *IamRolesPageView {
	var trustPolicyView = core.NewSearchableTextView("Trust Policy", serviceCtx.AppContext)
	trustPolicyView.SetJsonHighlighting(true)

	// The trust policy comes with the role, so it is shown as the details load
	roleDetailsTable.AddLoadedFunc(func(role *types.Role) {
		var document = ""
		if role != nil {
			document, _ = utils.TryFormatToJson(aws.ToString(role.AssumeRolePolicyDocument))
		}
		trustPolicyView.SetText(document, false)
	})

	var policiesView = NewPrincipalPoliciesView(serviceCtx,
		func(ctx context.Context, key string) ([]awsapi.PrincipalPolicy, error) {
			return serviceCtx.Api.ListRolePolicies(ctx, key)
		},
		func(ctx context.Context, principal string, policy awsapi.PrincipalPolicy) (string, error) {
			return serviceCtx.Api.GetRolePolicyDocument(ctx, principal, policy)
		},
	)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(IamTabNameDetails, roleDetailsTable, 0, 1, true).
		AddTab(IamTabNameTrustPolicy, trustPolicyView, 0, 1, true).
		AddTab(IamTabNamePolicies, policiesView, 0, 1, true)

	var view = &IamRolesPageView{
		ServicePageView: newIamPageView(serviceCtx.AppContext, tabView, roleListTable,
			&roleListTable.ErrorMessageCallback,
			&roleDetailsTable.ErrorMessageCallback,
			&trustPolicyView.ErrorMessageCallback,
		),
		RoleListTable:    roleListTable,
		RoleDetailsTable: roleDetailsTable,
		TrustPolicyView:  trustPolicyView,
		PoliciesView:     policiesView,
		TabView:          tabView,
		serviceCtx:       serviceCtx,
	}
	policiesView.SetErrorMessageCallback(roleListTable.ErrorMessageCallback)

	roleListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		var roleName = roleListTable.GetSelectedKey()
		if roleName == "" {
			return
		}
		roleDetailsTable.RefreshDetails(roleName)
		policiesView.Refresh(roleName)
	})

	return view
}
