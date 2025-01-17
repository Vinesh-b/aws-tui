package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachinesDetailsPageView struct {
	*core.ServicePageView
	selectedStateMachine        string
	stateMachinesTable          *tables.StateMachinesListTable
	stateMachineExecutionsTable *tables.StateMachineExecutionsTable
	stateMachineDetailsTable    *tables.StateMachineDetailsTable
	app                         *tview.Application
	api                         *awsapi.StateMachineApi
}

func NewStateMachinesDetailsPageView(
	stateMachinesList *tables.StateMachinesListTable,
	stateMachineExecutions *tables.StateMachineExecutionsTable,
	stateMachineDetailsTable *tables.StateMachineDetailsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachinesDetailsPageView {
	var tabView = core.NewTabView(
		[]string{"Executions", "Details"},
		app,
		logger,
	)
	tabView.GetTab("Executions").MainPage.AddItem(stateMachineExecutions, 0, 1, true)
	tabView.GetTab("Details").MainPage.AddItem(stateMachineDetailsTable, 0, 1, true)

	const detailsViewSize = 4000
	const tableViewSize = 6000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		stateMachinesList, tableViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{stateMachinesList},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	stateMachinesList.ErrorMessageCallback = errorHandler
	stateMachineExecutions.ErrorMessageCallback = errorHandler
	stateMachineDetailsTable.ErrorMessageCallback = errorHandler

	var detailsView = &StateMachinesDetailsPageView{
		ServicePageView:             serviceView,
		selectedStateMachine:        "",
		stateMachinesTable:          stateMachinesList,
		stateMachineExecutionsTable: stateMachineExecutions,
		stateMachineDetailsTable:    stateMachineDetailsTable,
		app:                         app,
		api:                         api,
	}

	detailsView.initInputCapture()

	return detailsView
}

func (inst *StateMachinesDetailsPageView) initInputCapture() {
	inst.stateMachinesTable.SetSelectedFunc(func(row, column int) {
		var selectedFunc = inst.stateMachinesTable.GetSeletedFunctionArn()
		inst.stateMachineExecutionsTable.SetSeletedFunctionArn(selectedFunc)
		inst.stateMachineDetailsTable.RefreshDetails(selectedFunc)
		if smType := inst.stateMachinesTable.GetSeletedFunctionType(); smType == "STANDARD" {
			inst.stateMachineExecutionsTable.RefreshExecutions(true)
		} else {
			var group = inst.stateMachineDetailsTable.GetSelectedSmLogGroup()
			inst.stateMachineExecutionsTable.RefreshExpressExecutions(group, true)
		}
	})
}

type StateMachineExectionDetailsPageView struct {
	*core.ServicePageView
	selectedExection string
	summaryTable     *tables.StateMachineExecutionSummaryTable
	detailsTable     *tables.StateMachineExecutionDetailsTable
	searchInput      *tview.InputField
	app              *tview.Application
	api              *awsapi.StateMachineApi
}

func NewStateMachineExectionDetailsPage(
	executionSummary *tables.StateMachineExecutionSummaryTable,
	executionDetails *tables.StateMachineExecutionDetailsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExectionDetailsPageView {

	var inputOutputExpandedView = core.JsonTextView[string]{
		TextView: core.NewSearchableTextView("", app),
		ExtractTextFunc: func(data string) string {
			return data
		},
	}

	var selectionFunc = func(_, _ int) {
		if input := executionDetails.GetSelectedStepInput(); len(input) > 0 {
			inputOutputExpandedView.SetTitle("Input")
			inputOutputExpandedView.SetText(input)
		} else if output := executionDetails.GetSelectedStepOutput(); len(output) > 0 {
			inputOutputExpandedView.SetTitle("Ouput")
			inputOutputExpandedView.SetText(output)
		} else {
			inputOutputExpandedView.SetTitle("Errors")
			inputOutputExpandedView.SetText(executionDetails.GetSelectedStepErrorCause())
		}
	}

	executionDetails.SetSelectedFunc(selectionFunc)
	executionDetails.SetSelectionChangedFunc(selectionFunc)

	var tabView = core.NewTabView(
		[]string{"Input/Output", "Summary"},
		app,
		logger,
	)
	tabView.GetTab("Input/Output").MainPage.AddItem(inputOutputExpandedView.TextView, 0, 1, true)
	tabView.GetTab("Summary").MainPage.AddItem(executionSummary, 0, 1, true)

	const detailsViewSize = 10
	const inputOutputViewSize = 10

	var resizableView = core.NewResizableView(
		tabView, inputOutputViewSize,
		executionDetails, detailsViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.
		AddItem(resizableView, 0, 1, false)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{executionDetails},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	executionSummary.ErrorMessageCallback = errorHandler
	executionDetails.ErrorMessageCallback = errorHandler

	var detailsView = &StateMachineExectionDetailsPageView{
		ServicePageView:  serviceView,
		selectedExection: "",
		summaryTable:     executionSummary,
		detailsTable:     executionDetails,
		app:              app,
		api:              api,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *StateMachineExectionDetailsPageView) initInputCapture() {
}

func NewStepFunctionsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0xFF3399))
	defer core.ResetGlobalStyle()

	var (
		api    = awsapi.NewStateMachineApi(config, logger)
		cwlApi = awsapi.NewCloudWatchLogsApi(config, logger)

		SfnDetailsView = NewStateMachinesDetailsPageView(
			tables.NewStateMachinesListTable(app, api, logger),
			tables.NewStateMachineExecutionsTable(app, api, cwlApi, logger),
			tables.NewStateMachineDetailsTable(app, api, logger),
			app, api, logger)

		SfnExeDetailsView = NewStateMachineExectionDetailsPage(
			tables.NewStateMachineExecutionSummaryTable(app, api, logger),
			tables.NewStateMachineExecutionDetailsTable(app, api, cwlApi, logger),
			app, api, logger)
	)

	var serviceRootView = core.NewServiceRootView(string(STATE_MACHINES), app, &config, logger)

	serviceRootView.
		AddAndSwitchToPage("StateMachines", SfnDetailsView, true).
		AddPage("Exection Details", SfnExeDetailsView, true, true)

	serviceRootView.InitPageNavigation()

	SfnDetailsView.stateMachinesTable.SetSelectedFunc(func(row, column int) {
	})

	SfnDetailsView.stateMachineExecutionsTable.SetSelectedFunc(func(row, column int) {
		var selectedExecution = SfnDetailsView.
			stateMachineExecutionsTable.GetSeletedExecutionArn()
		var sfType = SfnDetailsView.stateMachinesTable.GetSeletedFunctionType()

		if len(selectedExecution) > 0 {
			if sfType == "EXPRESS" {
				var execution = SfnDetailsView.
					stateMachineExecutionsTable.GetSeletedExecution()
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
