package cloudwatch

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InsightsTabName = string

const (
	InsightsTabLogGroups InsightsTabName = "Log Groups"
	InsightsTabQuery     InsightsTabName = "Query"
	InsightsTabRecord    InsightsTabName = "Record"
)

// LogsInsightsPageView is a page to pick the log groups, write the query and
// look through its results, which are listed with the selected one expanded.
type LogsInsightsPageView struct {
	*core.ServicePageView
	LogGroupsTable      *LogGroupsTable
	SelectedGroupsTable *SelectedGroupsTable
	QueryView           *InsightsQueryInputView
	ResultsTable        *InsightsQueryResultsTable
	RecordView          *core.SearchableTextView
	TabView             *core.TabViewHorizontal
	serviceCtx          *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogsInsightsPageView(
	logGroupsTable *LogGroupsTable,
	selectedGroupsTable *SelectedGroupsTable,
	resultsTable *InsightsQueryResultsTable,
	serviceCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogsInsightsPageView {
	var queryView = NewInsightsQueryInputView(serviceCtx.AppContext)
	var recordView = core.CreateJsonTableDataView(serviceCtx.AppContext, resultsTable, -1)
	recordView.SetTitle("Record")

	const logGroupsSize = 6000
	const selectedGroupsSize = 4000
	var groupsView = core.NewResizableView(
		logGroupsTable, logGroupsSize,
		selectedGroupsTable, selectedGroupsSize,
		tview.FlexColumn,
	)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(InsightsTabLogGroups, groupsView, 0, 1, true).
		AddTab(InsightsTabQuery, queryView, 0, 1, true).
		AddTab(InsightsTabRecord, recordView, 0, 1, true)

	tabView.GetTab(InsightsTabLogGroups).InitViewNavigation(
		[][]core.View{{logGroupsTable, selectedGroupsTable}},
	)
	tabView.GetTab(InsightsTabQuery).InitViewNavigation(queryView.NavigationRows())

	const tabViewSize = 5000
	const resultsTableSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		resultsTable, resultsTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &LogsInsightsPageView{
		ServicePageView:     serviceView,
		LogGroupsTable:      logGroupsTable,
		SelectedGroupsTable: selectedGroupsTable,
		QueryView:           queryView,
		ResultsTable:        resultsTable,
		RecordView:          recordView,
		TabView:             tabView,
		serviceCtx:          serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	logGroupsTable.ErrorMessageCallback = errorHandler
	selectedGroupsTable.ErrorMessageCallback = errorHandler
	resultsTable.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{resultsTable},
		},
	)
	// The log groups are the first thing to pick, while there are no results
	serviceView.SetFocusedRow(0)
	view.initInputCapture()

	return view
}

func (inst *LogsInsightsPageView) initInputCapture() {
	// The selection of the table is only tracked once there is a handler
	inst.LogGroupsTable.SetSelectionChangedFunc(func(row, column int) {})
	inst.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.SelectedGroupsTable.AddLogGroup(inst.LogGroupsTable.GetSelectedLogGroup())
		inst.SelectedGroupsTable.RefreshSelectedGroups()
	})

	inst.QueryView.RunButton.SetSelectedFunc(func() { inst.RunQuery() })
	inst.QueryView.StopButton.SetSelectedFunc(func() { inst.ResultsTable.StopQuery() })
}

// RunQuery runs the query on the selected log groups and moves to the results.
// Without log groups it moves to where they are selected.
func (inst *LogsInsightsPageView) RunQuery() {
	var query, err = inst.QueryView.GenerateQuery()
	if err != nil {
		inst.ResultsTable.ErrorMessageCallback("%s", err.Error())
		return
	}

	var logGroups = inst.SelectedGroupsTable.GetAllLogGroups()
	inst.ResultsTable.ExecuteQuery(query, logGroups)
	if len(logGroups) == 0 {
		inst.TabView.SwitchToTab(InsightsTabLogGroups)
		return
	}
	inst.serviceCtx.App.SetFocus(inst.ResultsTable)
}

func NewLogsInsightsHomeView(appCtx *core.AppContext, api awsapi.CloudWatchLogsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var insightsView = NewLogsInsightsPageView(
		NewLogGroupsTable(serviceCtx),
		NewSelectedGroupsTable(serviceCtx),
		NewInsightsQueryResultsTable(serviceCtx),
		serviceCtx,
	)
	var logEventsView = NewLogEventsPageView(
		NewLogEventsTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDWATCH_LOGS_INSIGHTS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Insights", insightsView, true).
		AddPage("Log Events", logEventsView, true, true)

	serviceRootView.InitPageNavigation()

	var recordPtr = ""
	insightsView.ResultsTable.SetSelectedFunc(func(row, column int) {
		recordPtr = insightsView.ResultsTable.GetRecordPtr(row)
		var record, err = api.GetInsightsLogRecord(appCtx.Ctx, recordPtr)
		if err != nil {
			logEventsView.LogEventsTable.ErrorMessageCallback("%s", err.Error())
		}

		var logStream = record["@logStream"]
		var _, logGroup, _ = strings.Cut(record["@log"], ":")

		logEventsView.LogEventsTable.SetSelectedLogGroup(logGroup)
		logEventsView.LogEventsTable.SetSelectedLogStream(logStream)
		logEventsView.LogEventsTable.RefreshLogEvents(true)
		serviceRootView.ChangePage(1, nil)
	})

	return serviceRootView
}
