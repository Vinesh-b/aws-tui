package cloudformation

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CloudFormationDetailsPageView struct {
	*core.ServicePageView
	stackListTable    *StackListTable
	stackDetailsTable *StackDetailsTable
	serviceCtx        *core.ServiceContext[awsapi.CloudFormationApi]
}

func NewStacksDetailsPageView(
	stackListTable *StackListTable,
	stackDetailsTable *StackDetailsTable,
	serviceContext *core.ServiceContext[awsapi.CloudFormationApi],
) *CloudFormationDetailsPageView {
	const stackDetailsSize = 5000
	const stackTablesSize = 3000

	var mainPage = core.NewResizableView(
		stackDetailsTable, stackDetailsSize,
		stackListTable, stackTablesSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceContext.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{stackDetailsTable},
			{stackListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	stackListTable.ErrorMessageCallback = errorHandler
	stackDetailsTable.ErrorMessageCallback = errorHandler

	return &CloudFormationDetailsPageView{
		ServicePageView:   serviceView,
		stackListTable:    stackListTable,
		stackDetailsTable: stackDetailsTable,
		serviceCtx:        serviceContext,
	}
}

func (inst *CloudFormationDetailsPageView) InitInputCapture() {
	inst.stackListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.stackDetailsTable.RefreshDetails(inst.stackListTable.GetSelectedStack())
	})
}

type CloudFormationStackEventsPageView struct {
	*core.ServicePageView
	stackEventsTable *StackEventsTable
	selectedStack    string
	searchPositions  []int
	serviceCtx       *core.ServiceContext[awsapi.CloudFormationApi]
}

func NewStackEventsPageView(
	stackEventsTable *StackEventsTable,
	serviceContext *core.ServiceContext[awsapi.CloudFormationApi],
) *CloudFormationStackEventsPageView {
	var expandedMsgView = tview.NewTextArea()
	expandedMsgView.
		SetBorder(true).
		SetTitle("Message").
		SetTitleAlign(tview.AlignLeft)

	stackEventsTable.SetSelectionChangedFunc(func(row, column int) {
		var logText = stackEventsTable.GetResourceStatusReason(row)
		expandedMsgView.SetText(logText, false)
	})

	const expandedMsgSize = 5
	const stackEventsSize = 15

	var mainPage = core.NewResizableView(
		expandedMsgView, expandedMsgSize,
		stackEventsTable, stackEventsSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceContext.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{expandedMsgView},
			{stackEventsTable},
		},
	)

	stackEventsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &CloudFormationStackEventsPageView{
		ServicePageView:  serviceView,
		stackEventsTable: stackEventsTable,
		selectedStack:    "",
		serviceCtx:       serviceContext,
	}
}

func (inst *CloudFormationStackEventsPageView) InitInputCapture() {
	inst.stackEventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.stackEventsTable.RefreshEvents(true)
			return nil
		case core.APP_KEY_BINDINGS.LoadMoreData:
			inst.stackEventsTable.RefreshEvents(false)
			return nil
		}
		return event
	})
}

func NewStacksHomeView(appCtx *core.AppContext, api awsapi.CloudFormationApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		stacksDetailsView = NewStacksDetailsPageView(
			NewStackListTable(serviceCtx),
			NewStackDetailsTable(serviceCtx),
			serviceCtx,
		)
		stackEventsView = NewStackEventsPageView(
			NewStackEventsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDFORMATION), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Stacks", stacksDetailsView, true).
		AddPage("Events", stackEventsView, true, true)

	serviceRootView.InitPageNavigation()

	stacksDetailsView.stackListTable.SetSelectedFunc(func(row, column int) {
		var selectedStackName = stacksDetailsView.stackListTable.GetSelectedStackName()
		if len(selectedStackName) > 0 {
			stackEventsView.stackEventsTable.SetSelectedStackName(selectedStackName)
			stackEventsView.stackEventsTable.SetTitleExtra(selectedStackName)
			stackEventsView.stackEventsTable.RefreshEvents(true)
			serviceRootView.ChangePage(1, nil)
		}
	})

	stackEventsView.InitInputCapture()
	stacksDetailsView.InitInputCapture()

	return serviceRootView
}
