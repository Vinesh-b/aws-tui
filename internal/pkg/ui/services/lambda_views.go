package services

import (
	"encoding/base64"
	"encoding/json"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"
	"aws-tui/internal/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// For some reason the lambda service uses a map[string]string for Tag unlike
// All other services which have a specific Tag type.
type LambdaTag struct {
	Key   string
	Value string
}

type LambdaTabName = string

const (
	LambdaTabDetails   LambdaTabName = "Details"
	LambdaTabLogSreams LambdaTabName = "Log Streams"
	LambdaTabEnvVars   LambdaTabName = "Environment Vars"
	LambdaTabVpcConfig LambdaTabName = "VPC Config"
	LambdaTabTags      LambdaTabName = "Tags"
)

type LambdaDetailsPageView struct {
	*core.ServicePageView
	SelectedLambda     string
	LambdaListTable    *tables.LambdaListTable
	LambdaDetailsTable *tables.LambdaDetailsTable
	LambdaEnvVarsTable *tables.LambdaEnvVarsTable
	LambdaVpcConfTable *tables.LambdaVpcConfigTable
	LambdaTagsTable    *tables.TagsTable[LambdaTag, awsapi.LambdaApi]
	LogStreamsTable    *tables.LogStreamsTable
	serviceCtx         *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaDetailsPageView(
	lambdaListTable *tables.LambdaListTable,
	lambdaDetailsTable *tables.LambdaDetailsTable,
	lambdaEnvVarsTable *tables.LambdaEnvVarsTable,
	lambdaVpcConfTable *tables.LambdaVpcConfigTable,
	logStreamsTable *tables.LogStreamsTable,
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaDetailsPageView {

	var lambdaTagsTable = tables.NewTagsTable[LambdaTag](serviceCtx).
		SetExtractKeyValFunc(func(lt LambdaTag) (k string, v string) {
			return lt.Key, lt.Value
		}).
		SetGetTagsFunc(func() ([]LambdaTag, error) {
			var tagsMap, err = serviceCtx.Api.ListTags(
				aws.ToString(lambdaListTable.GetSeletedLambda().FunctionArn),
			)
			var tags = []LambdaTag{}
			for k, v := range tagsMap {
				tags = append(tags, LambdaTag{Key: k, Value: v})
			}

			return tags, err
		})

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(LambdaTabDetails, lambdaDetailsTable, 0, 1, true).
		AddTab(LambdaTabLogSreams, logStreamsTable, 0, 1, true).
		AddTab(LambdaTabEnvVars, lambdaEnvVarsTable, 0, 1, true).
		AddTab(LambdaTabVpcConfig, lambdaVpcConfTable, 0, 1, true).
		AddTab(LambdaTabTags, lambdaTagsTable, 0, 1, true)

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
		LambdaEnvVarsTable: lambdaEnvVarsTable,
		LambdaVpcConfTable: lambdaVpcConfTable,
		LambdaTagsTable:    lambdaTagsTable,
		LogStreamsTable:    logStreamsTable,
		serviceCtx:         serviceCtx,
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
			{tabView.GetTabDisplayView()},
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
	serviceCtx     *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaInvokePageView(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaInvokePageView {
	var payloadInput = core.NewTextArea("Payload", serviceCtx.Theme)
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

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
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
			inst.responseOutput.ErrorMessageCallback(err.Error())
			return
		}

		var data *lambda.InvokeOutput = nil
		data, err = inst.serviceCtx.Api.InvokeLambda(inst.selectedLambda, payload)
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

type FloatingLambdaInvoke struct {
	*tview.Flex
	Input *LambdaInvokePageView
}

func NewFloatingLambdaInvoke(
	input *LambdaInvokePageView,
) *FloatingLambdaInvoke {
	return &FloatingLambdaInvoke{
		Flex:  core.FloatingViewRelative("Invoke", input, 90, 90),
		Input: input,
	}
}

func (inst *FloatingLambdaInvoke) GetLastFocusedView() tview.Primitive {
	return inst.Input.GetLastFocusedView()
}

func NewLambdaHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api       = awsapi.NewLambdaApi(appCtx.Logger)
		lambdaCtx = core.NewServiceViewContext(appCtx, api)

		cwl_api   = awsapi.NewCloudWatchLogsApi(appCtx.Logger)
		cwLogsCtx = core.NewServiceViewContext(appCtx, cwl_api)

		lambdasDetailsView = NewLambdaDetailsPageView(
			tables.NewLambdasListTable(lambdaCtx),
			tables.NewLambdaDetailsTable(lambdaCtx),
			tables.NewLambdaEnvVarsTable(lambdaCtx),
			tables.NewLambdaVpcConfigTable(lambdaCtx),
			tables.NewLogStreamsTable(cwLogsCtx),
			lambdaCtx,
		)
		logEventsView = NewLogEventsPageView(
			tables.NewLogEventsTable(cwLogsCtx),
			cwLogsCtx,
		)
		lambdaInvokeView = NewLambdaInvokePageView(
			lambdaCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(LAMBDA), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Lambdas", lambdasDetailsView, true).
		AddPage("Log Events", logEventsView, true, true)

	serviceRootView.AddKeyToggleOverlay(
		"INVOKE", NewFloatingLambdaInvoke(lambdaInvokeView),
		tcell.KeyCtrlX, false,
	)

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
		lambdaTagsTable.RefreshDetails()
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
