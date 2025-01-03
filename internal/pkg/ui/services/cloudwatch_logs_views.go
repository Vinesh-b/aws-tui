package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogEventsPageView struct {
	*core.ServicePageView
	LogEventsTable       *tables.LogEventsTable
	ExpandedLogsTextArea *tview.TextArea
	selectedLogGroup     string
	selectedLogStream    string
	app                  *tview.Application
	api                  *awsapi.CloudWatchLogsApi
}

func NewLogEventsPageView(
	logEventsTable *tables.LogEventsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogEventsPageView {

	var expandedLogsView = core.CreateJsonTableDataView(app, logEventsTable, 1)

	const expandedLogsSize = 7
	const logTableSize = 13

	var mainPage = core.NewResizableView(
		expandedLogsView, expandedLogsSize,
		logEventsTable, logTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			logEventsTable,
			expandedLogsView,
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
		app:                  app,
		api:                  api,
	}
}

func (inst *LogEventsPageView) InitInputCapture() {}

type LogStreamsPageView struct {
	*core.ServicePageView
	LogStreamsTable *tables.LogStreamsTable
	app             *tview.Application
	api             *awsapi.CloudWatchLogsApi
}

func NewLogStreamsPageView(
	logStreamsTable *tables.LogStreamsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogStreamsPageView {

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(logStreamsTable, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			logStreamsTable,
		},
	)

	logStreamsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &LogStreamsPageView{
		ServicePageView: serviceView,
		LogStreamsTable: logStreamsTable,
		app:             app,
		api:             api,
	}
}

func (inst *LogStreamsPageView) InitInputCapture() {}

type LogGroupsPageView struct {
	*core.ServicePageView
	LogGroupsTable   *tables.LogGroupsTable
	selectedLogGroup string
	app              *tview.Application
	api              *awsapi.CloudWatchLogsApi
}

func NewLogGroupsPageView(
	logGroupsTable *tables.LogGroupsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsPageView {

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(logGroupsTable, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			logGroupsTable,
		},
	)

	logGroupsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &LogGroupsPageView{
		ServicePageView:  serviceView,
		LogGroupsTable:   logGroupsTable,
		selectedLogGroup: "",
		app:              app,
		api:              api,
	}
}

func (inst *LogGroupsPageView) InitInputCapture() {}

func NewLogsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(config, logger)
	var logEventsView = NewLogEventsPageView(
		tables.NewLogEventsTable(app, api, logger),
		app, api, logger,
	)
	var logStreamsView = NewLogStreamsPageView(
		tables.NewLogStreamsTable(app, api, logger),
		app, api, logger,
	)
	var logGroupsView = NewLogGroupsPageView(
		tables.NewLogGroupsTable(app, api, logger),
		app, api, logger,
	)

	var serviceRootView = core.NewServiceRootView(app, string(CLOUDWATCH_LOGS_GROUPS))

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
		serviceRootView.ChangePage(1, logStreamsView.LogStreamsTable)
	})

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		var logStream = logStreamsView.LogStreamsTable.GetSeletedLogStream()
		var logGroup = logStreamsView.LogStreamsTable.GetSeletedLogGroup()

		logEventsView.LogEventsTable.SetSeletedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()
	logGroupsView.InitInputCapture()

	return serviceRootView
}
