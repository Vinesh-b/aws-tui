package eventbridge

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type EventBridgeTabName = string

const (
	EventBridgeTabNameDetails EventBridgeTabName = "Details"
	EventBridgeTabNamePolicy  EventBridgeTabName = "Policy"
	EventBridgeTabNameRules   EventBridgeTabName = "Rules"
	EventBridgeTabNameSend    EventBridgeTabName = "Send Event"
)

type EventBridgeDetailsPageView struct {
	*core.ServicePageView
	EventBusListTable    *EventBusListTable
	EventBusDetailsTable *EventBusDetailsTable
	RulesTabView         *RulesTabView
	EventBusPolicyView   *core.JsonTextView[types.EventBus]
	SendEventView        *SendEventPageView
	TabView              *core.TabViewHorizontal
	serviceCtx           *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewEventBridgeDetailsPageView(
	busListTable *EventBusListTable,
	busDetailsTable *EventBusDetailsTable,
	rulesTabView *RulesTabView,
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *EventBridgeDetailsPageView {
	var policyView = core.JsonTextView[types.EventBus]{
		TextView: core.NewSearchableTextView("", serviceCtx.AppContext),
		ExtractTextFunc: func(data types.EventBus) string {
			return aws.ToString(data.Policy)
		},
	}
	policyView.SetTitle("Policy")

	var sendEventView = NewSendEventPageView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(EventBridgeTabNameDetails, busDetailsTable, 0, 1, true).
		AddTab(EventBridgeTabNamePolicy, policyView.TextView, 0, 1, true).
		AddTab(EventBridgeTabNameRules, rulesTabView, 0, 1, true).
		AddTab(EventBridgeTabNameSend, sendEventView, 0, 1, true)

	tabView.GetTab(EventBridgeTabNameRules).InitViewNavigation(
		[][]core.View{
			{rulesTabView.RulesTable, rulesTabView.JsonView.TextView},
		},
	)

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

		EventBusListTable:    busListTable,
		EventBusDetailsTable: busDetailsTable,
		RulesTabView:         rulesTabView,
		EventBusPolicyView:   &policyView,
		SendEventView:        sendEventView,
		TabView:              tabView,
		serviceCtx:           serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	busDetailsTable.ErrorMessageCallback = errorHandler
	busListTable.ErrorMessageCallback = errorHandler
	rulesTabView.RulesTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{busListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *EventBridgeDetailsPageView) initInputCapture() {
	inst.EventBusListTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedEventBus = inst.EventBusListTable.GetSelectedEventBus()
		inst.EventBusDetailsTable.RefreshDetails(selectedEventBus)
		inst.RulesTabView.RefreshRules(aws.ToString(selectedEventBus.Arn))

		inst.EventBusPolicyView.SetText(inst.EventBusListTable.GetSelectedEventBus())
		inst.SendEventView.SetSelectedBus(
			aws.ToString(selectedEventBus.Name), aws.ToString(selectedEventBus.Arn),
		)
	})
}

func NewEventBridgeHomeView(appCtx *core.AppContext, api awsapi.EventBridgeApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		eventbridgeDetailsView = NewEventBridgeDetailsPageView(
			NewEventBusListTable(serviceCtx),
			NewEventBusDetailsTable(serviceCtx),
			NewRulesTabView(NewRulesTable(serviceCtx), serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.EVENTBRIDGE), appCtx)

	serviceRootView.
		AddAndSwitchToPage("EventBuses", eventbridgeDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
