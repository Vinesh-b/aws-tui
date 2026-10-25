package sfn

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SfnTabName = string

const (
	SfnTabNameDetails          SfnTabName = "Details"
	SfnTabNameDefinition       SfnTabName = "Definition"
	SfnTabNameStates           SfnTabName = "States"
	SfnTabNameExecutions       SfnTabName = "Executions"
	SfnTabNameStartExecution   SfnTabName = "Start Execution"
	SfnTabNameExecutionSummary SfnTabName = "Summary"
	SfnTabNameStateIO          SfnTabName = "Input/Output"
	SfnTabNameStateFlow        SfnTabName = "State Flow"
)

type SfnDetailsPageView struct {
	*core.ServicePageView
	selectedStateMachine string
	sfnListTable         *SfnListTable
	sfnExecutionsTable   *SfnExecutionsTable
	sfnDetailsTable      *SfnDetailsTable
	startExecutionView   *SfnStartExecutionPageView
	tabView              *core.TabViewHorizontal
	serviceCtx           *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnDetailsPageView(
	sfnListTable *SfnListTable,
	sfnExecutionsTable *SfnExecutionsTable,
	stateMachineDetailsTable *SfnDetailsTable,
	serviceViewCtx *core.ServiceContext[awsapi.StateMachineApi],
) *SfnDetailsPageView {
	var definitionView = NewSfnDefinitionView(serviceViewCtx.AppContext)
	var statesView = NewSfnStatesTreeView()
	stateMachineDetailsTable.AddDetailsLoadedFunc(func(stateMachine *sfn.DescribeStateMachineOutput) {
		definitionView.SetDefinition(stateMachine)
		statesView.SetStateMachine(stateMachine)
	})

	var startExecutionView = NewSfnStartExecutionPageView(serviceViewCtx)

	var tabView = core.NewTabViewHorizontal(serviceViewCtx.AppContext).
		AddTab(SfnTabNameDetails, stateMachineDetailsTable, 0, 1, true).
		AddTab(SfnTabNameStates, statesView, 0, 1, true).
		AddTab(SfnTabNameDefinition, definitionView, 0, 1, true).
		AddAndSwitchToTab(SfnTabNameExecutions, sfnExecutionsTable, 0, 1, true).
		AddTab(SfnTabNameStartExecution, startExecutionView, 0, 1, true)

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
			{tabView},
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
		startExecutionView:   startExecutionView,
		tabView:              tabView,
		serviceCtx:           serviceViewCtx,
	}

	detailsView.initInputCapture()

	return detailsView
}

func (inst *SfnDetailsPageView) initInputCapture() {
	inst.sfnListTable.SetSelectedFunc(func(row, column int) {
		var selectedFunc = inst.sfnListTable.GetSelectedFunction()
		inst.sfnExecutionsTable.SetSelectedFunction(selectedFunc)
		inst.sfnDetailsTable.RefreshDetails(selectedFunc)
		inst.startExecutionView.SetSelectedStateMachine(
			inst.sfnListTable.GetSelectedFunctionName(),
			inst.sfnListTable.GetSelectedFunctionArn(),
		)
		inst.refreshExecutions()
	})

	inst.startExecutionView.SetOnExecutionStarted(inst.refreshExecutions)
}

func (inst *SfnDetailsPageView) refreshExecutions() {
	switch inst.sfnListTable.GetSelectedFunctionType() {
	case types.StateMachineTypeStandard:
		inst.sfnExecutionsTable.RefreshExecutions(true)
	case types.StateMachineTypeExpress:
		var group = inst.sfnDetailsTable.GetSelectedSmLogGroup()
		inst.sfnExecutionsTable.RefreshExpressExecutions(group, true)
	}
}

type SfnExecutionDetailsPageView struct {
	*core.ServicePageView
	selectedExecution string
	summaryTable      *SfnExecutionSummaryTable
	detailsTable      *SfnExecutionStatesTable
	searchInput       *tview.InputField
	stateFlowView     *SfnStatesTreeView
	serviceCtx        *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnExecutionDetailsPage(
	executionSummary *SfnExecutionSummaryTable,
	executionStates *SfnExecutionStatesTable,
	executionStateEvents *SfnExecutionStateEventsTable,
	serviceViewCtx *core.ServiceContext[awsapi.StateMachineApi],
) *SfnExecutionDetailsPageView {

	var inputOutputExpandedView = core.JsonTextView[string]{
		TextView: core.NewSearchableTextView("", serviceViewCtx.AppContext),
		ExtractTextFunc: func(data string) string {
			return data
		},
	}

	var stateFlowView = NewSfnStatesTreeView()
	stateFlowView.SetTitle("State Flow")
	executionStates.SetStatesLoadedFunc(stateFlowView.SetExecution)

	var stateSelectionFunc = func(_row, _col int) {
		var selectedState = executionStates.GetSelectedState()
		executionStateEvents.RefreshExecutionState(selectedState)
		stateFlowView.SelectState(selectedState.Name)
	}

	executionStates.SetSelectedFunc(stateSelectionFunc)
	executionStates.SetSelectionChangedFunc(stateSelectionFunc)

	var eventSelectionFunc = func(_row, _col int) {
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

	var tabView = core.NewTabViewHorizontal(serviceViewCtx.AppContext).
		AddTab(SfnTabNameExecutionSummary, executionSummary, 0, 1, true).
		AddTab(SfnTabNameStateFlow, stateFlowView, 0, 1, true).
		AddAndSwitchToTab(SfnTabNameStateIO, inputOutputExpandedView.TextView, 0, 1, true)

	const statesViewSize = 45
	const eventsViewSize = 55

	var resizableStatesView = core.NewResizableView(
		executionStates, statesViewSize,
		executionStateEvents, eventsViewSize,
		tview.FlexColumn,
	)

	const stateEventsViewSize = 10
	const tabPaneViewSize = 10

	var resizableView = core.NewResizableView(
		tabView, tabPaneViewSize,
		resizableStatesView, stateEventsViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.
		AddItem(resizableView, 0, 1, false)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{executionStates, executionStateEvents},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	executionSummary.ErrorMessageCallback = errorHandler
	executionStates.ErrorMessageCallback = errorHandler

	var detailsView = &SfnExecutionDetailsPageView{
		ServicePageView:   serviceView,
		selectedExecution: "",
		summaryTable:      executionSummary,
		detailsTable:      executionStates,
		stateFlowView:     stateFlowView,
		serviceCtx:        serviceViewCtx,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *SfnExecutionDetailsPageView) initInputCapture() {
}

// SetStateMachine sets the state machine the executions shown are of. The State
// Flow tab shows the execution on its states.
func (inst *SfnExecutionDetailsPageView) SetStateMachine(stateMachine *sfn.DescribeStateMachineOutput) {
	inst.stateFlowView.SetStateMachine(stateMachine)
}

func NewStepFunctionsHomeView(
	appCtx *core.AppContext,
	api awsapi.StateMachineApi,
	cwlApi awsapi.CloudWatchLogsApi,
) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xFF3399))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		SfnDetailsView = NewSfnDetailsPageView(
			NewSfnListTable(serviceCtx),
			NewSfnExecutionsTable(serviceCtx.AppContext, api, cwlApi),
			NewSfnDetailsTable(serviceCtx),
			serviceCtx,
		)

		SfnExeDetailsView = NewSfnExecutionDetailsPage(
			NewSfnExecutionSummaryTable(serviceCtx),
			NewSfnExecutionDetailsTable(serviceCtx.AppContext, api, cwlApi),
			NewSfnExecutionStatesTable(serviceCtx.AppContext, api),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.STATE_MACHINES), appCtx)

	serviceRootView.
		AddAndSwitchToPage("StateMachines", SfnDetailsView, true).
		AddPage("Execution Details", SfnExeDetailsView, true, true)

	serviceRootView.InitPageNavigation()

	SfnDetailsView.sfnDetailsTable.AddDetailsLoadedFunc(SfnExeDetailsView.SetStateMachine)

	SfnDetailsView.sfnExecutionsTable.SetSelectedFunc(func(row, column int) {
		var selectedExecution = SfnDetailsView.
			sfnExecutionsTable.GetSelectedExecutionArn()
		var sfType = SfnDetailsView.sfnListTable.GetSelectedFunctionType()

		if len(selectedExecution) > 0 {
			if sfType == "EXPRESS" {
				var execution = SfnDetailsView.
					sfnExecutionsTable.GetSelectedExecution()
				SfnExeDetailsView.detailsTable.RefreshExpressExecutionStates(execution, true)
			} else {
				SfnExeDetailsView.summaryTable.RefreshExecutionDetails(selectedExecution, true)
				SfnExeDetailsView.detailsTable.RefreshExecutionStates(selectedExecution, true)
			}
			serviceRootView.ChangePage(1, nil)
		}
	})

	SfnDetailsView.initInputCapture()
	SfnExeDetailsView.initInputCapture()

	return serviceRootView
}
