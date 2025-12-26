package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SfnDetailsPageView struct {
	*core.ServicePageView
	selectedStateMachine string
	sfnListTable         *tables.SfnListTable
	sfnExecutionsTable   *tables.SfnExecutionsTable
	sfnDetailsTable      *tables.SfnDetailsTable
	serviceCtx           *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnDetailsPageView(
	sfnListTable *tables.SfnListTable,
	sfnExecutionsTable *tables.SfnExecutionsTable,
	stateMachineDetailsTable *tables.SfnDetailsTable,
	serviceViewCtx *core.ServiceContext[awsapi.StateMachineApi],
) *SfnDetailsPageView {
	var tabView = core.NewTabView(serviceViewCtx.AppContext).
		AddAndSwitchToTab("Executions", sfnExecutionsTable, 0, 1, true).
		AddTab("Details", stateMachineDetailsTable, 0, 1, true)

	const detailsViewSize = 4000
	const tableViewSize = 6000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		sfnListTable, tableViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{sfnListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	sfnListTable.ErrorMessageCallback = errorHandler
	sfnExecutionsTable.ErrorMessageCallback = errorHandler
	stateMachineDetailsTable.ErrorMessageCallback = errorHandler

	var detailsView = &SfnDetailsPageView{
		ServicePageView:      serviceView,
		selectedStateMachine: "",
		sfnListTable:         sfnListTable,
		sfnExecutionsTable:   sfnExecutionsTable,
		sfnDetailsTable:      stateMachineDetailsTable,
		serviceCtx:           serviceViewCtx,
	}

	detailsView.initInputCapture()

	return detailsView
}

func (inst *SfnDetailsPageView) initInputCapture() {
	inst.sfnListTable.SetSelectedFunc(func(row, column int) {
		var smTable = inst.sfnListTable
		var smExeTable = inst.sfnExecutionsTable
		var smDetTable = inst.sfnDetailsTable

		var selectedFunc = smTable.GetSeletedFunction()
		smExeTable.SetSeletedFunction(selectedFunc)
		smDetTable.RefreshDetails(selectedFunc)

		switch smTable.GetSeletedFunctionType() {
		case types.StateMachineTypeStandard:
			smExeTable.RefreshExecutions(true)
		case types.StateMachineTypeExpress:
			var group = smDetTable.GetSelectedSmLogGroup()
			smExeTable.RefreshExpressExecutions(group, true)
		}
	})
}

type SfnExectionDetailsPageView struct {
	*core.ServicePageView
	selectedExection string
	summaryTable     *tables.SfnExecutionSummaryTable
	detailsTable     *tables.SfnExecutionStatesTable
	searchInput      *tview.InputField
	serviceCtx       *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnExectionDetailsPage(
	executionSummary *tables.SfnExecutionSummaryTable,
	executionStates *tables.SfnExecutionStatesTable,
	executionStateEvents *tables.SfnExecutionStateEventsTable,
	serviceViewCtx *core.ServiceContext[awsapi.StateMachineApi],
) *SfnExectionDetailsPageView {

	var inputOutputExpandedView = core.JsonTextView[string]{
		TextView: core.NewSearchableTextView("", serviceViewCtx.AppContext),
		ExtractTextFunc: func(data string) string {
			return data
		},
	}

	var stateSelectionFunc = func(row, _ int) {
		executionStateEvents.RefreshExecutionState(executionStates.GetSelectedState())
	}

	executionStates.SetSelectedFunc(stateSelectionFunc)
	executionStates.SetSelectionChangedFunc(stateSelectionFunc)

	var eventSelectionFunc = func(row, _ int) {
		if input := executionStateEvents.GetSelectedStepInput(); len(input) > 0 {
			inputOutputExpandedView.SetTitle("Input")
			inputOutputExpandedView.SetText(input)
		} else if output := executionStateEvents.GetSelectedStepOutput(); len(output) > 0 {
			inputOutputExpandedView.SetTitle("Ouput")
			inputOutputExpandedView.SetText(output)
		} else {
			inputOutputExpandedView.SetTitle("Errors")
			inputOutputExpandedView.SetText(executionStateEvents.GetSelectedStepErrorCause())
		}
	}

	executionStateEvents.SetSelectedFunc(eventSelectionFunc)
	executionStateEvents.SetSelectionChangedFunc(eventSelectionFunc)

	var tabView = core.NewTabView(serviceViewCtx.AppContext).
		AddAndSwitchToTab("Input/Output", inputOutputExpandedView.TextView, 0, 1, true).
		AddTab("Summary", executionSummary, 0, 1, true)

	const detailsViewSize = 10
	const inputOutputViewSize = 10

	var resizableStatesView = core.NewResizableView(
		executionStates, detailsViewSize,
		executionStateEvents, inputOutputViewSize,
		tview.FlexColumn,
	)

	var resizableView = core.NewResizableView(
		tabView, inputOutputViewSize,
		resizableStatesView, detailsViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.
		AddItem(resizableView, 0, 1, false)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{executionStates, executionStateEvents},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	executionSummary.ErrorMessageCallback = errorHandler
	executionStates.ErrorMessageCallback = errorHandler

	var detailsView = &SfnExectionDetailsPageView{
		ServicePageView:  serviceView,
		selectedExection: "",
		summaryTable:     executionSummary,
		detailsTable:     executionStates,
		serviceCtx:       serviceViewCtx,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *SfnExectionDetailsPageView) initInputCapture() {
}

func NewStepFunctionsHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xFF3399))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewStateMachineApi(*appCtx.Config, appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		cwlApi = awsapi.NewCloudWatchLogsApi(*appCtx.Config, appCtx.Logger)

		SfnDetailsView = NewSfnDetailsPageView(
			tables.NewSfnListTable(serviceCtx),
			tables.NewSfnExecutionsTable(serviceCtx.AppContext, api, cwlApi),
			tables.NewSfnDetailsTable(serviceCtx),
			serviceCtx,
		)

		SfnExeDetailsView = NewSfnExectionDetailsPage(
			tables.NewSfnExecutionSummaryTable(serviceCtx),
			tables.NewSfnExecutionDetailsTable(serviceCtx.AppContext, api, cwlApi),
			tables.NewSfnExecutionStatesTable(serviceCtx.AppContext, api),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(STATE_MACHINES), appCtx)

	serviceRootView.
		AddAndSwitchToPage("StateMachines", SfnDetailsView, true).
		AddPage("Exection Details", SfnExeDetailsView, true, true)

	serviceRootView.InitPageNavigation()

	SfnDetailsView.sfnExecutionsTable.SetSelectedFunc(func(row, column int) {
		var selectedExecution = SfnDetailsView.
			sfnExecutionsTable.GetSeletedExecutionArn()
		var sfType = SfnDetailsView.sfnListTable.GetSeletedFunctionType()

		if len(selectedExecution) > 0 {
			if sfType == "EXPRESS" {
				var execution = SfnDetailsView.
					sfnExecutionsTable.GetSeletedExecution()
				SfnExeDetailsView.detailsTable.RefreshExpressExecutionDetails(execution, true)
			} else {
				SfnExeDetailsView.summaryTable.RefreshExecutionDetails(selectedExecution, true)
				SfnExeDetailsView.detailsTable.RefreshExecutionDetails(selectedExecution, true)
			}
			serviceRootView.ChangePage(1, nil)
		}
	})

	SfnDetailsView.initInputCapture()
	SfnExeDetailsView.initInputCapture()

	return serviceRootView
}
