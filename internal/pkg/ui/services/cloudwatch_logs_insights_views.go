package services

import (
	"strings"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"
	"aws-tui/internal/pkg/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogGroupsSelectionPageView struct {
	*core.ServicePageView
	LogGroupsTable     *tables.LogGroupsTable
	SeletedGroupsTable *tables.SelectedGroupsTable
	SearchInput        *tview.InputField
	selectedGroups     utils.StringSet
	serviceCtx         *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogGroupsSelectionPageView(
	selectedGroupsTable *tables.SelectedGroupsTable,
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogGroupsSelectionPageView {

	var logGroupsView = NewLogGroupsPageView(
		tables.NewLogGroupDetailsTable(serviceViewCtx),
		tables.NewLogGroupsTable(serviceViewCtx),
		serviceViewCtx,
	)
	logGroupsView.InitInputCapture()

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.
		AddItem(selectedGroupsTable, 0, 1, false).
		AddItem(logGroupsView.LogGroupsTable, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{selectedGroupsTable},
			{logGroupsView.LogGroupsTable},
		},
	)

	return &LogGroupsSelectionPageView{
		ServicePageView:    serviceView,
		SeletedGroupsTable: selectedGroupsTable,
		LogGroupsTable:     logGroupsView.LogGroupsTable,
		selectedGroups:     utils.StringSet{},
		serviceCtx:         serviceViewCtx,
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
	QueryResultsTable *tables.InsightsQueryResultsTable
	ExpandedResult    *core.SearchableTextView
	selectedLogGroups *[]string
	serviceCtx        *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewInsightsQueryResultsPageView(
	insightsQueryResultsTable *tables.InsightsQueryResultsTable,
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *InsightsQueryResultsPageView {
	var expandedResultView = core.CreateJsonTableDataView(
		serviceViewCtx.AppContext, insightsQueryResultsTable, -1,
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

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{expandedResultView},
			{insightsQueryResultsTable},
		},
	)

	insightsQueryResultsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &InsightsQueryResultsPageView{
		ServicePageView:   serviceView,
		QueryResultsTable: insightsQueryResultsTable,
		ExpandedResult:    expandedResultView,
		serviceCtx:        serviceViewCtx,
	}
}

func (inst *InsightsQueryResultsPageView) InitInputCapture() {}

func NewLogsInsightsHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer appCtx.Theme.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(appCtx.Logger)
	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var insightsResultsView = NewInsightsQueryResultsPageView(
		tables.NewInsightsQueryResultsTable(serviceCtx),
		serviceCtx,
	)
	var groupSelectionView = NewLogGroupsSelectionPageView(
		tables.NewSelectedGroupsTable(serviceCtx),
		serviceCtx,
	)
	var logEventsView = NewLogEventsPageView(
		tables.NewLogEventsTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(CLOUDWATCH_LOGS_INSIGHTS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("GroupsSelection", groupSelectionView, true).
		AddPage("Query", insightsResultsView, true, true).
		AddPage("LogEvents", logEventsView, true, true)

	serviceRootView.InitPageNavigation()

	groupSelectionView.SeletedGroupsTable.SetSelectedFunc(func(row, column int) {
		var logGroups = groupSelectionView.SeletedGroupsTable.GetAllLogGroups()
		if len(logGroups) > 0 {
			insightsResultsView.QueryResultsTable.SetSelectedLogGroups(logGroups)
			serviceRootView.ChangePage(1, nil)
		}
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
		serviceRootView.ChangePage(2, nil)
	})

	return serviceRootView
}
