package lambda

import (
	"encoding/base64"
	"encoding/json"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaTabName = string

const (
	LambdaTabDetails   LambdaTabName = "Details"
	LambdaTabLogSreams LambdaTabName = "Log Streams"
	LambdaTabTriggers  LambdaTabName = "Triggers"
	LambdaTabPerms     LambdaTabName = "Permissions"
	LambdaTabVersions  LambdaTabName = "Versions"
	LambdaTabLayers    LambdaTabName = "Layers"
	LambdaTabInvoke    LambdaTabName = "Invoke"
	LambdaTabMetrics   LambdaTabName = "Metrics"
)

type LambdaDetailsPageView struct {
	*core.ServicePageView
	SelectedLambda     string
	LambdaListTable    *LambdaListTable
	LambdaDetailsTable *LambdaDetailsTable
	LogStreamsTable    *cloudwatch.LogStreamsTable
	LambdaInvokeView   *LambdaInvokePageView
	LambdaMetricsView  *LambdaMetricsPageView
	TriggersView       *LambdaItemsView[eventSourceMapping]
	PermissionsView    *LambdaPermissionsView
	VersionsView       *LambdaItemsView[lambdaVersion]
	LayersView         *LambdaItemsView[lambdaLayer]
	serviceCtx         *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaDetailsPageView(
	lambdaListTable *LambdaListTable,
	lambdaDetailsTable *LambdaDetailsTable,
	logStreamsTable *cloudwatch.LogStreamsTable,
	lambdaInvokeView *LambdaInvokePageView,
	lambdaMetricsView *LambdaMetricsPageView,
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaDetailsPageView {

	var triggersView = NewLambdaTriggersView(serviceCtx)
	var permissionsView = NewLambdaPermissionsView(serviceCtx)
	var versionsView = NewLambdaVersionsView(serviceCtx)
	var layersView = NewLambdaLayersView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(LambdaTabDetails, lambdaDetailsTable, 0, 1, true).
		AddTab(LambdaTabLogSreams, logStreamsTable, 0, 1, true).
		AddTab(LambdaTabTriggers, triggersView, 0, 1, true).
		AddTab(LambdaTabPerms, permissionsView, 0, 1, true).
		AddTab(LambdaTabVersions, versionsView, 0, 1, true).
		AddTab(LambdaTabLayers, layersView, 0, 1, true).
		AddTab(LambdaTabInvoke, lambdaInvokeView, 0, 1, true).
		AddTab(LambdaTabMetrics, lambdaMetricsView, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		lambdaMetricsView.SetVisible(tabName == LambdaTabMetrics)
	})

	const detailsViewSize = 3000
	const tableViewSize = 7000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		lambdaListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &LambdaDetailsPageView{
		ServicePageView: serviceView,
		SelectedLambda:  "",

		LambdaListTable:    lambdaListTable,
		LambdaDetailsTable: lambdaDetailsTable,
		LogStreamsTable:    logStreamsTable,
		LambdaInvokeView:   lambdaInvokeView,
		LambdaMetricsView:  lambdaMetricsView,
		TriggersView:       triggersView,
		PermissionsView:    permissionsView,
		VersionsView:       versionsView,
		LayersView:         layersView,
		serviceCtx:         serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	lambdaDetailsTable.ErrorMessageCallback = errorHandler
	lambdaListTable.ErrorMessageCallback = errorHandler
	triggersView.SetErrorMessageCallback(errorHandler)
	permissionsView.ErrorMessageCallback = errorHandler
	versionsView.SetErrorMessageCallback(errorHandler)
	layersView.SetErrorMessageCallback(errorHandler)

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{lambdaListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *LambdaDetailsPageView) initInputCapture() {
	inst.LambdaListTable.SetSelectionChangedFunc(func(row, column int) {
		var selectedLambda = inst.LambdaListTable.GetSelectedLambda()
		inst.LambdaDetailsTable.RefreshDetails(selectedLambda)
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
	serviceCtx     *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaInvokePageView(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaInvokePageView {
	var payloadInput = core.NewTextArea("Payload", serviceCtx.AppContext)
	var logResults = core.NewSearchableTextView("Logs", serviceCtx.AppContext)
	var responseOutput = core.NewSearchableTextView("Response", serviceCtx.AppContext)
	var invokeButton = tview.NewButton("Invoke")
	var formatButton = tview.NewButton("Format Payload")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(invokeButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	payloadInput.SetTitleExtra("NO LAMBDA SELECTED")

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(payloadInput, 0, 1, true).
		AddItem(buttonsView, 3, 0, false)
	var outputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(responseOutput, 0, 1, false).
		AddItem(logResults, 0, 1, false)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		SetDirection(tview.FlexColumn).
		AddItem(inputColumn, 0, 1, true).
		AddItem(outputColumn, 0, 1, false)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	logResults.ErrorMessageCallback = errorHandler
	responseOutput.ErrorMessageCallback = errorHandler
	payloadInput.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{payloadInput, responseOutput},
			{invokeButton, formatButton, logResults},
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
		serviceCtx:      serviceCtx,
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
	var newText, _ = utils.TryFormatToJson(text)
	inst.responseOutput.SetText(newText, false)
}

func (inst *LambdaInvokePageView) SetSelectedLambda(name string) {
	inst.selectedLambda = name
	inst.payloadInput.SetTitleExtra(name)
}

func (inst *LambdaInvokePageView) Invoke() {
	var logResults = []byte{}
	var responseOutput = []byte{}
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		var payload = make(map[string]any)
		err = json.Unmarshal([]byte(inst.payloadInput.GetText()), &payload)
		if err != nil {
			inst.responseOutput.ErrorMessageCallback("%s", err.Error())
			return
		}

		var data *lambda.InvokeOutput = nil
		data, err = inst.serviceCtx.Api.InvokeLambda(inst.serviceCtx.Ctx, inst.selectedLambda, payload)
		if err != nil {
			inst.responseOutput.ErrorMessageCallback("%s", err.Error())
			return
		}

		logResults, err = base64.StdEncoding.DecodeString(aws.ToString(data.LogResult))
		if err != nil {
			inst.logResults.ErrorMessageCallback("%s", err.Error())
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
	appCtx *core.AppContext,
	api awsapi.LambdaApi,
	cwlApi awsapi.CloudWatchLogsApi,
	cwmApi awsapi.CloudWatchMetricsApi,
) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		lambdaCtx = core.NewServiceViewContext(appCtx, api)

		cwLogsCtx = core.NewServiceViewContext(appCtx, cwlApi)

		cwMetricsCtx = core.NewServiceViewContext(appCtx, cwmApi)

		lambdaInvokeView = NewLambdaInvokePageView(lambdaCtx)

		lambdaMetricsView = NewLambdaMetricsPageView(cwMetricsCtx)

		lambdasDetailsView = NewLambdaDetailsPageView(
			NewLambdasListTable(lambdaCtx),
			NewLambdaDetailsTable(lambdaCtx),
			cloudwatch.NewLogStreamsTable(cwLogsCtx),
			lambdaInvokeView,
			lambdaMetricsView,
			lambdaCtx,
		)
		logEventsView = cloudwatch.NewLogEventsPageView(
			cloudwatch.NewLogEventsTable(cwLogsCtx),
			cwLogsCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.LAMBDA), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Lambdas", lambdasDetailsView, true).
		AddPage("Log Events", logEventsView, true, true)

	serviceRootView.InitPageNavigation()
	var lambdasListTable = lambdasDetailsView.LambdaListTable
	var logStreamsTable = lambdasDetailsView.LogStreamsTable
	var logEventsTable = logEventsView.LogEventsTable

	var lambdaSelectedFunc = func(_, _ int) {
		var logGroup = lambdasListTable.GetSelectedLambdaLogGroup()
		logStreamsTable.SetSelectedLogGroup(logGroup)
		logStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsTable.RefreshStreams(true)

		var selectedLambda = lambdasListTable.GetSelectedLambda()
		var lambdaName = aws.ToString(selectedLambda.FunctionName)
		var lambdaArn = aws.ToString(selectedLambda.FunctionArn)
		lambdasDetailsView.TriggersView.Refresh(lambdaArn)
		lambdasDetailsView.PermissionsView.Refresh(lambdaArn)
		lambdasDetailsView.VersionsView.Refresh(lambdaArn)
		lambdasDetailsView.LayersView.SetItems(lambdaArn, selectedLambda.Layers)
		lambdaInvokeView.SetSelectedLambda(lambdaName)
		lambdaMetricsView.SetSelectedLambda(lambdaName)
	}

	lambdasListTable.SetSelectedFunc(lambdaSelectedFunc)

	serviceRootView.SetResourceOpener("lambda", func(resource arn.ARN) bool {
		_, ok := core.ArnResourceName(resource, "function")
		return ok
	}, func(resource arn.ARN) {
		if name, ok := core.ArnResourceName(resource, "function"); ok {
			serviceRootView.ChangePage(0, nil)
			lambdasListTable.OpenLambda(name, func() { lambdaSelectedFunc(0, 0) })
		}
	})
	lambdasDetailsView.LambdaDetailsTable.SetSelectedFunc(lambdaSelectedFunc)

	logStreamsTable.SetSelectedFunc(func(row, column int) {
		var selectedLogStream = logStreamsTable.GetSelectedLogStream()
		var selectedLogGroup = logStreamsTable.GetSelectedLogGroup()

		logEventsTable.SetSelectedLogGroup(selectedLogGroup)
		logEventsTable.SetSelectedLogStream(selectedLogStream)
		logEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(1, nil)
	})

	logEventsView.InitInputCapture()

	return serviceRootView
}
