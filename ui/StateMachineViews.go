package ui

import (
	"log"
	"time"

	"aws-tui/statemachine"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachinesListTable struct {
	Table          *tview.Table
	SelectedLambda string
	Data           map[string]types.StateMachineListItem

	logger *log.Logger
	app    *tview.Application
	api    *statemachine.StateMachineApi
}

func NewStateMachinesListTable(
	app *tview.Application,
	api *statemachine.StateMachineApi,
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
		table.SelectedLambda = table.Table.GetCell(row, 0).Text
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
	var tableData []tableRow
	for _, row := range inst.Data {
		tableData = append(tableData, tableRow{
			*row.Name,
			row.CreationDate.Format(time.DateTime),
		})
	}

	initSelectableTable(inst.Table, "State Machines",
		tableRow{
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

	go loadData(inst.app, inst.Table.Box, resultChannel, func() {
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
	api    *statemachine.StateMachineApi
}

func NewStateMachineExecutionsTable(
	app *tview.Application,
	api *statemachine.StateMachineApi,
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
	var tableData []tableRow
	for _, row := range inst.Data {
		tableData = append(tableData, tableRow{
			*row.Name,
			string(row.Status),
			row.StartDate.Format(time.DateTime),
			row.StopDate.Format(time.DateTime),
		})
	}

	initSelectableTable(inst.Table, "Executions",
		tableRow{
			"Name",
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

	go loadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateExecutionsTable()
	})
}

type StateMachinesDetailsView struct {
	RootView                    *tview.Flex
	SelectedStateMachine        string
	StateMachineExecutionsTable *StateMachineExecutionsTable
	StateMachinesTable          *StateMachinesListTable

	searchInput *tview.InputField
	app         *tview.Application
	api         *statemachine.StateMachineApi
}

func NewStateMachinesDetailsView(
	stateMachinesList *StateMachinesListTable,
	stateMachineExecutions *StateMachineExecutionsTable,
	app *tview.Application,
	api *statemachine.StateMachineApi,
	logger *log.Logger,
) *StateMachinesDetailsView {

	var inputField = createSearchInput("State Machine")
	const detailsViewSize = 4000
	const tableViewSize = 6000

	var serviceView = NewServiceView(app, logger)
	serviceView.RootView.
		AddItem(stateMachineExecutions.Table, 0, detailsViewSize, false).
		AddItem(stateMachinesList.Table, 0, tableViewSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		stateMachineExecutions.Table, stateMachinesList.Table,
		detailsViewSize, tableViewSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			stateMachinesList.Table,
			stateMachineExecutions.Table,
		},
	)
	var detailsView = &StateMachinesDetailsView{
		RootView:                    serviceView.RootView,
		SelectedStateMachine:        "",
		StateMachinesTable:          stateMachinesList,
		StateMachineExecutionsTable: stateMachineExecutions,

		searchInput: inputField,
		app:         app,
		api:         api,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *StateMachinesDetailsView) initInputCapture() {
	inst.searchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.StateMachinesTable.RefreshStateMachines(inst.searchInput.GetText(), false)
		case tcell.KeyEsc:
			inst.searchInput.SetText("")
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

func createStepFunctionsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0xFF3399))
	defer resetGlobalStyle()

	var (
		api = statemachine.NewStateMachineApi(config, logger)

		stateMachinesDetailsView = NewStateMachinesDetailsView(
			NewStateMachinesListTable(app, api, logger),
			NewStateMachineExecutionsTable(app, api, logger),
			app, api, logger)
	)

	var pages = tview.NewPages().
		AddAndSwitchToPage("StateMachines", stateMachinesDetailsView.RootView, true)

	var orderedPages = []string{
		"StateMachines",
	}

	var serviceRootView = NewServiceRootView(
		app, string(STATE_MACHINES), pages, orderedPages).Init()

	stateMachinesDetailsView.initInputCapture()

	return serviceRootView.RootView
}
