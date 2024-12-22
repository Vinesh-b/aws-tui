package serviceviews

import (
	"log"
	"slices"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachinesListTable struct {
	Table            *tview.Table
	SelectedFunction string
	Data             map[string]types.StateMachineListItem

	logger *log.Logger
	app    *tview.Application
	api    *awsapi.StateMachineApi
}

func NewStateMachinesListTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachinesListTable {

	var table = &StateMachinesListTable{
		Table: tview.NewTable(),
		Data:  nil,

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateStateMachinesTable()
	table.Table.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		table.SelectedFunction = table.Table.GetCell(row, 0).Text
	})

	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshStateMachines("", true)
		}
		return event
	})

	return table
}

func (inst *StateMachinesListTable) populateStateMachinesTable() {
	var tableData []core.TableRow
	for _, row := range inst.Data {
		tableData = append(tableData, core.TableRow{
			*row.Name,
			row.CreationDate.Format(time.DateTime),
		})
	}

	core.InitSelectableTable(inst.Table, "State Machines",
		core.TableRow{
			"Name",
			"Creation Date",
		},
		tableData,
		[]int{0, 1},
	)
	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(1, 0)
}

func (inst *StateMachinesListTable) RefreshStateMachines(search string, force bool) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			inst.Data = inst.api.FilterByName(search)
		} else {
			inst.Data = inst.api.ListStateMachines(force)
		}

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateStateMachinesTable()
	})
}

type StateMachineExecutionsTable struct {
	nextToken *string

	Table          *tview.Table
	SelectedLambda string
	Data           []types.ExecutionListItem

	logger *log.Logger
	app    *tview.Application
	api    *awsapi.StateMachineApi
}

func NewStateMachineExecutionsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionsTable {

	var table = &StateMachineExecutionsTable{
		Table: tview.NewTable(),
		Data:  nil,

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateExecutionsTable()
	table.Table.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		table.SelectedLambda = table.Table.GetCell(row, 0).Text
	})

	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshExecutions("", true)
		}
		return event
	})

	return table
}

func (inst *StateMachineExecutionsTable) populateExecutionsTable() {
	var tableData []core.TableRow
	for _, row := range inst.Data {
		tableData = append(tableData, core.TableRow{
			*row.ExecutionArn,
			string(row.Status),
			row.StartDate.Format(time.DateTime),
			row.StopDate.Format(time.DateTime),
		})
	}

	core.InitSelectableTable(inst.Table, "Executions",
		core.TableRow{
			"Execution Arn",
			"Status",
			"Start Date",
			"Stop Date",
		},
		tableData,
		[]int{0, 1, 2, 3},
	)
	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(1, 0)
}

func (inst *StateMachineExecutionsTable) RefreshExecutions(search string, force bool) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			inst.Data, inst.nextToken = inst.api.ListExecutions(search, nil)
		}

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateExecutionsTable()
	})
}

type StateMachineExecutionSummaryTable struct {
	Table                *tview.Table
	Data                 *sfn.DescribeExecutionOutput
	SelectedExecutionArn string

	logger *log.Logger
	app    *tview.Application
	api    *awsapi.StateMachineApi
}

func NewStateMachineExecutionSummaryTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionSummaryTable {

	var table = &StateMachineExecutionSummaryTable{
		Table:                tview.NewTable(),
		Data:                 nil,
		SelectedExecutionArn: "",

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateTable()
	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshExecutionDetails(table.SelectedExecutionArn, true)
		}
		return event
	})

	return table
}

func (inst *StateMachineExecutionSummaryTable) populateTable() {
	var tableData []core.TableRow
	if inst.Data != nil {
		tableData = []core.TableRow{
			{"Name", *inst.Data.Name},
			{"Execution Arn", *inst.Data.ExecutionArn},
			{"StateMachine Arn", *inst.Data.StateMachineArn},
			{"Status", string(inst.Data.Status)},
			{"Start Date", inst.Data.StartDate.Format(time.DateTime)},
			{"Stop Date", inst.Data.StopDate.Format(time.DateTime)},
		}
	}

	core.InitBasicTable(inst.Table, "Execution Summary", tableData, false)
	inst.Table.Select(0, 0)
	inst.Table.ScrollToBeginning()
}

func (inst *StateMachineExecutionSummaryTable) RefreshExecutionDetails(executionArn string, force bool) {
	inst.SelectedExecutionArn = executionArn
	var resultChannel = make(chan struct{})

	go func() {
		inst.Data = inst.api.DescribeExecution(executionArn)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateTable()
	})
}

type StateMachineExecutionDetailsTable struct {
	Table                *tview.Table
	Data                 *sfn.GetExecutionHistoryOutput
	SelectedExecutionArn string

	logger *log.Logger
	app    *tview.Application
	api    *awsapi.StateMachineApi
}

func NewStateMachineExecutionDetailsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionDetailsTable {

	var table = &StateMachineExecutionDetailsTable{
		Table:                tview.NewTable(),
		Data:                 nil,
		SelectedExecutionArn: "",

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateTable()
	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshExecutionDetails(table.SelectedExecutionArn, true)
		}
		return event
	})

	return table
}

type StateDetails struct {
	Id        int64
	Name      string
	Type      string
	Status    string
	Input     string
	Output    string
	StartTime time.Time
	EndTime   time.Time
}

func (inst *StateMachineExecutionDetailsTable) populateTable() {
	var tableData []core.TableRow
	var enteredEventTypes = []types.HistoryEventType{
		types.HistoryEventTypeTaskStateEntered,
		types.HistoryEventTypePassStateEntered,
		types.HistoryEventTypeParallelStateEntered,
		types.HistoryEventTypeFailStateEntered,
		types.HistoryEventTypeSucceedStateEntered,
		types.HistoryEventTypeMapStateEntered,
		types.HistoryEventTypeChoiceStateEntered,
	}

	var exitedEventTypes = []types.HistoryEventType{
		types.HistoryEventTypeTaskStateExited,
		types.HistoryEventTypePassStateExited,
		types.HistoryEventTypeParallelStateExited,
		types.HistoryEventTypeMapStateExited,
		types.HistoryEventTypeChoiceStateExited,
		types.HistoryEventTypeSucceedStateExited,
	}

	var results = []StateDetails{}

	if inst.Data != nil {
		for _, row := range inst.Data.Events {
			if slices.Contains(enteredEventTypes, row.Type) {
				results = append(results, StateDetails{
					Id:        row.Id,
					Name:      aws.ToString(row.StateEnteredEventDetails.Name),
					Type:      strings.Replace(string(row.Type), "Entered", "", 1),
					Status:    "Entered",
					Input:     aws.ToString(row.StateEnteredEventDetails.Input),
					Output:    "",
					StartTime: aws.ToTime(row.Timestamp),
					EndTime:   aws.ToTime(row.Timestamp),
				})
			}

			if slices.Contains(exitedEventTypes, row.Type) {
				var idx = slices.IndexFunc(results, func(d StateDetails) bool {
					return d.Name == aws.ToString(row.StateExitedEventDetails.Name)
				})

				if idx > -1 {
					results[idx].Status = "Succeeded"
					results[idx].Output = aws.ToString(row.StateExitedEventDetails.Output)
					results[idx].EndTime = aws.ToTime(row.Timestamp)
				}
			}
		}
	}

	for _, row := range results {
		tableData = append(tableData, core.TableRow{
			row.Name,
			row.Type,
			row.Status,
			row.EndTime.Sub(row.StartTime).String(),
		})
	}
	core.InitSelectableTable2(inst.Table, "Execution Details",
		core.TableRow{
			"Name",
			"Type",
			"Status",
			"Duration",
		},
		tableData,
		results,
		0,
	)
	inst.Table.Select(1, 0)
}

func (inst *StateMachineExecutionDetailsTable) RefreshExecutionDetails(executionArn string, force bool) {
	inst.SelectedExecutionArn = executionArn
	var resultChannel = make(chan struct{})

	go func() {
		inst.Data = inst.api.GetExecutionHistory(executionArn)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateTable()
	})
}

type StateMachinesDetailsView struct {
	RootView                    *tview.Flex
	SelectedStateMachine        string
	StateMachineExecutionsTable *StateMachineExecutionsTable
	StateMachinesTable          *StateMachinesListTable

	searchabelView *core.SearchableView
	app            *tview.Application
	api            *awsapi.StateMachineApi
}

func NewStateMachinesDetailsView(
	stateMachinesList *StateMachinesListTable,
	stateMachineExecutions *StateMachineExecutionsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachinesDetailsView {

	const detailsViewSize = 4000
	const tableViewSize = 6000

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(stateMachineExecutions.Table, 0, detailsViewSize, false).
		AddItem(stateMachinesList.Table, 0, tableViewSize, true)

	var searchabelView = core.NewSearchableView(app, logger, mainPage)
	var serviceView = core.NewServiceView(app, logger)

	serviceView.RootView = searchabelView.RootView

	serviceView.SetResizableViews(
		stateMachineExecutions.Table, stateMachinesList.Table,
		detailsViewSize, tableViewSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			stateMachinesList.Table,
			stateMachineExecutions.Table,
		},
	)
	var detailsView = &StateMachinesDetailsView{
		RootView:                    serviceView.RootView,
		SelectedStateMachine:        "",
		StateMachinesTable:          stateMachinesList,
		StateMachineExecutionsTable: stateMachineExecutions,

		searchabelView: searchabelView,
		app:            app,
		api:            api,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *StateMachinesDetailsView) initInputCapture() {
	inst.searchabelView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.StateMachinesTable.RefreshStateMachines(inst.searchabelView.GetText(), false)
		case tcell.KeyEsc:
			inst.searchabelView.SetText("")
		default:
			return
		}
	})

	var refreshSelection = func(row int) {
		if row < 1 {
			return
		}
		inst.SelectedStateMachine = inst.StateMachinesTable.Table.GetCell(row, 0).Text
	}

	inst.StateMachinesTable.Table.SetSelectionChangedFunc(func(row, column int) {
		refreshSelection(row)
		inst.StateMachineExecutionsTable.RefreshExecutions(inst.SelectedStateMachine, false)
	})

	inst.StateMachinesTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.StateMachinesTable.RefreshStateMachines("", true)
		}
		return event
	})
}

type StateMachineExectionDetailsView struct {
	RootView         *tview.Flex
	SelectedExection string
	SummaryTable     *StateMachineExecutionSummaryTable
	DetailsTable     *StateMachineExecutionDetailsTable

	searchInput *tview.InputField
	app         *tview.Application
	api         *awsapi.StateMachineApi
}

func NewStateMachineExectionDetailsView(
	executionSummary *StateMachineExecutionSummaryTable,
	executionDetails *StateMachineExecutionDetailsTable,
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExectionDetailsView {

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

	var serviceView = core.NewServiceView(app, logger)
	serviceView.RootView.
		AddItem(executionSummary.Table, 8, 0, true).
		AddItem(executionDetails.Table, 0, detailsViewSize, false).
		AddItem(inputOutputView, 0, inputOutputViewSize, false)

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
	var detailsView = &StateMachineExectionDetailsView{
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

func (inst *StateMachineExectionDetailsView) initInputCapture() {
}

func CreateStepFunctionsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xFF3399))
	defer core.ResetGlobalStyle()

	var (
		api = awsapi.NewStateMachineApi(config, logger)

		stateMachinesDetailsView = NewStateMachinesDetailsView(
			NewStateMachinesListTable(app, api, logger),
			NewStateMachineExecutionsTable(app, api, logger),
			app, api, logger)

		executionDetailsView = NewStateMachineExectionDetailsView(
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
	stateMachinesDetailsView.StateMachineExecutionsTable.Table.SetSelectedFunc(func(row, column int) {
		selectedExecution = stateMachinesDetailsView.
			StateMachineExecutionsTable.
			Table.
			GetCell(row, 0).Text
		executionDetailsView.SummaryTable.RefreshExecutionDetails(selectedExecution, true)
		executionDetailsView.DetailsTable.RefreshExecutionDetails(selectedExecution, true)
		serviceRootView.ChangePage(1, executionDetailsView.SummaryTable.Table)
	})

	stateMachinesDetailsView.initInputCapture()
	executionDetailsView.initInputCapture()

	return serviceRootView.RootView
}
