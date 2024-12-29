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
	RootView                    *tview.Flex
	SelectedStateMachine        string
	StateMachineExecutionsTable *StateMachineExecutionsTable
	StateMachinesTable          *StateMachinesListTable

	searchableView *core.SearchableView_OLD
	app            *tview.Application
	api            *awsapi.StateMachineApi
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

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(stateMachineExecutions.RootView, 0, detailsViewSize, false).
		AddItem(stateMachinesList.RootView, 0, tableViewSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		stateMachineExecutions.RootView, stateMachinesList.RootView,
		detailsViewSize, tableViewSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			stateMachinesList.RootView,
			stateMachineExecutions.RootView,
		},
	)
	var detailsView = &StateMachinesDetailsPageView{
		RootView:                    serviceView.RootView,
		SelectedStateMachine:        "",
		StateMachinesTable:          stateMachinesList,
		StateMachineExecutionsTable: stateMachineExecutions,

		searchableView: serviceView.SearchableView,
		app:            app,
		api:            api,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *StateMachinesDetailsPageView) initInputCapture() {
	inst.searchableView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.StateMachinesTable.RefreshStateMachines(inst.searchableView.GetText(), false)
		case tcell.KeyEsc:
			inst.searchableView.SetText("")
		default:
			return
		}
	})

	inst.StateMachinesTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedFunc = inst.StateMachinesTable.GetSeletedFunctionArn()
		inst.StateMachineExecutionsTable.SetSeletedFunctionArn(selectedFunc)
		inst.StateMachineExecutionsTable.RefreshExecutions(false)
	})
}

type StateMachineExectionDetailsPageView struct {
	RootView         *tview.Flex
	SelectedExection string
	SummaryTable     *StateMachineExecutionSummaryTable
	DetailsTable     *StateMachineExecutionDetailsTable

	searchInput *tview.InputField
	app         *tview.Application
	api         *awsapi.StateMachineApi
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

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(executionSummary.Table, 8, 0, true).
		AddItem(executionDetails.Table, 0, detailsViewSize, false).
		AddItem(inputOutputView, 0, inputOutputViewSize, false)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		executionDetails.Table, inputOutputView,
		detailsViewSize, inputOutputViewSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			outputsExpandedView.TextArea,
			inputsExpandedView.TextArea,
			executionDetails.Table,
			executionSummary.Table,
		},
	)
	var detailsView = &StateMachineExectionDetailsPageView{
		RootView:         serviceView.RootView,
		SelectedExection: "",

		SummaryTable: executionSummary,
		DetailsTable: executionDetails,
		app:          app,
		api:          api,
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
		AddPage("Exection Details", executionDetailsView.RootView, true, true).
		AddAndSwitchToPage("StateMachines", stateMachinesDetailsView.RootView, true)

	var orderedPages = []string{
		"StateMachines",
		"Exection Details",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(STATE_MACHINES), pages, orderedPages).Init()

	var selectedExecution = ""
	stateMachinesDetailsView.StateMachineExecutionsTable.SetSelectedFunc(func(row, column int) {
		selectedExecution = stateMachinesDetailsView.
			StateMachineExecutionsTable.GetSeletedExecutionArn()
		executionDetailsView.SummaryTable.RefreshExecutionDetails(selectedExecution, true)
		executionDetailsView.DetailsTable.RefreshExecutionDetails(selectedExecution, true)
		serviceRootView.ChangePage(1, executionDetailsView.SummaryTable.Table)
	})

	stateMachinesDetailsView.initInputCapture()
	executionDetailsView.initInputCapture()

	return serviceRootView.RootView
}
