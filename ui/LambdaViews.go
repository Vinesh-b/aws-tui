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

type LambdasDetailsView struct {
	LambdasTable *tview.Table
	DetailsTable *tview.Table
	SearchInput  *tview.InputField
	RootView     *tview.Flex

	app *tview.Application
	api *lambda.LambdaApi
}

func populateLambdasTable(table *tview.Table, data map[string]types.FunctionConfiguration) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.FunctionName,
			*row.LastModified,
		})
	}

	initSelectableTable(table, "Lambdas",
		tableRow{
			"Name",
			"LastModified",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLambdaDetailsTable(table *tview.Table, data *types.FunctionConfiguration) {
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

	initBasicTable(table, "Lambda Details", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func NewLambdasDetailsView(
	app *tview.Application,
	api *lambda.LambdaApi,
	logger *log.Logger,
) *LambdasDetailsView {
	var lambdaDetails = tview.NewTable()
	populateLambdaDetailsTable(lambdaDetails, nil)

	var lambdasTable = tview.NewTable()
	populateLambdasTable(lambdasTable, make(map[string]types.FunctionConfiguration, 0))

	var inputField = createSearchInput("Lambdas")
	const detailsViewSize = 4000
	const tableViewSize = 6000

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(lambdaDetails, 0, detailsViewSize, false).
		AddItem(lambdasTable, 0, tableViewSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		lambdaDetails, lambdasTable,
		detailsViewSize, tableViewSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			lambdasTable,
			lambdaDetails,
		},
	)
	return &LambdasDetailsView{
		LambdasTable: lambdasTable,
		DetailsTable: lambdaDetails,
		SearchInput:  inputField,
		RootView:     serviceView.RootView,
		app:          app,
		api:          api,
	}
}

func (inst *LambdasDetailsView) RefreshLambdas(search string, force bool) {
	var data map[string]types.FunctionConfiguration
	var dataChannel = make(chan map[string]types.FunctionConfiguration)
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			dataChannel <- inst.api.FilterByName(search)
		} else {
			dataChannel <- inst.api.ListLambdas(force)
		}
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.LambdasTable.Box, resultChannel, func() {
		populateLambdasTable(inst.LambdasTable, data)
	})
}

func (inst *LambdasDetailsView) RefreshDetails(lambdaName string, force bool) {
	var data map[string]types.FunctionConfiguration
	var dataChannel = make(chan map[string]types.FunctionConfiguration)
	var resultChannel = make(chan struct{})

	go func() {
		dataChannel <- inst.api.ListLambdas(force)
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.DetailsTable.Box, resultChannel, func() {
		var details *types.FunctionConfiguration = nil
		var val, ok = data[lambdaName]
		if ok {
			details = &val
		}
		populateLambdaDetailsTable(inst.DetailsTable, details)
	})
}

func (inst *LambdasDetailsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshLambdas(inst.SearchInput.GetText(), false)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})

	var refreshDetails = func(row int, force bool) {
		if row < 1 {
			return
		}
		inst.RefreshDetails(inst.LambdasTable.GetCell(row, 0).Text, force)
	}

	inst.LambdasTable.SetSelectionChangedFunc(func(row, column int) {
		refreshDetails(row, false)
	})

	inst.LambdasTable.SetSelectedFunc(func(row, column int) {
		refreshDetails(row, false)
		inst.app.SetFocus(inst.LambdasTable)
	})

	inst.LambdasTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshLambdas("", true)
		}
		return event
	})

	inst.DetailsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var selctedRow, _ = inst.LambdasTable.GetSelection()
		switch event.Key() {
		case tcell.KeyCtrlR:
			refreshDetails(selctedRow, true)
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
		lambdasDetailsView = NewLambdasDetailsView(app, api, logger)
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

	lambdasDetailsView.InitInputCapture()

	var selectedGroupName = ""
	lambdasDetailsView.DetailsTable.SetSelectedFunc(func(row, column int) {
		selectedGroupName = lambdasDetailsView.DetailsTable.GetCell(7, 1).Text
		logStreamsView.RefreshStreams(selectedGroupName, true)
		serviceRootView.ChangePage(1, logStreamsView.LogStreamsTable)
	})

	var streamName = ""
	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		streamName = logStreamsView.LogStreamsTable.GetCell(row, 0).Text
		logEventsView.RefreshEvents(selectedGroupName, streamName, true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	logEventsView.InitInputCapture()

	var searchPrefix = ""
	logStreamsView.InitInputCapture()
	logStreamsView.InitSearchInputBuffer(&searchPrefix)

	return serviceRootView.RootView
}
