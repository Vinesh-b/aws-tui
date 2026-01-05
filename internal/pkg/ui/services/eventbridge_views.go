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
	EventBusListTable    *tables.EventBusListTable
	EventBusDetailsTable *tables.EventBusDetailsTable
	EventBusTagsTable    *tables.TagsTable[types.Tag, awsapi.EventBridgeApi]
	EventBusPolicyView   *core.JsonTextView[types.EventBus]
	serviceCtx           *core.ServiceContext[awsapi.EventBridgeApi]
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

	var busTagsTable = tables.NewTagsTable(serviceCtx,
		func(t types.Tag) (string, string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		},
		func() ([]types.Tag, error) {
			return serviceCtx.Api.ListTags(
				true, aws.ToString(busListTable.GetSeletedEventBus().Arn),
			)
		},
	)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab("Details", busDetailsTable, 0, 1, true).
		AddTab("Policy", policyView.TextView, 0, 1, true).
		AddTab("Tags", busTagsTable, 0, 1, true)

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
		EventBusTagsTable:    busTagsTable,
		EventBusPolicyView:   &policyView,
		serviceCtx:           serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	busDetailsTable.ErrorMessageCallback = errorHandler
	busListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabDisplayView()},
			{busListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *EventBridgeDetailsPageView) initInputCapture() {
	inst.EventBusListTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedEventBus = inst.EventBusListTable.GetSeletedEventBus()
		inst.EventBusDetailsTable.RefreshDetails(selectedEventBus)
		inst.EventBusTagsTable.RefreshDetails()

		inst.EventBusPolicyView.SetText(inst.EventBusListTable.GetSeletedEventBus())
	})
}

func NewEventBridgeHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewEventBridgeApi(appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		eventbridgeDetailsView = NewEventBridgeDetailsPageView(
			tables.NewEventBusListTable(serviceCtx),
			tables.NewEventBusDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(LAMBDA), appCtx)

	serviceRootView.
		AddAndSwitchToPage("EventBuses", eventbridgeDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
