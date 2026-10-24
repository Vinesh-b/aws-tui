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
	EventBridgeTabNameTags    EventBridgeTabName = "Tags"
)

type EventBridgeDetailsPageView struct {
	*core.ServicePageView
	EventBusListTable    *EventBusListTable
	EventBusDetailsTable *EventBusDetailsTable
	RulesTabView         *RulesTabView
	EventBusTagsTable    *core.TagsTable[types.Tag, awsapi.EventBridgeApi]
	EventBusPolicyView   *core.JsonTextView[types.EventBus]
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

	var tagsTable = core.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() func() ([]types.Tag, error) {
			var arn = aws.ToString(busListTable.GetSelectedEventBus().Arn)
			return func() ([]types.Tag, error) {
				return serviceCtx.Api.ListTags(serviceCtx.Ctx, true, arn)
			}
		})

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(EventBridgeTabNameDetails, busDetailsTable, 0, 1, true).
		AddTab(EventBridgeTabNamePolicy, policyView.TextView, 0, 1, true).
		AddTab(EventBridgeTabNameRules, rulesTabView, 0, 1, true).
		AddTab(EventBridgeTabNameTags, tagsTable, 0, 1, true)

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
		EventBusTagsTable:    tagsTable,
		EventBusPolicyView:   &policyView,
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
			{tabView.GetTabDisplayView()},
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
		inst.EventBusTagsTable.RefreshDetails()
		inst.RulesTabView.RefreshRules(aws.ToString(selectedEventBus.Arn))

		inst.EventBusPolicyView.SetText(inst.EventBusListTable.GetSelectedEventBus())
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
