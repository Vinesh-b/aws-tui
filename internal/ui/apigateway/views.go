package apigateway

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ApiGatewayTabName = string

const (
	ApiGatewayTabNameDetails ApiGatewayTabName = "Details"
)

type ApiGatewayApisPageView struct {
	*core.ServicePageView
	ApisTable    *ApiGatewayApisTable
	DetailsTable *ApiGatewayDetailsTable
	TabView      *core.TabViewHorizontal
	serviceCtx   *core.ServiceContext[awsapi.ApiGatewayApi]
}

func NewApiGatewayApisPageView(
	apisTable *ApiGatewayApisTable,
	detailsTable *ApiGatewayDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.ApiGatewayApi],
) *ApiGatewayApisPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(ApiGatewayTabNameDetails, detailsTable, 0, 1, true)

	const tabViewSize = 3000
	const apisTableSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		apisTable, apisTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &ApiGatewayApisPageView{
		ServicePageView: serviceView,
		ApisTable:       apisTable,
		DetailsTable:    detailsTable,
		TabView:         tabView,
		serviceCtx:      serviceCtx,
	}

	apisTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{apisTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *ApiGatewayApisPageView) initInputCapture() {
	inst.ApisTable.SetSelectionChangedFunc(func(row, column int) {
		inst.DetailsTable.SetApi(inst.ApisTable.GetSelectedApi())
	})
}

func NewApiGatewayHomeView(appCtx *core.AppContext, api awsapi.ApiGatewayApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xFF4F8B))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		apisView = NewApiGatewayApisPageView(
			NewApiGatewayApisTable(serviceCtx),
			NewApiGatewayDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.API_GATEWAY), appCtx)

	serviceRootView.
		AddAndSwitchToPage("REST APIs", apisView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
