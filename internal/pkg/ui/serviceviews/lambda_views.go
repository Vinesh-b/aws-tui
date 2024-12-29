package serviceviews

import (
	"encoding/base64"
	"encoding/json"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaDetailsPageView struct {
	*core.ServicePageView
	SelectedLambda     string
	LambdaListTable    *LambdaListTable
	LambdaDetailsTable *LambdaDetailsTable

	app *tview.Application
	api *awsapi.LambdaApi
}

func NewLambdaDetailsPageView(
	lambdaDetailsTable *LambdaDetailsTable,
	lambdaListTable *LambdaListTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsPageView {
	const detailsViewSize = 4000
	const tableViewSize = 6000

	var mainPage = core.NewResizableView(
		lambdaDetailsTable.RootView, detailsViewSize,
		lambdaListTable.RootView, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	var view = &LambdaDetailsPageView{
		ServicePageView: serviceView,
		SelectedLambda:  "",

		LambdaListTable:    lambdaListTable,
		LambdaDetailsTable: lambdaDetailsTable,
		app:                app,
		api:                api,
	}

	view.InitViewNavigation(
		[]core.View{
			lambdaListTable.RootView,
			lambdaDetailsTable.RootView,
		},
	)
	view.initInputCapture()

	return view
}

func (inst *LambdaDetailsPageView) initInputCapture() {
	inst.LambdaListTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.LambdaListTable.RefreshLambdas(false)
		}
	})

	inst.LambdaListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.LambdaDetailsTable.RefreshDetails(inst.LambdaListTable.GetSeletedLambda(), false)
	})
}

// Todo: Fix navigaion with shared components
type LambdaInvokePageView struct {
	*core.ServicePageView
	SelectedLambda string
	DetailsTable   *LambdaDetailsTable

	logResults     *tview.TextArea
	payloadInput   *tview.TextArea
	responseOutput *tview.TextArea
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdaInvokePageView(
	lambdaDetails *LambdaDetailsTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaInvokePageView {

	var payloadInput = core.CreateTextArea("Event Payload")
	var logResults = core.CreateTextArea("Logs")
	var responseOutput = core.CreateTextArea("Response")

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.
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
	var view = &LambdaInvokePageView{
		ServicePageView: serviceView,
		SelectedLambda:  "",

		DetailsTable:   lambdaDetails,
		payloadInput:   payloadInput,
		logResults:     logResults,
		responseOutput: responseOutput,
		app:            app,
		api:            api,
	}

	view.initInputCapture()

	return view
}

func (inst *LambdaInvokePageView) initInputCapture() {
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

func (inst *LambdaInvokePageView) loadLogs(text string) {
	inst.logResults.SetTitle("Logs")
	inst.logResults.SetText(text, false)
}

func (inst *LambdaInvokePageView) loadResponse(text string) {
	inst.responseOutput.SetTitle("Response")
	var newText, _ = core.TryFormatToJson(text)
	inst.responseOutput.SetText(newText, false)
}

func (inst *LambdaInvokePageView) Invoke() {
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

func NewLambdaHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer core.ResetGlobalStyle()

	var (
		api                = awsapi.NewLambdaApi(config, logger)
		cwl_api            = awsapi.NewCloudWatchLogsApi(config, logger)
		lambdasDetailsView = NewLambdaDetailsPageView(
			NewLambdaDetailsTable(app, api, logger),
			NewLambdasListTable(app, api, logger),
			app, api, logger,
		)
		lambdaInvokeView = NewLambdaInvokePageView(
			NewLambdaDetailsTable(app, api, logger),
			app, api, logger,
		)
		logEventsView = NewLogEventsPage(
			NewLogEventsTable(app, cwl_api, logger),
			app, cwl_api, logger,
		)
		logStreamsView = NewLogStreamsPage(
			NewLogStreamsTable(app, cwl_api, logger),
			app, cwl_api, logger,
		)
	)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView.RootView, true, true).
		AddPage("Streams", logStreamsView.RootView, true, true).
		AddPage("Invoke", lambdaInvokeView, true, true).
		AddAndSwitchToPage("Lambdas", lambdasDetailsView, true)

	var orderedPages = []string{
		"Lambdas",
		"Invoke",
		"Streams",
		"Events",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(LAMBDA), pages, orderedPages).Init()

	lambdasDetailsView.LambdaListTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		lambdaInvokeView.SelectedLambda = lambdasDetailsView.SelectedLambda
		app.SetFocus(lambdasDetailsView.LambdaDetailsTable.RootView)
	})

	lambdasDetailsView.LambdaDetailsTable.Table.SetSelectedFunc(func(row, column int) {
		var selectedLogGroup = lambdasDetailsView.LambdaDetailsTable.Table.GetCell(7, 1).Text

		logStreamsView.LogStreamsTable.SetSeletedLogGroup(selectedLogGroup)
		logStreamsView.LogStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsView.LogStreamsTable.RefreshStreams(true)
		serviceRootView.ChangePage(2, logStreamsView.LogStreamsTable.Table)
	})

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		var selectedLogStream = logStreamsView.LogStreamsTable.GetSeletedLogStream()
		var selectedLogGroup = logStreamsView.LogStreamsTable.GetSeletedLogGroup()

		logEventsView.LogEventsTable.SetSeletedLogGroup(selectedLogGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(selectedLogStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(3, logEventsView.LogEventsTable.Table)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()

	return serviceRootView.RootView
}
