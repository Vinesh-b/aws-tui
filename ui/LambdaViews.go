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
	LambdasTable   *tview.Table
	DetailsTable   *tview.Table
	SearchInput    *tview.InputField
	RefreshLambdas func(search string)
	RefreshDetails func(lambdaName string)
	RootView       *tview.Flex
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

func createLambdasTable(params tableCreationParams, api *lambda.LambdaApi) (
	*tview.Table, func(search string),
) {
	var table = tview.NewTable()
	populateLambdasTable(table, make(map[string]types.FunctionConfiguration, 0))

	var refreshViewsFunc = func(search string) {
		var data map[string]types.FunctionConfiguration
		var dataChannel = make(chan map[string]types.FunctionConfiguration)
		var resultChannel = make(chan struct{})

		go func() {
			if len(search) > 0 {
				dataChannel <- api.FilterByName(search)
			} else {
				dataChannel <- api.ListLambdas(false)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateLambdasTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func createLambdaDetailsTable(
	params tableCreationParams,
	api *lambda.LambdaApi,
) (*tview.Table, func(lambdaName string)) {
	var table = tview.NewTable()
	populateLambdaDetailsTable(table, nil)

	var refreshViewsFunc = func(lambdaName string) {
		var data map[string]types.FunctionConfiguration
		var dataChannel = make(chan map[string]types.FunctionConfiguration)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListLambdas(false)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			var details *types.FunctionConfiguration = nil
			var val, ok = data[lambdaName]
			if ok {
				details = &val
			}
			populateLambdaDetailsTable(table, details)
		})
	}

	return table, refreshViewsFunc
}

func NewLambdasDetailsView(
	app *tview.Application,
	api *lambda.LambdaApi,
	logger *log.Logger,
) *LambdasDetailsView {
	var (
		params = tableCreationParams{app, logger}

		lambdasTable, refreshLambdasTable   = createLambdasTable(params, api)
		lambdaDetails, refreshLambdaDetails = createLambdaDetailsTable(params, api)
	)

	var onTableSelction = func(row int) {
		if row < 1 {
			return
		}
		refreshLambdaDetails(lambdasTable.GetCell(row, 0).Text)
	}

	lambdasTable.SetSelectionChangedFunc(func(row, column int) {
		onTableSelction(row)
	})

	lambdasTable.SetSelectedFunc(func(row, column int) {
		onTableSelction(row)
		app.SetFocus(lambdaDetails)
	})

	var inputField = createSearchInput("Lambdas")
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			refreshLambdasTable(inputField.GetText())
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var lambdasView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(lambdaDetails, 0, 3000, false).
		AddItem(lambdasTable, 0, 4000, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var startIdx = 0
	initViewNavigation(app, lambdasView, &startIdx,
		[]view{
			inputField,
			lambdasTable,
			lambdaDetails,
		},
	)
	return &LambdasDetailsView{
		LambdasTable:   lambdasTable,
		DetailsTable:   lambdaDetails,
		SearchInput:    inputField,
		RefreshLambdas: refreshLambdasTable,
		RefreshDetails: refreshLambdaDetails,
		RootView:       lambdasView,
	}
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

	var selectedGroupName = ""
	lambdasDetailsView.DetailsTable.SetSelectedFunc(func(row, column int) {
		selectedGroupName = lambdasDetailsView.DetailsTable.GetCell(7, 1).Text
		logStreamsView.RefreshStreams(selectedGroupName, nil, false)
		serviceRootView.ChangePage(1, logStreamsView.LogStreamsTable)
	})

	var streamName = ""
	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		streamName = logStreamsView.LogStreamsTable.GetCell(row, 0).Text
		logEventsView.RefreshEvents(selectedGroupName, streamName, false)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	var searchPrefix = ""
	var searchEvent = ""
	logEventsView.InitInputCapture(&selectedGroupName, &streamName)
	logEventsView.InitSearchInputDoneCallback(&searchEvent)
	logStreamsView.InitInputCapture(&selectedGroupName, &searchPrefix)
	logStreamsView.InitSearchInputDoneCallback(&selectedGroupName, &searchPrefix)

	return serviceRootView.RootView
}
