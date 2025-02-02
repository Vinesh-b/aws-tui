package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	LambdaTagsTable    *tables.LambdaTagsTable
	LogStreamsTable    *tables.LogStreamsTable

	app *tview.Application
	api *awsapi.LambdaApi
}

func NewLambdaDetailsPageView(
	lambdaListTable *tables.LambdaListTable,
	lambdaDetailsTable *tables.LambdaDetailsTable,
	lambdaEnvVarsTable *tables.LambdaEnvVarsTable,
	lambdaVpcConfTable *tables.LambdaVpcConfigTable,
	lambdaTagsTable *tables.LambdaTagsTable,
	logStreamsTable *tables.LogStreamsTable,
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsPageView {
	var tabView = core.NewTabView(app, logger).
		AddAndSwitchToTab("Details", lambdaDetailsTable, 0, 1, true).
		AddTab("Log Streams", logStreamsTable, 0, 1, true).
		AddTab("Environment Vars", lambdaEnvVarsTable, 0, 1, true).
		AddTab("VPC Config", lambdaVpcConfTable, 0, 1, true).
		AddTab("Tags", lambdaTagsTable, 0, 1, true)

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
		LambdaTagsTable:    lambdaTagsTable,
		LogStreamsTable:    logStreamsTable,
		app:                app,
		api:                api,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	lambdaDetailsTable.ErrorMessageCallback = errorHandler
	lambdaListTable.ErrorMessageCallback = errorHandler
	lambdaEnvVarsTable.ErrorMessageCallback = errorHandler
	lambdaVpcConfTable.ErrorMessageCallback = errorHandler
	lambdaTagsTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{lambdaListTable},
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

type LambdaInvokePageView struct {
	*core.ServicePageView
	selectedLambda string
	invokeButton   *tview.Button
	formatButton   *tview.Button
	logResults     *core.SearchableTextView
	payloadInput   *core.TextArea
	responseOutput *core.SearchableTextView
	title          string
	titleExtra     string
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdaInvokePageView(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaInvokePageView {
	var payloadInput = core.NewTextArea("Payload")
	var logResults = core.NewSearchableTextView("Logs", app)
	var responseOutput = core.NewSearchableTextView("Response", app)
	var invokeButton = tview.NewButton("Invoke")
	var formatButton = tview.NewButton("Format Payload")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(invokeButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.
		AddItem(payloadInput, 0, 4000, false).
		AddItem(buttonsView, 3, 0, false).
		AddItem(responseOutput, 0, 4000, false).
		AddItem(logResults, 0, 5000, false)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	logResults.ErrorMessageCallback = errorHandler
	responseOutput.ErrorMessageCallback = errorHandler
	payloadInput.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{payloadInput},
			{invokeButton, formatButton},
			{responseOutput},
			{logResults},
		},
	)
	var view = &LambdaInvokePageView{
		ServicePageView: serviceView,
		selectedLambda:  "",
		invokeButton:    invokeButton,
		formatButton:    formatButton,
		payloadInput:    payloadInput,
		logResults:      logResults,
		responseOutput:  responseOutput,
		title:           "Payload",
		titleExtra:      "",
		app:             app,
		api:             api,
	}

	view.initInputCapture()

	return view
}

func (inst *LambdaInvokePageView) initInputCapture() {
	inst.invokeButton.SetSelectedFunc(func() { inst.Invoke() })
	inst.formatButton.SetSelectedFunc(func() { inst.payloadInput.FormatAsJson() })
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

func (inst *LambdaInvokePageView) SetSelectedLambda(name string) {
	inst.selectedLambda = name
	inst.titleExtra = name
	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	dataLoader.AsyncLoadData(func() {})
	dataLoader.AsyncUpdateView(inst.payloadInput, func() {
		inst.payloadInput.SetTitle(fmt.Sprintf("%s [%s]", inst.title, inst.titleExtra))
	})
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
			inst.responseOutput.ErrorMessageCallback(err.Error())
			return
		}

		var data *lambda.InvokeOutput = nil
		data, err = inst.api.InvokeLambda(inst.selectedLambda, payload)
		if err != nil {
			inst.responseOutput.ErrorMessageCallback(err.Error())
			return
		}

		logResults, err = base64.StdEncoding.DecodeString(aws.ToString(data.LogResult))
		if err != nil {
			inst.logResults.ErrorMessageCallback(err.Error())
			return
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
			tables.NewLambdasListTable(app, api, logger),
			tables.NewLambdaDetailsTable(app, api, logger),
			tables.NewLambdaEnvVarsTable(app, api, logger),
			tables.NewLambdaVpcConfigTable(app, api, logger),
			tables.NewLambdaTagsTable(app, api, logger),
			tables.NewLogStreamsTable(app, cwl_api, logger),
			app, api, logger,
		)
		logEventsView = NewLogEventsPageView(
			tables.NewLogEventsTable(app, cwl_api, logger),
			app, cwl_api, logger,
		)
		lambdaInvokeView = NewLambdaInvokePageView(
			app, api, logger,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(LAMBDA), app, &config, logger)

	serviceRootView.
		AddAndSwitchToPage("Lambdas", lambdasDetailsView, true).
		AddPage("Log Events", logEventsView, true, true).
		AddPage("Invoke", lambdaInvokeView, true, true)

	serviceRootView.InitPageNavigation()
	var lambdasListTable = lambdasDetailsView.LambdaListTable
	var logStreamsTable = lambdasDetailsView.LogStreamsTable
	var lambdaTagsTable = lambdasDetailsView.LambdaTagsTable
	var logEventsTable = logEventsView.LogEventsTable

	var lambdaSelectedFunc = func(_, _ int) {
		var logGroup = lambdasListTable.GetSeletedLambdaLogGroup()
		logStreamsTable.SetSeletedLogGroup(logGroup)
		logStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsTable.RefreshStreams(true)

		var selectedLambda = lambdasListTable.GetSeletedLambda()
		var lambdaName = aws.ToString(selectedLambda.FunctionName)
		lambdaTagsTable.RefreshDetails(selectedLambda)
		lambdaInvokeView.SetSelectedLambda(lambdaName)
	}

	lambdasListTable.SetSelectedFunc(lambdaSelectedFunc)
	lambdasDetailsView.LambdaDetailsTable.SetSelectedFunc(lambdaSelectedFunc)

	logStreamsTable.SetSelectedFunc(func(row, column int) {
		var selectedLogStream = logStreamsTable.GetSeletedLogStream()
		var selectedLogGroup = logStreamsTable.GetSeletedLogGroup()

		logEventsTable.SetSeletedLogGroup(selectedLogGroup)
		logEventsTable.SetSeletedLogStream(selectedLogStream)
		logEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(1, nil)
	})

	logEventsView.InitInputCapture()

	return serviceRootView
}
