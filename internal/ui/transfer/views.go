package transfer

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	ServerTabNameDetails = "Details"
	ServerTabNameUsers   = "Users"
)

// ServersPageView is the servers, with the details and users of the one
// selected in tabs over them
type ServersPageView struct {
	*core.ServicePageView
	ServersTable *ServersTable
	DetailsTable *ServerDetailsTable
	UsersTable   *ServerUsersTable
	TabView      *core.TabViewHorizontal
}

func NewServersPageView(
	serversTable *ServersTable,
	detailsTable *ServerDetailsTable,
	usersTable *ServerUsersTable,
	serviceCtx *core.ServiceContext[awsapi.TransferApi],
) *ServersPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(ServerTabNameDetails, detailsTable, 0, 1, true).
		AddTab(ServerTabNameUsers, usersTable, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		usersTable.SetVisible(tabName == ServerTabNameUsers)
	})

	var serviceView = newTabsPage(tabView, serversTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	serversTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	usersTable.ErrorMessageCallback = errorHandler

	// What the list has is shown at once, but the rest is only loaded for the
	// server the selection stops on
	serversTable.SetSelectionChangedFunc(func(row, column int) {
		var server = serversTable.GetSelectedServer()
		detailsTable.SetServer(server)
		usersTable.SetServer(aws.ToString(server.ServerId))
	})
	serversTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
		usersTable.Load()
	})

	return &ServersPageView{
		ServicePageView: serviceView,
		ServersTable:    serversTable,
		DetailsTable:    detailsTable,
		UsersTable:      usersTable,
		TabView:         tabView,
	}
}

const ConnectorTabNameDetails = "Details"

// ConnectorsPageView is the connectors, with the details of the one selected
// in a tab over them
type ConnectorsPageView struct {
	*core.ServicePageView
	ConnectorsTable *ConnectorsTable
	DetailsTable    *ConnectorDetailsTable
	TabView         *core.TabViewHorizontal
}

func NewConnectorsPageView(
	connectorsTable *ConnectorsTable,
	detailsTable *ConnectorDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.TransferApi],
) *ConnectorsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(ConnectorTabNameDetails, detailsTable, 0, 1, true)

	var serviceView = newTabsPage(tabView, connectorsTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	connectorsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	connectorsTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetConnector(connectorsTable.GetSelectedConnector())
	})
	connectorsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
	})

	return &ConnectorsPageView{
		ServicePageView: serviceView,
		ConnectorsTable: connectorsTable,
		DetailsTable:    detailsTable,
		TabView:         tabView,
	}
}

// The pages of the service
const (
	transferPageServers = iota
	transferPageConnectors
)

func NewTransferHomeView(appCtx *core.AppContext, api awsapi.TransferApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x3F8624))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		serversView = NewServersPageView(
			NewServersTable(serviceCtx),
			NewServerDetailsTable(serviceCtx),
			NewServerUsersTable(serviceCtx),
			serviceCtx,
		)
		connectorsView = NewConnectorsPageView(
			NewConnectorsTable(serviceCtx),
			NewConnectorDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.TRANSFER_FAMILY), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Servers", serversView, true).
		AddPage("Connectors", connectorsView, true, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("transfer", isOpenableTransferResource, func(resource arn.ARN) {
		switch transferArnKind(resource) {
		case "server":
			serviceRootView.ChangePage(transferPageServers, nil)
			serversView.ServersTable.Open(resource.String())
		case "connector":
			serviceRootView.ChangePage(transferPageConnectors, nil)
			connectorsView.ConnectorsTable.Open(resource.String())
		}
	})

	return serviceRootView
}
