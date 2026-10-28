package cloudwatch

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

type InsightsTabName = string

const (
	InsightsTabQuery  InsightsTabName = "Query"
	InsightsTabGroups InsightsTabName = "Groups"
	InsightsTabRecord InsightsTabName = "Record"
)

// LogsInsightsPageView is a page to write the query on the marked log groups
// and look through its results, which are listed with the selected one
// expanded.
type LogsInsightsPageView struct {
	*core.ServicePageView
	SelectedGroupsTable *SelectedGroupsTable
	QueryView           *InsightsQueryInputView
	ResultsTable        *InsightsQueryResultsTable
	RecordView          *core.SearchableTextView
	TabView             *core.TabViewHorizontal
	// SelectLogGroups is called when a query is run without log groups, to
	// have them marked
	SelectLogGroups func()
	serviceCtx      *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogsInsightsPageView(
	selectedGroupsTable *SelectedGroupsTable,
	resultsTable *InsightsQueryResultsTable,
	serviceCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogsInsightsPageView {
	var queryView = NewInsightsQueryInputView(serviceCtx.AppContext)
	var recordView = core.CreateJsonTableDataView(serviceCtx.AppContext, resultsTable, -1)
	recordView.SetTitle("Record")
	recordView.SetJsonHighlighting(true)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(InsightsTabQuery, queryView, 0, 1, true).
		AddTab(InsightsTabGroups, selectedGroupsTable, 0, 1, true).
		AddTab(InsightsTabRecord, recordView, 0, 1, true)

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
		SelectedGroupsTable: selectedGroupsTable,
		QueryView:           queryView,
		ResultsTable:        resultsTable,
		RecordView:          recordView,
		TabView:             tabView,
		serviceCtx:          serviceCtx,
	}
	view.SelectLogGroups = func() {
		resultsTable.ErrorMessageCallback("No log groups marked")
		tabView.SwitchToTab(InsightsTabGroups)
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	selectedGroupsTable.ErrorMessageCallback = errorHandler
	resultsTable.ErrorMessageCallback = errorHandler
	queryView.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{resultsTable},
		},
	)
	// The query is the first thing to enter, while there are no results
	serviceView.SetFocusedRow(0)
	view.initInputCapture()

	return view
}

func (inst *LogsInsightsPageView) initInputCapture() {
	inst.QueryView.RunButton.SetSelectedFunc(func() { inst.RunQuery() })
	inst.QueryView.StopButton.SetSelectedFunc(func() { inst.ResultsTable.StopQuery() })
}

// RunQuery runs the query on the marked log groups and moves to the results.
// Without log groups it has them marked.
func (inst *LogsInsightsPageView) RunQuery() {
	var query, err = inst.QueryView.GenerateQuery()
	if err != nil {
		inst.ResultsTable.ErrorMessageCallback("%s", err.Error())
		return
	}

	inst.QueryView.RememberQuery(query)

	var logGroups = inst.SelectedGroupsTable.marks.All()
	if len(logGroups) == 0 {
		inst.SelectLogGroups()
		return
	}
	inst.ResultsTable.ExecuteQuery(query, logGroups)
	inst.serviceCtx.App.SetFocus(inst.ResultsTable)
}

// OpenRecord loads the log record of the result in the row in the background,
// and then calls open with the log group and stream it is from. Results that
// are not records, such as those of a stats query, and failing to load the
// record are reported instead.
func (inst *LogsInsightsPageView) OpenRecord(row int, open func(logGroup string, logStream string)) {
	var recordPtr = inst.ResultsTable.GetRecordPtr(row)
	if len(recordPtr) == 0 {
		inst.ResultsTable.ErrorMessageCallback("This result has no log record to open")
		return
	}

	go func() {
		var record, err = inst.serviceCtx.Api.GetInsightsLogRecord(inst.serviceCtx.Ctx, recordPtr)
		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ResultsTable.ErrorMessageCallback("%s", err.Error())
				return
			}

			var _, logGroup, _ = strings.Cut(record["@log"], ":")
			open(logGroup, record["@logStream"])
		})
	}()
}
