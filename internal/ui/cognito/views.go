package cognito

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newTabsPage is the tabs over the list, as the pages of WAF have them
func newTabsPage(
	tabView *core.TabViewHorizontal, list core.View, appCtx *core.AppContext,
) *core.ServicePageView {
	const tabViewSize = 5000
	const listSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		list, listSize,
		tview.FlexRow,
		appCtx,
	)
	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{list},
		},
	)
	return serviceView
}

const (
	UserPoolTabNameDetails    = "Details"
	UserPoolTabNameUsers      = "Users"
	UserPoolTabNameAppClients = "App Clients"
)

// UserPoolsPageView is the user pools, with the details, users and app
// clients of the one selected in tabs over them
type UserPoolsPageView struct {
	*core.ServicePageView
	UserPoolsTable  *UserPoolsTable
	DetailsTable    *UserPoolDetailsTable
	UsersTable      *UsersTable
	AppClientsTable *AppClientsTable
	TabView         *core.TabViewHorizontal
}

func NewUserPoolsPageView(
	userPoolsTable *UserPoolsTable,
	detailsTable *UserPoolDetailsTable,
	usersTable *UsersTable,
	appClientsTable *AppClientsTable,
	serviceCtx *core.ServiceContext[awsapi.CognitoApi],
) *UserPoolsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(UserPoolTabNameDetails, detailsTable, 0, 1, true).
		AddTab(UserPoolTabNameUsers, usersTable, 0, 1, true).
		AddTab(UserPoolTabNameAppClients, appClientsTable, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		usersTable.SetVisible(tabName == UserPoolTabNameUsers)
		appClientsTable.SetVisible(tabName == UserPoolTabNameAppClients)
	})

	var serviceView = newTabsPage(tabView, userPoolsTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	userPoolsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	usersTable.ErrorMessageCallback = errorHandler
	appClientsTable.ErrorMessageCallback = errorHandler

	// What the list has is shown at once, but the rest is only loaded for the
	// user pool the selection stops on
	userPoolsTable.SetSelectionChangedFunc(func(row, column int) {
		var pool = userPoolsTable.GetSelectedUserPool()
		detailsTable.SetUserPool(pool)
		usersTable.SetUserPool(pool)
		appClientsTable.SetUserPool(pool)
	})
	userPoolsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
		usersTable.Load()
		appClientsTable.Load()
	})

	return &UserPoolsPageView{
		ServicePageView: serviceView,
		UserPoolsTable:  userPoolsTable,
		DetailsTable:    detailsTable,
		UsersTable:      usersTable,
		AppClientsTable: appClientsTable,
		TabView:         tabView,
	}
}

const (
	IdentityPoolTabNameDetails = "Details"
	IdentityPoolTabNameRoles   = "Roles"
)

// IdentityPoolsPageView is the identity pools, with the details and roles of
// the one selected in tabs over them
type IdentityPoolsPageView struct {
	*core.ServicePageView
	IdentityPoolsTable *IdentityPoolsTable
	DetailsTable       *IdentityPoolDetailsTable
	RolesTable         *IdentityPoolRolesTable
	TabView            *core.TabViewHorizontal
}

func NewIdentityPoolsPageView(
	identityPoolsTable *IdentityPoolsTable,
	detailsTable *IdentityPoolDetailsTable,
	rolesTable *IdentityPoolRolesTable,
	serviceCtx *core.ServiceContext[awsapi.CognitoApi],
) *IdentityPoolsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(IdentityPoolTabNameDetails, detailsTable, 0, 1, true).
		AddTab(IdentityPoolTabNameRoles, rolesTable, 0, 1, true)

	var serviceView = newTabsPage(tabView, identityPoolsTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	identityPoolsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	rolesTable.ErrorMessageCallback = errorHandler

	identityPoolsTable.SetSelectionChangedFunc(func(row, column int) {
		var pool = identityPoolsTable.GetSelectedIdentityPool()
		detailsTable.SetIdentityPool(pool)
		rolesTable.SetIdentityPool(pool)
	})
	identityPoolsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
		rolesTable.RefreshRoles()
	})

	return &IdentityPoolsPageView{
		ServicePageView:    serviceView,
		IdentityPoolsTable: identityPoolsTable,
		DetailsTable:       detailsTable,
		RolesTable:         rolesTable,
		TabView:            tabView,
	}
}

// The pages of the service
const (
	cognitoPageUserPools = iota
	cognitoPageIdentityPools
)

func isOpenableUserPool(resource arn.ARN) bool {
	var named, err = awsapi.ParseCognitoArn(resource.String())
	return err == nil && named.Service == "cognito-idp" && named.Kind == "userpool"
}

func isOpenableIdentityPool(resource arn.ARN) bool {
	var named, err = awsapi.ParseCognitoArn(resource.String())
	return err == nil && named.Service == "cognito-identity" && named.Kind == "identitypool"
}

func NewCognitoHomeView(appCtx *core.AppContext, api awsapi.CognitoApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xA6194A))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		userPoolsView = NewUserPoolsPageView(
			NewUserPoolsTable(serviceCtx),
			NewUserPoolDetailsTable(serviceCtx),
			NewUsersTable(serviceCtx),
			NewAppClientsTable(serviceCtx),
			serviceCtx,
		)
		identityPoolsView = NewIdentityPoolsPageView(
			NewIdentityPoolsTable(serviceCtx),
			NewIdentityPoolDetailsTable(serviceCtx),
			NewIdentityPoolRolesTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.COGNITO), appCtx)

	serviceRootView.
		AddAndSwitchToPage("User Pools", userPoolsView, true).
		AddPage("Identity Pools", identityPoolsView, true, true)

	serviceRootView.InitPageNavigation()

	// Each kind of pool has the ARNs of its own service
	serviceRootView.SetResourceOpener("cognito-idp", isOpenableUserPool, func(resource arn.ARN) {
		var named, _ = awsapi.ParseCognitoArn(resource.String())
		serviceRootView.ChangePage(cognitoPageUserPools, nil)
		userPoolsView.UserPoolsTable.Open(named.Id)
	})
	serviceRootView.SetResourceOpener("cognito-identity", isOpenableIdentityPool, func(resource arn.ARN) {
		var named, _ = awsapi.ParseCognitoArn(resource.String())
		serviceRootView.ChangePage(cognitoPageIdentityPools, nil)
		identityPoolsView.IdentityPoolsTable.Open(named.Id)
	})

	return serviceRootView
}
