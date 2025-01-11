package services

import (
	"encoding/base64"
	"encoding/json"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaDetailsPageView struct {
	*core.ServicePageView
	SelectedLambda     string
	LambdaListTable    *tables.LambdaListTable
	LambdaDetailsTable *tables.LambdaDetailsTable
	LambdaEnvVarsTable *tables.LambdaEnvVarsTable
	LambdaVpcConfTable *tables.LambdaVpcConfigTable

	app *tview.Application
	api *awsapi.LambdaApi
}

func NewLambdaDetailsPageView(
	lambdaDetailsTable *tables.LambdaDetailsTable,
	lambdaEnvVarsTable *tables.LambdaEnvVarsTable,
	lambdaVpcConfTable *tables.LambdaVpcConfigTable,
	lambdaListTable *tables.LambdaListTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsPageView {
	var tabView = core.NewTabView(
		[]string{"Details", "Environment Vars", "VPC Config"},
		app,
		logger,
	)
	tabView.GetTab("Details").MainPage.AddItem(lambdaDetailsTable, 0, 1, true)
	tabView.GetTab("Environment Vars").MainPage.AddItem(lambdaEnvVarsTable, 0, 1, true)
	tabView.GetTab("VPC Config").MainPage.AddItem(lambdaVpcConfTable, 0, 1, true)

	const detailsViewSize = 3000
	const tableViewSize = 7000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		lambdaListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &LambdaDetailsPageView{
		ServicePageView: serviceView,
		SelectedLambda:  "",

		LambdaListTable:    lambdaListTable,
		LambdaDetailsTable: lambdaDetailsTable,
		LambdaEnvVarsTable: lambdaEnvVarsTable,
		LambdaVpcConfTable: lambdaVpcConfTable,
		app:                app,
		api:                api,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	lambdaDetailsTable.ErrorMessageCallback = errorHandler
	lambdaListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[]core.View{
			lambdaListTable,
			tabView.GetTabsList(),
			tabView.GetTabDisplayView(),
		},
	)
	view.initInputCapture()

	return view
}

func (inst *LambdaDetailsPageView) initInputCapture() {
	inst.LambdaListTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedLambda = inst.LambdaListTable.GetSeletedLambda()
		inst.LambdaDetailsTable.RefreshDetails(selectedLambda)
		inst.LambdaEnvVarsTable.RefreshDetails(selectedLambda)
		inst.LambdaVpcConfTable.RefreshDetails(selectedLambda)
	})
}

// Todo: Fix navigaion with shared components
type LambdaInvokePageView struct {
	*core.ServicePageView
	SelectedLambda string
	DetailsTable   *tables.LambdaDetailsTable

	logResults     *core.SearchableTextView
	payloadInput   *core.SearchableTextView
	responseOutput *core.SearchableTextView
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdaInvokePageView(
	lambdaDetails *tables.LambdaDetailsTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaInvokePageView {

	var payloadInput = core.NewSearchableTextView("Event Payload", app)
	var logResults = core.NewSearchableTextView("Logs", app)
	var responseOutput = core.NewSearchableTextView("Response", app)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.
		AddItem(lambdaDetails, 0, 3000, false).
		AddItem(payloadInput, 0, 4000, false).
		AddItem(responseOutput, 0, 4000, false).
		AddItem(logResults, 0, 5000, false)

	serviceView.InitViewNavigation(
		[]core.View{
			payloadInput,
			lambdaDetails,
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
	inst.payloadInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
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
	var logResults = []byte{}
	var responseOutput = []byte{}
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		var payload = make(map[string]any)
		err = json.Unmarshal([]byte(inst.payloadInput.GetText()), &payload)
		if err != nil {
			// log something to the console
			return
		}

		var data *lambda.InvokeOutput = nil
		data, err = inst.api.InvokeLambda(inst.SelectedLambda, payload)
		if err != nil {
			// log something to the console
		}

		logResults, err = base64.StdEncoding.DecodeString(aws.ToString(data.LogResult))
		if err != nil {
			// log something to the console
		}

		responseOutput = data.Payload
	})

	dataLoader.AsyncUpdateView(inst.responseOutput.Box, func() {
		inst.loadResponse(string(responseOutput))
		inst.loadLogs(string(logResults))
	})
}

func NewLambdaHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer core.ResetGlobalStyle()

	var (
		api                = awsapi.NewLambdaApi(config, logger)
		cwl_api            = awsapi.NewCloudWatchLogsApi(config, logger)
		lambdasDetailsView = NewLambdaDetailsPageView(
			tables.NewLambdaDetailsTable(app, api, logger),
			tables.NewLambdaEnvVarsTable(app, api, logger),
			tables.NewLambdaVpcConfigTable(app, api, logger),
			tables.NewLambdasListTable(app, api, logger),
			app, api, logger,
		)
		logEventsView = NewLogEventsPageView(
			tables.NewLogEventsTable(app, cwl_api, logger),
			app, cwl_api, logger,
		)
		logStreamsView = NewLogStreamsPageView(
			tables.NewLogStreamDetailsTable(app, cwl_api, logger),
			tables.NewLogStreamsTable(app, cwl_api, logger),
			app, cwl_api, logger,
		)
	)

	var serviceRootView = core.NewServiceRootView(app, string(LAMBDA))

	serviceRootView.
		AddAndSwitchToPage("Lambdas", lambdasDetailsView, true).
		AddPage("Streams", logStreamsView, true, true).
		AddPage("Events", logEventsView, true, true)

	serviceRootView.InitPageNavigation()

	var lambdaSelectedFunc = func(_, _ int) {
		var logGroup = lambdasDetailsView.LambdaListTable.GetSeletedLambdaLogGroup()
		logStreamsView.LogStreamsTable.SetSeletedLogGroup(logGroup)
		logStreamsView.LogStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsView.LogStreamsTable.RefreshStreams(true)
		serviceRootView.ChangePage(1, nil)
	}

	lambdasDetailsView.LambdaListTable.SetSelectedFunc(lambdaSelectedFunc)
	lambdasDetailsView.LambdaDetailsTable.SetSelectedFunc(lambdaSelectedFunc)

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		var selectedLogStream = logStreamsView.LogStreamsTable.GetSeletedLogStream()
		var selectedLogGroup = logStreamsView.LogStreamsTable.GetSeletedLogGroup()

		logEventsView.LogEventsTable.SetSeletedLogGroup(selectedLogGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(selectedLogStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, nil)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()

	return serviceRootView
}
