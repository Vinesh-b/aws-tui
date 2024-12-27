package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogEventsPage struct {
	LogEventsTable       *LogEventsTable
	ExpandedLogsTextArea *tview.TextArea
	selectedLogGroup     string
	selectedLogStream    string
	RootView             *tview.Flex
	app                  *tview.Application
	api                  *awsapi.CloudWatchLogsApi
}

func NewLogEventsPage(
	logEventsTable *LogEventsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogEventsPage {

	var expandedLogsView = core.CreateExpandedLogView(
		app, logEventsTable.Table, 1, core.DATA_TYPE_STRING,
	)

	const expandedLogsSize = 7
	const logTableSize = 13

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandedLogsView, 0, expandedLogsSize, false).
		AddItem(logEventsTable.RootView, 0, logTableSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		expandedLogsView, logEventsTable.RootView,
		expandedLogsSize, logTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			logEventsTable.RootView,
			expandedLogsView,
		},
	)

	return &LogEventsPage{
		LogEventsTable:       logEventsTable,
		ExpandedLogsTextArea: expandedLogsView,
		RootView:             serviceView.RootView,
		selectedLogGroup:     "",
		selectedLogStream:    "",
		app:                  app,
		api:                  api,
	}
}

func (inst *LogEventsPage) InitInputCapture() {
	inst.LogEventsTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.LogEventsTable.RefreshLogEvents(true)
		case tcell.KeyCtrlN:
			inst.LogEventsTable.RefreshLogEvents(false)
		}

		return event
	})
}

type LogStreamsPage struct {
	LogStreamsTable *LogStreamsTable
	RootView        *tview.Flex
	app             *tview.Application
	api             *awsapi.CloudWatchLogsApi
}

func NewLogStreamsPage(
	logStreamsTable *LogStreamsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogStreamsPage {

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logStreamsTable.RootView, 0, 1, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.InitViewNavigation(
		[]core.View{
			logStreamsTable.RootView,
		},
	)

	return &LogStreamsPage{
		LogStreamsTable: logStreamsTable,
		RootView:        serviceView.RootView,
		app:             app,
		api:             api,
	}
}

func (inst *LogStreamsPage) InitInputCapture() {
	inst.LogStreamsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.LogStreamsTable.SetLogStreamSearchPrefix(inst.LogStreamsTable.GetSearchText())
			inst.LogStreamsTable.RefreshStreams(true)
			inst.app.SetFocus(inst.LogStreamsTable.Table)
		}
	})

	inst.LogStreamsTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.LogStreamsTable.RefreshStreams(true)
		case tcell.KeyCtrlN:
			inst.LogStreamsTable.RefreshStreams(false)
		}
		return event
	})
}

type LogGroupsPage struct {
	LogGroupsTable   *LogGroupsTable
	RootView         *tview.Flex
	selectedLogGroup string
	app              *tview.Application
	api              *awsapi.CloudWatchLogsApi
}

func NewLogGroupsPage(
	logGroupsTable *LogGroupsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsPage {

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logGroupsTable.RootView, 0, 1, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.InitViewNavigation(
		[]core.View{
			logGroupsTable.Table,
		},
	)

	return &LogGroupsPage{
		LogGroupsTable:   logGroupsTable,
		RootView:         serviceView.RootView,
		selectedLogGroup: "",
		app:              app,
		api:              api,
	}
}

func (inst *LogGroupsPage) InitInputCapture() {
	inst.LogGroupsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.LogGroupsTable.RefreshLogGroups(inst.LogGroupsTable.GetSearchText())
			inst.app.SetFocus(inst.LogGroupsTable.Table)
		}
	})

	inst.LogGroupsTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.LogGroupsTable.RefreshLogGroups(inst.LogGroupsTable.GetSearchText())
		}
		return event
	})

}

func NewLogsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(config, logger)
	var logEventsView = NewLogEventsPage(
		NewLogEventsTable(app, api, logger),
		app, api, logger,
	)
	var logStreamsView = NewLogStreamsPage(
		NewLogStreamsTable(app, api, logger),
		app, api, logger,
	)
	var logGroupsView = NewLogGroupsPage(
		NewLogGroupsTable(app, api, logger),
		app, api, logger,
	)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView.RootView, true, true).
		AddPage("Streams", logStreamsView.RootView, true, true).
		AddAndSwitchToPage("Groups", logGroupsView.RootView, true)

	var orderedPages = []string{
		"Groups",
		"Streams",
		"Events",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDWATCH_LOGS_GROUPS), pages, orderedPages).Init()

	logGroupsView.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		var logGroup = logGroupsView.LogGroupsTable.GetSeletedLogGroup()

		logStreamsView.LogStreamsTable.SetSeletedLogGroup(logGroup)
		logStreamsView.LogStreamsTable.SetLogStreamSearchPrefix("")
		logStreamsView.LogStreamsTable.RefreshStreams(true)
		serviceRootView.ChangePage(1, logStreamsView.LogStreamsTable.Table)
	})

	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		var logStream = logStreamsView.LogStreamsTable.GetSeletedLogStream()
		var logGroup = logStreamsView.LogStreamsTable.GetSeletedLogGroup()

		logEventsView.LogEventsTable.SetSeletedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable.Table)
	})

	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()
	logGroupsView.InitInputCapture()

	return serviceRootView.RootView
}
