package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type EventBridgeDetailsPageView struct {
	*core.ServicePageView
	EventBridgeListTable    *tables.EventBusListTable
	EventBridgeDetailsTable *tables.EventBusDetailsTable
	PolicyView              *core.JsonTextView[types.EventBus]
	serviceCtx              *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewEventBridgeDetailsPageView(
	busListTable *tables.EventBusListTable,
	busDetailsTable *tables.EventBusDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *EventBridgeDetailsPageView {
	var policyView = core.JsonTextView[types.EventBus]{
		TextView: core.NewSearchableTextView("", serviceCtx.AppContext),
		ExtractTextFunc: func(data types.EventBus) string {
			return aws.ToString(data.Policy)
		},
	}
	policyView.SetTitle("Policy")

	var tabView = core.NewTabView(serviceCtx.AppContext).
		AddAndSwitchToTab("Details", busDetailsTable, 0, 1, true).
		AddTab("Policy", policyView.TextView, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		busListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &EventBridgeDetailsPageView{
		ServicePageView: serviceView,

		EventBridgeListTable:    busListTable,
		EventBridgeDetailsTable: busDetailsTable,
		PolicyView:              &policyView,
		serviceCtx:              serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	busDetailsTable.ErrorMessageCallback = errorHandler
	busListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{busListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *EventBridgeDetailsPageView) initInputCapture() {
	inst.EventBridgeListTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedEventBus = inst.EventBridgeListTable.GetSeletedEventBus()
		inst.EventBridgeDetailsTable.RefreshDetails(selectedEventBus)

		inst.PolicyView.SetText(inst.EventBridgeListTable.GetSeletedEventBus())
	})
}

func NewEventBridgeHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api       = awsapi.NewEventBridgeApi(*appCtx.Config, appCtx.Logger)
		lambdaCtx = core.NewServiceViewContext(appCtx, api)

		eventbridgeDetailsView = NewEventBridgeDetailsPageView(
			tables.NewEventBusListTable(lambdaCtx),
			tables.NewEventBusDetailsTable(lambdaCtx),
			lambdaCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(LAMBDA), appCtx)

	serviceRootView.
		AddAndSwitchToPage("EventBuses", eventbridgeDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
