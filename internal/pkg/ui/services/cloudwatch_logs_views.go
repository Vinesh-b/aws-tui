package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogEventsPageView struct {
	*core.ServicePageView
	LogEventsTable       *tables.LogEventsTable
	ExpandedLogsTextArea *core.SearchableTextView
	selectedLogGroup     string
	selectedLogStream    string
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogEventsPageView(
	logEventsTable *tables.LogEventsTable,
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogEventsPageView {

	var expandedLogsView = core.CreateJsonTableDataView(
		serviceViewCtx.AppContext, logEventsTable, 1,
	)

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
	LogStreamsTable       *tables.LogStreamsTable
	LogStreamDetailsTable *tables.LogStreamDetailsTable
	serviceCtx            *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogStreamsPageView(
	logStreamDetailsTable *tables.LogStreamDetailsTable,
	logStreamsTable *tables.LogStreamsTable,
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
	LogGroupsTable       *tables.LogGroupsTable
	LogGroupDetailsTable *tables.LogGroupDetailsTable
	selectedLogGroup     string
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogGroupsPageView(
	logGroupDetailsTable *tables.LogGroupDetailsTable,
	logGroupsTable *tables.LogGroupsTable,
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

func NewLogsHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer appCtx.Theme.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(appCtx.Logger)
	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var logEventsView = NewLogEventsPageView(
		tables.NewLogEventsTable(serviceCtx),
		serviceCtx,
	)
	var logStreamsView = NewLogStreamsPageView(
		tables.NewLogStreamDetailsTable(serviceCtx),
		tables.NewLogStreamsTable(serviceCtx),
		serviceCtx,
	)
	var logGroupsView = NewLogGroupsPageView(
		tables.NewLogGroupDetailsTable(serviceCtx),
		tables.NewLogGroupsTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(CLOUDWATCH_LOGS_GROUPS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Groups", logGroupsView, true).
		AddPage("Streams", logStreamsView, true, true).
		AddPage("Events", logEventsView, true, true)

	serviceRootView.InitPageNavigation()

	logGroupsView.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		var logGroup = logGroupsView.LogGroupsTable.GetSeletedLogGroup()

		logStreamsView.LogStreamsTable.SetSeletedLogGroup(logGroup)
		logStreamsView.LogStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsView.LogStreamsTable.RefreshStreams(true)
		serviceRootView.ChangePage(1, nil)
	})

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		var logStream = logStreamsView.LogStreamsTable.GetSeletedLogStream()
		var logGroup = logStreamsView.LogStreamsTable.GetSeletedLogGroup()

		logEventsView.LogEventsTable.SetSeletedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, nil)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()
	logGroupsView.InitInputCapture()

	return serviceRootView
}
