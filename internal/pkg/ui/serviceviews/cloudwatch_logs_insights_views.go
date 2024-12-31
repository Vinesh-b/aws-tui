package serviceviews

import (
	"log"
	"strings"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogGroupsSelectionPageView struct {
	*core.ServicePageView
	LogGroupsTable     *LogGroupsTable
	SeletedGroupsTable *SelectedGroupsTable
	SearchInput        *tview.InputField
	selectedGroups     StringSet
	app                *tview.Application
	api                *awsapi.CloudWatchLogsApi
}

func NewLogGroupsSelectionPageView(
	selectedGroupsTable *SelectedGroupsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsSelectionPageView {

	var logGroupsView = NewLogGroupsPageView(
		NewLogGroupsTable(app, api, logger),
		app, api, logger)
	logGroupsView.InitInputCapture()

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.
		AddItem(selectedGroupsTable, 0, 1, false).
		AddItem(logGroupsView.LogGroupsTable, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			logGroupsView.LogGroupsTable,
			selectedGroupsTable,
		},
	)

	return &LogGroupsSelectionPageView{
		ServicePageView:    serviceView,
		SeletedGroupsTable: selectedGroupsTable,
		LogGroupsTable:     logGroupsView.LogGroupsTable,
		selectedGroups:     StringSet{},
		app:                app,
		api:                api,
	}
}

func (inst *LogGroupsSelectionPageView) InitInputCapture() {
	inst.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var logGroup = inst.LogGroupsTable.GetSeletedLogGroup()
		inst.SeletedGroupsTable.AddLogGroup(logGroup)
		inst.SeletedGroupsTable.RefreshSelectedGroups()
	})

}

type InsightsQueryResultsPageView struct {
	*core.ServicePageView
	QueryResultsTable *InsightsQueryResultsTable
	ExpandedResult    *tview.TextArea
	app               *tview.Application
	api               *awsapi.CloudWatchLogsApi
	selectedLogGroups *[]string
}

func NewInsightsQueryResultsPageView(
	insightsQueryResultsTable *InsightsQueryResultsTable,
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsPageView {
	var expandedResultView = core.CreateExpandedLogView(
		app, insightsQueryResultsTable, -1, core.DATA_TYPE_STRING,
	)

	const expandedLogsSize = 5
	const resultsTableSize = 10
	const queryViewSize = 9

	var resizableView = core.NewResizableView(
		expandedResultView, expandedLogsSize,
		insightsQueryResultsTable, resultsTableSize,
		tview.FlexRow,
	)

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resizableView, 0, 1, true)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			insightsQueryResultsTable,
			expandedResultView,
		},
	)

	insightsQueryResultsTable.ErrorMessageCallback = func(text string) {
		serviceView.SetAndDisplayError(text)
	}

	return &InsightsQueryResultsPageView{
		ServicePageView:   serviceView,
		QueryResultsTable: insightsQueryResultsTable,
		ExpandedResult:    expandedResultView,
		app:               app,
		api:               api,
	}
}

func (inst *InsightsQueryResultsPageView) InitInputCapture() {}

func NewLogsInsightsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(config, logger)
	var insightsResultsView = NewInsightsQueryResultsPageView(
		NewInsightsQueryResultsTable(app, api, logger),
		app, api, logger,
	)
	var groupSelectionView = NewLogGroupsSelectionPageView(
		NewSelectedGroupsTable(app, api, logger),
		app, api, logger,
	)
	var logEventsView = NewLogEventsPageView(
		NewLogEventsTable(app, api, logger),
		app, api, logger,
	)

	var serviceRootView = core.NewServiceRootView(app, string(CLOUDWATCH_LOGS_INSIGHTS))

	serviceRootView.
		AddAndSwitchToPage("GroupsSelection", groupSelectionView, true).
		AddPage("Query", insightsResultsView, true, true).
		AddPage("LogEvents", logEventsView, true, true)

	serviceRootView.InitPageNavigation()

	groupSelectionView.SeletedGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var logGroups = groupSelectionView.SeletedGroupsTable.GetAllLogGroups()
		insightsResultsView.QueryResultsTable.SetSelectedLogGroups(logGroups)
		serviceRootView.ChangePage(1, insightsResultsView.QueryResultsTable)
	})

	groupSelectionView.InitInputCapture()
	insightsResultsView.InitInputCapture()

	var recordPtr = ""
	insightsResultsView.QueryResultsTable.SetSelectedFunc(func(row, column int) {
		recordPtr = insightsResultsView.QueryResultsTable.GetRecordPtr(row)
		var record, err = api.GetInsightsLogRecord(recordPtr)
		if err != nil {
			logEventsView.LogEventsTable.ErrorMessageCallback(err.Error())
		}

		var logStream = record["@logStream"]
		var _, logGroup, _ = strings.Cut(record["@log"], ":")

		logEventsView.LogEventsTable.SetSeletedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSeletedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	return serviceRootView
}
