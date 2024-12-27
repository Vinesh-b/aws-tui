package serviceviews

import (
	"log"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogGroupsSelectionPage struct {
	LogGroupsTable     *LogGroupsTable
	SeletedGroupsTable *SelectedGroupsTable
	SearchInput        *tview.InputField
	RootView           *tview.Flex
	selectedGroups     StringSet
	app                *tview.Application
	api                *awsapi.CloudWatchLogsApi
}

func NewLogGroupsSelectionPage(
	selectedGroupsTable *SelectedGroupsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsSelectionPage {

	var logGroupsView = NewLogGroupsPage(
		NewLogGroupsTable(app, api, logger),
		app, api, logger)
	logGroupsView.InitInputCapture()

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(selectedGroupsTable.RootView, 0, 1, false).
		AddItem(logGroupsView.LogGroupsTable.RootView, 0, 1, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.InitViewNavigation(
		[]core.View{
			logGroupsView.LogGroupsTable.RootView,
			selectedGroupsTable.RootView,
		},
	)

	return &LogGroupsSelectionPage{
		SeletedGroupsTable: selectedGroupsTable,
		LogGroupsTable:     logGroupsView.LogGroupsTable,
		RootView:           serviceView.RootView,
		selectedGroups:     StringSet{},
		app:                app,
		api:                api,
	}
}

func (inst *LogGroupsSelectionPage) InitInputCapture() {
	inst.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var logGroup = inst.LogGroupsTable.GetSeletedLogGroup()
		inst.SeletedGroupsTable.AddLogGroup(logGroup)
		inst.SeletedGroupsTable.RefreshSelectedGroups()
	})

}

type InsightsQueryResultsPage struct {
	QueryResultsTable   *InsightsQueryResultsTable
	ExpandedResult      *tview.TextArea
	QueryInput          *tview.TextArea
	QueryStartDateInput *tview.InputField
	QueryEndDateInput   *tview.InputField
	RunQueryButton      *tview.Button
	RootView            *tview.Flex
	app                 *tview.Application
	api                 *awsapi.CloudWatchLogsApi
	selectedLogGroups   *[]string
}

func NewInsightsQueryResultsPage(
	insightsQueryResultsTable *InsightsQueryResultsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsPage {

	var queryInputView = core.CreateTextArea("Query")
	queryInputView.SetText(
		"fields @timestamp, @message, @log\n"+
			"| sort @timestamp desc\n"+
			"| limit 1000\n",
		false,
	)

	var runQueryButton = tview.NewButton("Run Query")
	var startDateInput = tview.NewInputField().SetFieldWidth(20).SetLabel("Start Date ")
	var endDateInput = tview.NewInputField().SetFieldWidth(20).SetLabel("End Date   ")
	var timeNow = time.Now()
	startDateInput.SetText(timeNow.Add(time.Duration(-time.Hour)).Format(time.DateTime))
	endDateInput.SetText(timeNow.Format(time.DateTime))

	var queryRunView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(startDateInput, 1, 0, false).
		AddItem(endDateInput, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(runQueryButton, 1, 0, false)
	queryRunView.SetBorder(true)

	var queryView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(queryInputView, 0, 1, false).
		AddItem(queryRunView, 34, 0, false)

	var expandedResultView = core.CreateExpandedLogView(
		app, insightsQueryResultsTable.Table, -1, core.DATA_TYPE_STRING,
	)

	const expandedLogsSize = 5
	const resultsTableSize = 10
	const queryViewSize = 9

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandedResultView, 0, expandedLogsSize, false).
		AddItem(insightsQueryResultsTable.RootView, 0, resultsTableSize, true).
		AddItem(queryView, queryViewSize, 0, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		expandedResultView, insightsQueryResultsTable.RootView,
		expandedLogsSize, resultsTableSize,
	)

	serviceView.InitViewTabNavigation(
		queryRunView,
		[]core.View{
			startDateInput,
			endDateInput,
			runQueryButton,
		})

	serviceView.InitViewNavigation(
		[]core.View{
			queryRunView,
			queryInputView,
			insightsQueryResultsTable.RootView,
			expandedResultView,
		},
	)

	return &InsightsQueryResultsPage{
		QueryResultsTable:   insightsQueryResultsTable,
		QueryInput:          queryInputView,
		ExpandedResult:      expandedResultView,
		QueryStartDateInput: startDateInput,
		QueryEndDateInput:   endDateInput,
		RunQueryButton:      runQueryButton,
		RootView:            serviceView.RootView,
		app:                 app,
		api:                 api,
	}
}

func (inst *InsightsQueryResultsPage) InitInputCapture() {
	inst.QueryResultsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyCtrlR:
			inst.QueryResultsTable.RefreshResults()
		}
	})

	inst.QueryResultsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.QueryResultsTable.RefreshResults()
		}

		return event
	})

	inst.RunQueryButton.SetSelectedFunc(func() {
		var layout = "2006-01-02 15:04:05"
		var startTime, err_1 = time.Parse(layout, inst.QueryStartDateInput.GetText())
		var endTime, err_2 = time.Parse(layout, inst.QueryEndDateInput.GetText())

		if err_1 != nil || err_2 != nil {
			return
		}

		var queryIdChan = make(chan string, 1)
		go func() {
			queryIdChan <- inst.api.StartInightsQuery(
				*inst.selectedLogGroups,
				startTime,
				endTime,
				inst.QueryInput.GetText(),
			)
		}()

		go func() {
			inst.QueryResultsTable.SetQueryId(<-queryIdChan)
			inst.QueryResultsTable.RefreshResults()
		}()
	})
}

func (inst *InsightsQueryResultsPage) InitSearchInputBuffer(selectedGroups *[]string) {
	inst.selectedLogGroups = selectedGroups
}

func NewLogsInsightsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(config, logger)
	var insightsResultsView = NewInsightsQueryResultsPage(
		NewInsightsQueryResultsTable(app, api, logger),
		app, api, logger,
	)
	var groupSelectionView = NewLogGroupsSelectionPage(
		NewSelectedGroupsTable(app, api, logger),
		app, api, logger,
	)
	var logEventsView = NewLogEventsPage(
		NewLogEventsTable(app, api, logger),
		app, api, logger,
	)

	var pages = tview.NewPages().
		AddPage("LogEvents", logEventsView.RootView, true, true).
		AddPage("Query", insightsResultsView.RootView, true, true).
		AddAndSwitchToPage("GroupsSelection", groupSelectionView.RootView, true)

	var orderedPages = []string{
		"GroupsSelection",
		"Query",
		"LogEvents",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDWATCH_LOGS_INSIGHTS), pages, orderedPages).Init()

	var logGroups []string
	groupSelectionView.SeletedGroupsTable.Table.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		logGroups = groupSelectionView.SeletedGroupsTable.GetAllLogGroups()
		serviceRootView.ChangePage(1, insightsResultsView.QueryInput)
	})

	groupSelectionView.InitInputCapture()
	insightsResultsView.InitInputCapture()
	insightsResultsView.InitSearchInputBuffer(&logGroups)

	var recordPtr = ""
	insightsResultsView.QueryResultsTable.SetSelectedFunc(func(row, column int) {
		recordPtr = insightsResultsView.QueryResultsTable.GetRecordPtr(row)
		var record = api.GetInsightsLogRecord(recordPtr)

		var logStream = record["@logStream"]
		var _, logGroup, _ = strings.Cut(record["@log"], ":")

		logEventsView.LogEventsTable.SetSeletedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable.Table)
	})

	return serviceRootView.RootView
}
