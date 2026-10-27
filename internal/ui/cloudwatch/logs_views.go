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

// The pages of the logs. Events are between the two ways to find them, the
// streams of a group and the records of an Insights query.
const (
	logsPageGroups = iota
	logsPageStreams
	logsPageEvents
	logsPageInsights
)

// NewLogsHomeView has the pages to go from the log groups down to their events,
// and to query the log groups that are marked on the way with Insights
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
	var logGroupsTable = logGroupsView.LogGroupsTable
	var insightsView = NewLogsInsightsPageView(
		NewSelectedGroupsTable(serviceCtx, logGroupsTable.Marks()),
		NewInsightsQueryResultsTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDWATCH_LOGS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Groups", logGroupsView, true).
		AddPage("Streams", logStreamsView, true, true).
		AddPage("Events", logEventsView, true, true).
		AddPage("Insights", insightsView, true, true)

	serviceRootView.InitPageNavigation()

	var showStreams = func(logGroup string) {
		logStreamsView.LogStreamsTable.SetSelectedLogGroup(logGroup)
		logStreamsView.LogStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsView.LogStreamsTable.RefreshStreams(true)
		serviceRootView.ChangePage(logsPageStreams, nil)
	}

	var showEvents = func(logGroup string, logStream string) {
		logEventsView.LogEventsTable.SetSelectedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSelectedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(logsPageEvents, nil)
	}

	logGroupsTable.SetSelectedFunc(func(row, column int) {
		showStreams(logGroupsTable.GetSelectedLogGroup())
	})

	// Without marks the log group the cursor is on is the one to query
	logGroupsTable.SetQueryFunc(func() {
		if logGroupsTable.Marks().Len() == 0 {
			logGroupsTable.Marks().Add(logGroupsTable.GetSelectedLogGroup())
		}
		serviceRootView.ChangePage(logsPageInsights, nil)
	})

	insightsView.SelectLogGroups = func() {
		serviceRootView.ChangePage(logsPageGroups, nil)
		logGroupsView.DisplayMessage(
			core.InfoPrompt,
			"No log groups are marked. Mark them with %s, then press %s to query them",
			core.FormatRune(core.APP_KEY_BINDINGS.Mark),
			core.FormatRune(core.APP_KEY_BINDINGS.TableQuery),
		)
	}

	serviceRootView.SetResourceOpener("logs", func(resource arn.ARN) {
		if logGroup, ok := core.ArnResourceName(resource, "log-group"); ok {
			showStreams(logGroup)
		}
	})

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		showEvents(
			logStreamsView.LogStreamsTable.GetSelectedLogGroup(),
			logStreamsView.LogStreamsTable.GetSelectedLogStream(),
		)
	})

	insightsView.ResultsTable.SetSelectedFunc(func(row, column int) {
		insightsView.OpenRecord(row, showEvents)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()
	logGroupsView.InitInputCapture()

	return serviceRootView
}
