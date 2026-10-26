package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogEventsPageView struct {
	*core.ServicePageView
	LogEventsTable       *LogEventsTable
	ExpandedLogsTextArea *core.SearchableTextView
	selectedLogGroup     string
	selectedLogStream    string
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogEventsPageView(
	logEventsTable *LogEventsTable,
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogEventsPageView {

	var expandedLogsView = core.CreateJsonTableDataView(
		serviceViewCtx.AppContext, logEventsTable, 1,
	)

	expandedLogsView.SetJsonHighlighting(true)

	const expandedLogsSize = 7
	const logTableSize = 13

	var mainPage = core.NewResizableView(
		expandedLogsView, expandedLogsSize,
		logEventsTable, logTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{expandedLogsView},
			{logEventsTable},
		},
	)

	logEventsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &LogEventsPageView{
		ServicePageView:      serviceView,
		LogEventsTable:       logEventsTable,
		ExpandedLogsTextArea: expandedLogsView,
		selectedLogGroup:     "",
		selectedLogStream:    "",
		serviceCtx:           serviceViewCtx,
	}
}

func (inst *LogEventsPageView) InitInputCapture() {}

type LogStreamsPageView struct {
	*core.ServicePageView
	LogStreamsTable       *LogStreamsTable
	LogStreamDetailsTable *LogStreamDetailsTable
	serviceCtx            *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogStreamsPageView(
	logStreamDetailsTable *LogStreamDetailsTable,
	logStreamsTable *LogStreamsTable,
	serviceCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogStreamsPageView {

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		AddItem(logStreamDetailsTable, 8, 0, true).
		AddItem(logStreamsTable, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{logStreamDetailsTable},
			{logStreamsTable},
		},
	)

	logStreamsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &LogStreamsPageView{
		ServicePageView:       serviceView,
		LogStreamsTable:       logStreamsTable,
		LogStreamDetailsTable: logStreamDetailsTable,
		serviceCtx:            serviceCtx,
	}
}

func (inst *LogStreamsPageView) InitInputCapture() {
	inst.LogStreamsTable.SetSelectionChangedFunc(func(row, column int) {
		var logStream = inst.LogStreamsTable.GetLogStreamDetail()
		inst.LogStreamDetailsTable.RefreshDetails(logStream)
	})
}

type LogGroupsPageView struct {
	*core.ServicePageView
	LogGroupsTable       *LogGroupsTable
	LogGroupDetailsTable *LogGroupDetailsTable
	selectedLogGroup     string
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogGroupsPageView(
	logGroupDetailsTable *LogGroupDetailsTable,
	logGroupsTable *LogGroupsTable,
	serviceCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogGroupsPageView {

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		AddItem(logGroupDetailsTable, 8, 0, true).
		AddItem(logGroupsTable, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{logGroupDetailsTable},
			{logGroupsTable},
		},
	)

	logGroupsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &LogGroupsPageView{
		ServicePageView:      serviceView,
		LogGroupsTable:       logGroupsTable,
		LogGroupDetailsTable: logGroupDetailsTable,
		selectedLogGroup:     "",
		serviceCtx:           serviceCtx,
	}
}

func (inst *LogGroupsPageView) InitInputCapture() {
	inst.LogGroupsTable.SetSelectionChangedFunc(func(row, column int) {
		var logGroup = inst.LogGroupsTable.GetLogGroupDetail()
		inst.LogGroupDetailsTable.RefreshDetails(logGroup)
	})
}

func NewLogsHomeView(appCtx *core.AppContext, api awsapi.CloudWatchLogsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var logEventsView = NewLogEventsPageView(
		NewLogEventsTable(serviceCtx),
		serviceCtx,
	)
	var logStreamsView = NewLogStreamsPageView(
		NewLogStreamDetailsTable(serviceCtx),
		NewLogStreamsTable(serviceCtx),
		serviceCtx,
	)
	var logGroupsView = NewLogGroupsPageView(
		NewLogGroupDetailsTable(serviceCtx),
		NewLogGroupsTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDWATCH_LOGS_GROUPS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Groups", logGroupsView, true).
		AddPage("Streams", logStreamsView, true, true).
		AddPage("Events", logEventsView, true, true)

	serviceRootView.InitPageNavigation()

	var showStreams = func(logGroup string) {
		logStreamsView.LogStreamsTable.SetSelectedLogGroup(logGroup)
		logStreamsView.LogStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsView.LogStreamsTable.RefreshStreams(true)
		serviceRootView.ChangePage(1, nil)
	}

	logGroupsView.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		showStreams(logGroupsView.LogGroupsTable.GetSelectedLogGroup())
	})

	serviceRootView.SetResourceOpener("logs", func(resource arn.ARN) {
		if logGroup, ok := core.ArnResourceName(resource, "log-group"); ok {
			showStreams(logGroup)
		}
	})

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		var logStream = logStreamsView.LogStreamsTable.GetSelectedLogStream()
		var logGroup = logStreamsView.LogStreamsTable.GetSelectedLogGroup()

		logEventsView.LogEventsTable.SetSelectedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSelectedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, nil)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()
	logGroupsView.InitInputCapture()

	return serviceRootView
}
