package serviceviews

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaDetailsTable struct {
	Table *tview.Table
	Data  *types.FunctionConfiguration

	selectedLambda string
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdaDetailsTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsTable {
	var table = &LambdaDetailsTable{
		Table: tview.NewTable(),
		Data:  nil,

		selectedLambda: "",
		logger:         logger,
		app:            app,
		api:            api,
	}

	table.populateLambdaDetailsTable()
	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshDetails(table.selectedLambda, true)
		}
		return event
	})

	return table
}

func (inst *LambdaDetailsTable) populateLambdaDetailsTable() {
	var tableData []core.TableRow
	if inst.Data != nil {
		tableData = []core.TableRow{
			{"Description", *inst.Data.Description},
			{"Arn", *inst.Data.FunctionArn},
			{"Version", *inst.Data.Version},
			{"MemorySize", fmt.Sprintf("%d", *inst.Data.MemorySize)},
			{"Runtime", string(inst.Data.Runtime)},
			{"Arch", fmt.Sprintf("%v", inst.Data.Architectures)},
			{"Timeout", fmt.Sprintf("%d", *inst.Data.Timeout)},
			{"LoggingGroup", *inst.Data.LoggingConfig.LogGroup},
			{"AppLogLevel", string(inst.Data.LoggingConfig.ApplicationLogLevel)},
			{"State", string(inst.Data.State)},
			{"LastModified", *inst.Data.LastModified},
		}
	}

	core.InitBasicTable(inst.Table, "Lambda Details", tableData, false)
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

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		var val, ok = data[lambdaName]
		if ok {
			inst.Data = &val
		}
		inst.populateLambdaDetailsTable()
	})
}

type LambdasListTable struct {
	Table          *tview.Table
	SelectedLambda string
	Data           map[string]types.FunctionConfiguration

	logger *log.Logger
	app    *tview.Application
	api    *awsapi.LambdaApi
}

func NewLambdasListTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdasListTable {

	var table = &LambdasListTable{
		Table: tview.NewTable(),
		Data:  nil,

		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateLambdasTable()
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

func (inst *LambdasListTable) populateLambdasTable() {
	var tableData []core.TableRow
	for _, row := range inst.Data {
		tableData = append(tableData, core.TableRow{
			*row.FunctionName,
			*row.LastModified,
		})
	}

	core.InitSelectableTable(inst.Table, "Lambdas",
		core.TableRow{
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
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			inst.Data = inst.api.FilterByName(search)
		} else {
			inst.Data = inst.api.ListLambdas(force)
		}

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateLambdasTable()
	})
}

type LambdasDetailsView struct {
	RootView       *tview.Flex
	SelectedLambda string
	LambdasTable   *LambdasListTable
	DetailsTable   *LambdaDetailsTable

	searchInput *tview.InputField
	app         *tview.Application
	api         *awsapi.LambdaApi
}

func NewLambdasDetailsView(
	lambdaDetails *LambdaDetailsTable,
	lambdasList *LambdasListTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdasDetailsView {

	var inputField = core.CreateSearchInput("Lambdas")
	const detailsViewSize = 4000
	const tableViewSize = 6000

	var serviceView = core.NewServiceView(app, logger)
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
		[]core.View{
			inputField,
			lambdasList.Table,
			lambdaDetails.Table,
		},
	)
	var detailsView = &LambdasDetailsView{
		RootView:       serviceView.RootView,
		SelectedLambda: "",

		LambdasTable: lambdasList,
		DetailsTable: lambdaDetails,
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
			inst.LambdasTable.RefreshLambdas(inst.searchInput.GetText(), false)
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
		inst.SelectedLambda = inst.LambdasTable.Table.GetCell(row, 0).Text
	}

	inst.LambdasTable.Table.SetSelectionChangedFunc(func(row, column int) {
		refreshSelection(row)
		inst.DetailsTable.RefreshDetails(inst.SelectedLambda, false)
	})

	inst.LambdasTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.LambdasTable.RefreshLambdas("", true)
		}
		return event
	})
}

// Todo: Fix navigaion with shared components
type LambdaInvokeView struct {
	SelectedLambda string
	RootView       *tview.Flex
	DetailsTable   *LambdaDetailsTable

	logResults     *tview.TextArea
	payloadInput   *tview.TextArea
	responseOutput *tview.TextArea
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdaInvokeView(
	lambdaDetails *LambdaDetailsTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaInvokeView {

	var payloadInput = core.CreateTextArea("Event Payload")
	var logResults = core.CreateTextArea("Logs")
	var responseOutput = core.CreateTextArea("Response")

	var serviceView = core.NewServiceView(app, logger)
	serviceView.RootView.
		AddItem(lambdaDetails.Table, 0, 3000, false).
		AddItem(payloadInput, 0, 4000, false).
		AddItem(responseOutput, 0, 4000, false).
		AddItem(logResults, 0, 5000, false)

	serviceView.InitViewNavigation(
		[]core.View{
			payloadInput,
			lambdaDetails.Table,
			logResults,
			responseOutput,
		},
	)
	var invokeView = &LambdaInvokeView{
		RootView:       serviceView.RootView,
		SelectedLambda: "",

		DetailsTable:   lambdaDetails,
		payloadInput:   payloadInput,
		logResults:     logResults,
		responseOutput: responseOutput,
		app:            app,
		api:            api,
	}
	invokeView.initInputCapture()

	return invokeView
}

func (inst *LambdaInvokeView) initInputCapture() {
	inst.DetailsTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.DetailsTable.RefreshDetails(inst.SelectedLambda, true)
		}
		return event
	})

	inst.payloadInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.Invoke()
			return nil
		}
		return event
	})
}

func (inst *LambdaInvokeView) loadLogs(text string) {
	inst.logResults.SetTitle("Logs")
	inst.logResults.SetText(text, false)
}

func (inst *LambdaInvokeView) loadResponse(text string) {
	inst.responseOutput.SetTitle("Response")
	var newText, _ = core.TryFormatToJson(text)
	inst.responseOutput.SetText(newText, false)
}

func (inst *LambdaInvokeView) Invoke() {
	var resultChannel = make(chan struct{})
	var logResults = []byte{}
	var responseOutput = []byte{}

	go func() {
		var payload = make(map[string]any)
		var err = json.Unmarshal([]byte(inst.payloadInput.GetText()), &payload)
		if err != nil {
			// log something to the console
			resultChannel <- struct{}{}
			return
		}

		var data = inst.api.InvokeLambda(inst.SelectedLambda, payload)
		if data != nil {
			logResults, err = base64.StdEncoding.DecodeString(aws.ToString(data.LogResult))
			if err != nil {
				// log something to the console
			}

			responseOutput = data.Payload
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.responseOutput.Box, resultChannel, func() {
		inst.loadResponse(string(responseOutput))
		inst.loadLogs(string(logResults))
	})
}

func CreateLambdaHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer core.ResetGlobalStyle()

	var (
		api                = awsapi.NewLambdaApi(config, logger)
		cwl_api            = awsapi.NewCloudWatchLogsApi(config, logger)
		lambdasDetailsView = NewLambdasDetailsView(
			NewLambdaDetailsTable(app, api, logger),
			NewLambdasListTable(app, api, logger),
			app, api, logger,
		)
		lambdaInvokeView = NewLambdaInvokeView(
			NewLambdaDetailsTable(app, api, logger),
			app, api, logger,
		)
		logEventsView  = NewLogEventsView(app, cwl_api, logger)
		logStreamsView = NewLogStreamsView(app, cwl_api, logger)
	)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView.RootView, true, true).
		AddPage("Streams", logStreamsView.RootView, true, true).
		AddPage("Invoke", lambdaInvokeView.RootView, true, true).
		AddAndSwitchToPage("Lambdas", lambdasDetailsView.RootView, true)

	var orderedPages = []string{
		"Lambdas",
		"Invoke",
		"Streams",
		"Events",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(LAMBDA), pages, orderedPages).Init()

	lambdasDetailsView.LambdasTable.Table.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		lambdaInvokeView.SelectedLambda = lambdasDetailsView.SelectedLambda
	})

	var selectedGroupName = ""
	lambdasDetailsView.DetailsTable.Table.SetSelectedFunc(func(row, column int) {
		selectedGroupName = lambdasDetailsView.DetailsTable.Table.GetCell(7, 1).Text
		logStreamsView.RefreshStreams(selectedGroupName, true)
		serviceRootView.ChangePage(2, logStreamsView.LogStreamsTable)
	})

	var streamName = ""
	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		streamName = logStreamsView.LogStreamsTable.GetCell(row, 0).Text
		logEventsView.RefreshEvents(selectedGroupName, streamName, true)
		serviceRootView.ChangePage(3, logEventsView.LogEventsTable)
	})

	var searchPrefix = ""
	logStreamsView.InitSearchInputBuffer(&searchPrefix)

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()

	return serviceRootView.RootView
}
