package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	app                  *tview.Application
	api                  *awsapi.StateMachineApi
}

func NewSfnDetailsPageView(
	sfnListTable *tables.SfnListTable,
	sfnExecutionsTable *tables.SfnExecutionsTable,
	stateMachineDetailsTable *tables.SfnDetailsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *SfnDetailsPageView {
	var tabView = core.NewTabView(app, logger).
		AddAndSwitchToTab("Executions", sfnExecutionsTable, 0, 1, true).
		AddTab("Details", stateMachineDetailsTable, 0, 1, true)

	const detailsViewSize = 4000
	const tableViewSize = 6000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		sfnListTable, tableViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
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
		app:                  app,
		api:                  api,
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
	detailsTable     *tables.SfnExecutionDetailsTable
	searchInput      *tview.InputField
	app              *tview.Application
	api              *awsapi.StateMachineApi
}

func NewSfnExectionDetailsPage(
	executionSummary *tables.SfnExecutionSummaryTable,
	executionDetails *tables.SfnExecutionDetailsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *SfnExectionDetailsPageView {

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

	var tabView = core.NewTabView(app, logger).
		AddAndSwitchToTab("Input/Output", inputOutputExpandedView.TextView, 0, 1, true).
		AddTab("Summary", executionSummary, 0, 1, true)

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

	var detailsView = &SfnExectionDetailsPageView{
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

func (inst *SfnExectionDetailsPageView) initInputCapture() {
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

		SfnDetailsView = NewSfnDetailsPageView(
			tables.NewSfnListTable(app, api, logger),
			tables.NewSfnExecutionsTable(app, api, cwlApi, logger),
			tables.NewSfnDetailsTable(app, api, logger),
			app, api, logger)

		SfnExeDetailsView = NewSfnExectionDetailsPage(
			tables.NewSfnExecutionSummaryTable(app, api, logger),
			tables.NewSfnExecutionDetailsTable(app, api, cwlApi, logger),
			app, api, logger)
	)

	var serviceRootView = core.NewServiceRootView(string(STATE_MACHINES), app, &config, logger)

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
