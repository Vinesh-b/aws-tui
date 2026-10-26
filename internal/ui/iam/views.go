package iam

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type IamTabName = string

const (
	IamTabNameDetails     IamTabName = "Details"
	IamTabNameTrustPolicy IamTabName = "Trust Policy"
	IamTabNamePolicies    IamTabName = "Policies"
	IamTabNameTags        IamTabName = "Tags"
)

type IamRolesPageView struct {
	*core.ServicePageView
	RoleListTable    *RoleListTable
	RoleDetailsTable *RoleDetailsTable
	TrustPolicyView  *core.SearchableTextView
	PoliciesView     *RolePoliciesView
	RoleTagsTable    *core.TagsTable[types.Tag, awsapi.IamApi]
	TabView          *core.TabViewHorizontal
	serviceCtx       *core.ServiceContext[awsapi.IamApi]
}

func NewIamRolesPageView(
	roleListTable *RoleListTable,
	roleDetailsTable *RoleDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *IamRolesPageView {
	var tagsTable = core.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() func() ([]types.Tag, error) {
			var roleName = roleListTable.GetSelectedRole()
			return func() ([]types.Tag, error) {
				return serviceCtx.Api.ListRoleTags(serviceCtx.Ctx, roleName)
			}
		})

	var trustPolicyView = core.NewSearchableTextView("Trust Policy", serviceCtx.AppContext)
	trustPolicyView.SetJsonHighlighting(true)

	// The trust policy comes with the role, so it is shown as the details load
	roleDetailsTable.AddRoleLoadedFunc(func(role *types.Role) {
		var document = ""
		if role != nil {
			document, _ = utils.TryFormatToJson(aws.ToString(role.AssumeRolePolicyDocument))
		}
		trustPolicyView.SetText(document, false)
	})

	var policiesView = NewRolePoliciesView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(IamTabNameDetails, roleDetailsTable, 0, 1, true).
		AddTab(IamTabNameTrustPolicy, trustPolicyView, 0, 1, true).
		AddTab(IamTabNamePolicies, policiesView, 0, 1, true).
		AddTab(IamTabNameTags, tagsTable, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		roleListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &IamRolesPageView{
		ServicePageView:  serviceView,
		RoleListTable:    roleListTable,
		RoleDetailsTable: roleDetailsTable,
		TrustPolicyView:  trustPolicyView,
		PoliciesView:     policiesView,
		RoleTagsTable:    tagsTable,
		TabView:          tabView,
		serviceCtx:       serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	roleListTable.ErrorMessageCallback = errorHandler
	roleDetailsTable.ErrorMessageCallback = errorHandler
	trustPolicyView.ErrorMessageCallback = errorHandler
	policiesView.PoliciesTable.ErrorMessageCallback = errorHandler
	tagsTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{roleListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *IamRolesPageView) initInputCapture() {
	inst.RoleListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		var roleName = inst.RoleListTable.GetSelectedRole()
		if roleName == "" {
			return
		}
		inst.RoleDetailsTable.RefreshDetails(roleName)
		inst.PoliciesView.RefreshPolicies(roleName)
		inst.RoleTagsTable.RefreshDetails()
	})
}

// roleNameFromArn is the name in a role ARN, whose resource is
// "role/<path>/<name>" with a path of one or more parts.
func roleNameFromArn(resource arn.ARN) (string, bool) {
	var path, found = strings.CutPrefix(resource.Resource, "role/")
	if !found {
		return "", false
	}
	var name = path[strings.LastIndex(path, "/")+1:]
	return name, len(name) > 0
}

func NewIamHomeView(appCtx *core.AppContext, api awsapi.IamApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xBB2244))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		rolesView = NewIamRolesPageView(
			NewRoleListTable(serviceCtx),
			NewRoleDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.IAM), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Roles", rolesView, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("iam", func(resource arn.ARN) {
		if name, ok := roleNameFromArn(resource); ok {
			serviceRootView.ChangePage(0, nil)
			rolesView.RoleListTable.OpenRole(name)
		}
	})

	return serviceRootView
}
