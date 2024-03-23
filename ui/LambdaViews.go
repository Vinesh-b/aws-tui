package ui

import (
	"fmt"
	"log"

	"aws-tui/cloudwatchlogs"
	"aws-tui/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaDetailsTable struct {
	Table *tview.Table

	selectedLambda string
	logger         *log.Logger
	app            *tview.Application
	api            *lambda.LambdaApi
}

func NewLambdaDetailsTable(
	app *tview.Application,
	api *lambda.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsTable {
	var table = &LambdaDetailsTable{
		Table: tview.NewTable(),

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateLambdaDetailsTable(nil)
	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshDetails("", true)
		}
		return event
	})

	return table
}

func (inst *LambdaDetailsTable) populateLambdaDetailsTable(data *types.FunctionConfiguration) {
	var tableData []tableRow
	if data != nil {
		tableData = []tableRow{
			{"Description", *data.Description},
			{"Arn", *data.FunctionArn},
			{"Version", *data.Version},
			{"MemorySize", fmt.Sprintf("%d", *data.MemorySize)},
			{"Runtime", string(data.Runtime)},
			{"Arch", fmt.Sprintf("%v", data.Architectures)},
			{"Timeout", fmt.Sprintf("%d", *data.Timeout)},
			{"LoggingGroup", *data.LoggingConfig.LogGroup},
			{"AppLogLevel", string(data.LoggingConfig.ApplicationLogLevel)},
			{"State", string(data.State)},
			{"LastModified", *data.LastModified},
		}
	}

	initBasicTable(inst.Table, "Lambda Details", tableData, false)
	inst.Table.Select(0, 0)
	inst.Table.ScrollToBeginning()
}

func (inst *LambdaDetailsTable) RefreshDetails(lambdaName string, force bool) {
	inst.selectedLambda = lambdaName
	var data map[string]types.FunctionConfiguration
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListLambdas(force)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.Table.Box, resultChannel, func() {
		var details *types.FunctionConfiguration = nil
		var val, ok = data[lambdaName]
		if ok {
			details = &val
		}
		inst.populateLambdaDetailsTable(details)
	})
}

type LambdasListTable struct {
	Table          *tview.Table
	SelectedLambda string

	logger *log.Logger
	app    *tview.Application
	api    *lambda.LambdaApi
}

func NewLambdasListTable(
	app *tview.Application,
	api *lambda.LambdaApi,
	logger *log.Logger,
) *LambdasListTable {

	var table = &LambdasListTable{
		Table: tview.NewTable(),

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateLambdasTable(nil)
	table.Table.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		table.SelectedLambda = table.Table.GetCell(row, 0).Text
	})

	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshLambdas("", true)
		}
		return event
	})

	return table
}

func (inst *LambdasListTable) populateLambdasTable(data map[string]types.FunctionConfiguration) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.FunctionName,
			*row.LastModified,
		})
	}

	initSelectableTable(inst.Table, "Lambdas",
		tableRow{
			"Name",
			"LastModified",
		},
		tableData,
		[]int{0, 1},
	)
	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(1, 0)
}

func (inst *LambdasListTable) RefreshLambdas(search string, force bool) {
	var data map[string]types.FunctionConfiguration
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			data = inst.api.FilterByName(search)
		} else {
			data = inst.api.ListLambdas(force)
		}

		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateLambdasTable(data)
	})
}

type LambdasDetailsView struct {
	RootView       *tview.Flex
	SelectedLambda string

	lambdasTable *LambdasListTable
	detailsTable *LambdaDetailsTable
	searchInput  *tview.InputField
	app          *tview.Application
	api          *lambda.LambdaApi
}

func NewLambdasDetailsView(
	lambdaDetails *LambdaDetailsTable,
	lambdasList *LambdasListTable,
	app *tview.Application,
	api *lambda.LambdaApi,
	logger *log.Logger,
) *LambdasDetailsView {

	var inputField = createSearchInput("Lambdas")
	const detailsViewSize = 4000
	const tableViewSize = 6000

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(lambdaDetails.Table, 0, detailsViewSize, false).
		AddItem(lambdasList.Table, 0, tableViewSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		lambdaDetails.Table, lambdasList.Table,
		detailsViewSize, tableViewSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			lambdasList.Table,
			lambdaDetails.Table,
		},
	)
	var detailsView = &LambdasDetailsView{
		RootView:       serviceView.RootView,
		SelectedLambda: "",

		lambdasTable: lambdasList,
		detailsTable: lambdaDetails,
		searchInput:  inputField,
		app:          app,
		api:          api,
	}
	detailsView.initInputCapture()

	return detailsView
}

func (inst *LambdasDetailsView) initInputCapture() {
	inst.searchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.lambdasTable.RefreshLambdas(inst.searchInput.GetText(), false)
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
		inst.SelectedLambda = inst.lambdasTable.Table.GetCell(row, 0).Text
	}

	inst.lambdasTable.Table.SetSelectionChangedFunc(func(row, column int) {
		refreshSelection(row)
		inst.detailsTable.RefreshDetails(inst.SelectedLambda, false)
	})

	inst.lambdasTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.lambdasTable.RefreshLambdas("", true)
		}
		return event
	})

	inst.detailsTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var selctedRow, _ = inst.lambdasTable.Table.GetSelection()
		switch event.Key() {
		case tcell.KeyCtrlR:
			refreshSelection(selctedRow)
			inst.detailsTable.RefreshDetails(inst.SelectedLambda, false)
		}
		return event
	})
}

func createLambdaHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0xCC6600))
	defer resetGlobalStyle()

	var (
		api                = lambda.NewLambdaApi(config, logger)
		cwl_api            = cloudwatchlogs.NewCloudWatchLogsApi(config, logger)
		detailsTable       = NewLambdaDetailsTable(app, api, logger)
		lambdasList        = NewLambdasListTable(app, api, logger)
		lambdasDetailsView = NewLambdasDetailsView(detailsTable, lambdasList, app, api, logger)
		logEventsView      = NewLogEventsView(app, cwl_api, logger)
		logStreamsView     = NewLogStreamsView(app, cwl_api, logger)
	)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView.RootView, true, true).
		AddPage("Streams", logStreamsView.RootView, true, true).
		AddAndSwitchToPage("Lambdas", lambdasDetailsView.RootView, true)

	var orderedPages = []string{
		"Lambdas",
		"Streams",
		"Events",
	}

	var serviceRootView = NewServiceRootView(
		app, string(LAMBDA), pages, orderedPages).Init()


	var selectedGroupName = ""
	detailsTable.Table.SetSelectedFunc(func(row, column int) {
		selectedGroupName = detailsTable.Table.GetCell(7, 1).Text
		logStreamsView.RefreshStreams(selectedGroupName, true)
		serviceRootView.ChangePage(1, logStreamsView.LogStreamsTable)
	})

	var streamName = ""
	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		streamName = logStreamsView.LogStreamsTable.GetCell(row, 0).Text
		logEventsView.RefreshEvents(selectedGroupName, streamName, true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	var searchPrefix = ""
	logStreamsView.InitSearchInputBuffer(&searchPrefix)

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()

	return serviceRootView.RootView
}
