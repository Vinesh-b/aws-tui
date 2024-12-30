package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogEventsPageView struct {
	*core.ServicePageView
	LogEventsTable       *LogEventsTable
	ExpandedLogsTextArea *tview.TextArea
	selectedLogGroup     string
	selectedLogStream    string
	app                  *tview.Application
	api                  *awsapi.CloudWatchLogsApi
}

func NewLogEventsPageView(
	logEventsTable *LogEventsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogEventsPageView {

	var expandedLogsView = core.CreateExpandedLogView(
		app, logEventsTable, 1, core.DATA_TYPE_STRING,
	)

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

func (inst *LogEventsPageView) InitInputCapture() {
	inst.LogEventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.LogEventsTable.RefreshLogEvents(true)
		case tcell.KeyCtrlN:
			inst.LogEventsTable.RefreshLogEvents(false)
		}

		return event
	})
}

type LogStreamsPageView struct {
	*core.ServicePageView
	LogStreamsTable *LogStreamsTable
	app             *tview.Application
	api             *awsapi.CloudWatchLogsApi
}

func NewLogStreamsPageView(
	logStreamsTable *LogStreamsTable,
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

	return &LogStreamsPageView{
		ServicePageView: serviceView,
		LogStreamsTable: logStreamsTable,
		app:             app,
		api:             api,
	}
}

func (inst *LogStreamsPageView) InitInputCapture() {
	inst.LogStreamsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.LogStreamsTable.SetLogStreamSearchPrefix(inst.LogStreamsTable.GetSearchText())
			inst.LogStreamsTable.RefreshStreams(true)
			inst.app.SetFocus(inst.LogStreamsTable)
		}
	})

	inst.LogStreamsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.LogStreamsTable.RefreshStreams(true)
		case tcell.KeyCtrlN:
			inst.LogStreamsTable.RefreshStreams(false)
		}
		return event
	})
}

type LogGroupsPageView struct {
	*core.ServicePageView
	LogGroupsTable   *LogGroupsTable
	selectedLogGroup string
	app              *tview.Application
	api              *awsapi.CloudWatchLogsApi
}

func NewLogGroupsPageView(
	logGroupsTable *LogGroupsTable,
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

	return &LogGroupsPageView{
		ServicePageView:  serviceView,
		LogGroupsTable:   logGroupsTable,
		selectedLogGroup: "",
		app:              app,
		api:              api,
	}
}

func (inst *LogGroupsPageView) InitInputCapture() {
	inst.LogGroupsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.LogGroupsTable.RefreshLogGroups(inst.LogGroupsTable.GetSearchText())
			inst.app.SetFocus(inst.LogGroupsTable)
		}
	})

	inst.LogGroupsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
	var logEventsView = NewLogEventsPageView(
		NewLogEventsTable(app, api, logger),
		app, api, logger,
	)
	var logStreamsView = NewLogStreamsPageView(
		NewLogStreamsTable(app, api, logger),
		app, api, logger,
	)
	var logGroupsView = NewLogGroupsPageView(
		NewLogGroupsTable(app, api, logger),
		app, api, logger,
	)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView, true, true).
		AddPage("Streams", logStreamsView, true, true).
		AddAndSwitchToPage("Groups", logGroupsView, true)

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
