package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachinesDetailsPageView struct {
	*core.ServicePageView
	selectedStateMachine        string
	stateMachineExecutionsTable *StateMachineExecutionsTable
	stateMachinesTable          *StateMachinesListTable
	app                         *tview.Application
	api                         *awsapi.StateMachineApi
}

func NewStateMachinesDetailsPageView(
	stateMachinesList *StateMachinesListTable,
	stateMachineExecutions *StateMachineExecutionsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachinesDetailsPageView {

	const detailsViewSize = 4000
	const tableViewSize = 6000

	var mainPage = core.NewResizableView(
		stateMachineExecutions, detailsViewSize,
		stateMachinesList, tableViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			stateMachinesList,
			stateMachineExecutions,
		},
	)

	var detailsView = &StateMachinesDetailsPageView{
		ServicePageView:             serviceView,
		selectedStateMachine:        "",
		stateMachinesTable:          stateMachinesList,
		stateMachineExecutionsTable: stateMachineExecutions,
		app:                         app,
		api:                         api,
	}

	detailsView.initInputCapture()

	return detailsView
}

func (inst *StateMachinesDetailsPageView) initInputCapture() {
	inst.stateMachinesTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedFunc = inst.stateMachinesTable.GetSeletedFunctionArn()
		inst.stateMachineExecutionsTable.SetSeletedFunctionArn(selectedFunc)
		inst.stateMachineExecutionsTable.RefreshExecutions(false)
	})
}

type StateMachineExectionDetailsPageView struct {
	*core.ServicePageView
	selectedExection string
	summaryTable     *StateMachineExecutionSummaryTable
	detailsTable     *StateMachineExecutionDetailsTable
	searchInput      *tview.InputField
	app              *tview.Application
	api              *awsapi.StateMachineApi
}

func NewStateMachineExectionDetailsPage(
	executionSummary *StateMachineExecutionSummaryTable,
	executionDetails *StateMachineExecutionDetailsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExectionDetailsPageView {

	var inputsExpandedView = core.JsonTextView[StateDetails]{
		TextArea: core.CreateTextArea("Input"),
		ExtractTextFunc: func(data StateDetails) string {
			return data.Input
		},
	}
	var outputsExpandedView = core.JsonTextView[StateDetails]{
		TextArea: core.CreateTextArea("Output"),
		ExtractTextFunc: func(data StateDetails) string {
			return data.Output
		},
	}

	executionDetails.Table.SetSelectionChangedFunc(func(row, column int) {
		var privateDataColIdx = 0
		var col = column
		if privateDataColIdx >= 0 {
			col = privateDataColIdx
		}
		var privateData = executionDetails.Table.GetCell(row, col).Reference
		if row < 1 || privateData == nil {
			return
		}
		inputsExpandedView.SetText(privateData.(StateDetails))
		outputsExpandedView.SetText(privateData.(StateDetails))
	})

	var inputOutputView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(inputsExpandedView.TextArea, 0, 1, false).
		AddItem(outputsExpandedView.TextArea, 0, 1, false)

	const detailsViewSize = 10
	const inputOutputViewSize = 10

	var resizableView = core.NewResizableView(
		executionDetails, detailsViewSize,
		inputOutputView, inputOutputViewSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.
		AddItem(executionSummary.Table, 8, 0, true).
		AddItem(resizableView, 0, 1, false)

	serviceView.InitViewNavigation(
		[]core.View{
			outputsExpandedView.TextArea,
			inputsExpandedView.TextArea,
			executionDetails.Table,
			executionSummary.Table,
		},
	)
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
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xFF3399))
	defer core.ResetGlobalStyle()

	var (
		api = awsapi.NewStateMachineApi(config, logger)

		stateMachinesDetailsView = NewStateMachinesDetailsPageView(
			NewStateMachinesListTable(app, api, logger),
			NewStateMachineExecutionsTable(app, api, logger),
			app, api, logger)

		executionDetailsView = NewStateMachineExectionDetailsPage(
			NewStateMachineExecutionSummaryTable(app, api, logger),
			NewStateMachineExecutionDetailsTable(app, api, logger),
			app, api, logger)
	)

	var pages = tview.NewPages().
		AddPage("Exection Details", executionDetailsView, true, true).
		AddAndSwitchToPage("StateMachines", stateMachinesDetailsView, true)

	var orderedPages = []string{
		"StateMachines",
		"Exection Details",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(STATE_MACHINES), pages, orderedPages).Init()

	var selectedExecution = ""
	stateMachinesDetailsView.stateMachineExecutionsTable.SetSelectedFunc(func(row, column int) {
		selectedExecution = stateMachinesDetailsView.
			stateMachineExecutionsTable.GetSeletedExecutionArn()
		executionDetailsView.summaryTable.RefreshExecutionDetails(selectedExecution, true)
		executionDetailsView.detailsTable.RefreshExecutionDetails(selectedExecution, true)
		serviceRootView.ChangePage(1, executionDetailsView.summaryTable.Table)
	})

	stateMachinesDetailsView.initInputCapture()
	executionDetailsView.initInputCapture()

	return serviceRootView.RootView
}
