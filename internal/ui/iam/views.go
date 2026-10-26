package iam

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
)

const (
	rolesPageIdx = iota
	usersPageIdx
	policiesPageIdx
)

// principalNameFromArn is the name in a role or user ARN, whose resource is
// "<type>/<path>/<name>" with a path of one or more parts.
func principalNameFromArn(resource arn.ARN, resourceType string) (string, bool) {
	var path, found = strings.CutPrefix(resource.Resource, resourceType+"/")
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
		usersView = NewIamUsersPageView(
			NewUserListTable(serviceCtx),
			NewUserDetailsTable(serviceCtx),
			serviceCtx,
		)
		policiesView = NewIamPoliciesPageView(
			NewPolicyListTable(serviceCtx),
			NewPolicyDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.IAM), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Roles", rolesView, true).
		AddPage("Users", usersView, true, true).
		AddPage("Policies", policiesView, true, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("iam", func(resource arn.ARN) {
		if name, ok := principalNameFromArn(resource, "role"); ok {
			serviceRootView.ChangePage(rolesPageIdx, nil)
			rolesView.RoleListTable.Open(name)
		} else if name, ok := principalNameFromArn(resource, "user"); ok {
			serviceRootView.ChangePage(usersPageIdx, nil)
			usersView.UserListTable.Open(name)
		} else if strings.HasPrefix(resource.Resource, "policy/") {
			// Policies are known by their ARN
			serviceRootView.ChangePage(policiesPageIdx, nil)
			policiesView.PolicyListTable.Open(resource.String())
		}
	})

	return serviceRootView
}
