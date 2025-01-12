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
		[]core.View{
			stateMachinesList,
			tabView.GetTabsList(),
			tabView.GetTabDisplayView(),
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

	var inputsExpandedView = core.JsonTextView[string]{
		TextView: core.NewSearchableTextView("Input", app),
		ExtractTextFunc: func(data string) string {
			return data
		},
	}
	var outputsExpandedView = core.JsonTextView[string]{
		TextView: core.NewSearchableTextView("Output", app),
		ExtractTextFunc: func(data string) string {
			return data
		},
	}

	var selectionFunc = func(_, _ int) {
		inputsExpandedView.SetText(executionDetails.GetSelectedStepInput())
		outputsExpandedView.SetText(executionDetails.GetSelectedStepOutput())
	}

	executionDetails.SetSelectedFunc(selectionFunc)
	executionDetails.SetSelectionChangedFunc(selectionFunc)

	var inputOutputView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(inputsExpandedView.TextView, 0, 1, false).
		AddItem(outputsExpandedView.TextView, 0, 1, false)

	const detailsViewSize = 10
	const inputOutputViewSize = 10

	var resizableView = core.NewResizableView(
		executionDetails, detailsViewSize,
		inputOutputView, inputOutputViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.
		AddItem(executionSummary, 8, 0, true).
		AddItem(resizableView, 0, 1, false)

	serviceView.InitViewNavigation(
		[]core.View{
			outputsExpandedView.TextView,
			inputsExpandedView.TextView,
			executionDetails,
			executionSummary,
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
		api = awsapi.NewStateMachineApi(config, logger)

		stateMachinesDetailsView = NewStateMachinesDetailsPageView(
			tables.NewStateMachinesListTable(app, api, logger),
			tables.NewStateMachineExecutionsTable(app, api, logger),
			tables.NewStateMachineDetailsTable(app, api, logger),
			app, api, logger)

		executionDetailsView = NewStateMachineExectionDetailsPage(
			tables.NewStateMachineExecutionSummaryTable(app, api, logger),
			tables.NewStateMachineExecutionDetailsTable(app, api, logger),
			app, api, logger)
	)

	var serviceRootView = core.NewServiceRootView(app, string(STATE_MACHINES))

	serviceRootView.
		AddAndSwitchToPage("StateMachines", stateMachinesDetailsView, true).
		AddPage("Exection Details", executionDetailsView, true, true)

	serviceRootView.InitPageNavigation()

	stateMachinesDetailsView.stateMachineExecutionsTable.SetSelectedFunc(func(row, column int) {
		var selectedExecution = stateMachinesDetailsView.
			stateMachineExecutionsTable.GetSeletedExecutionArn()
		if len(selectedExecution) > 0 {
			executionDetailsView.summaryTable.RefreshExecutionDetails(selectedExecution, true)
			executionDetailsView.detailsTable.RefreshExecutionDetails(selectedExecution, true)
			serviceRootView.ChangePage(1, nil)
		}
	})

	stateMachinesDetailsView.initInputCapture()
	executionDetailsView.initInputCapture()

	return serviceRootView
}
