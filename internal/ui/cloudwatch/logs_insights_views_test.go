package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type fakeInsightsApi struct {
	fakeCloudWatchLogsApi
	// responses are given one for each time the results are asked for, and the
	// last of them after that
	responses []awsapi.InsightsQueryResults
	record    map[string]string
	recordErr error

	mu         sync.Mutex
	started    [][]string
	queries    []string
	stopped    []string
	polls      int
	recordPtrs []string
}

func completeWith(rows ...[]types.ResultField) awsapi.InsightsQueryResults {
	return awsapi.InsightsQueryResults{Results: rows, Status: types.QueryStatusComplete}
}

func runningWith(rows ...[]types.ResultField) awsapi.InsightsQueryResults {
	return awsapi.InsightsQueryResults{
		Results: rows,
		Status:  types.QueryStatusRunning,
		Statistics: &types.QueryStatistics{
			RecordsMatched: float64(len(rows)),
			RecordsScanned: 2500,
		},
	}
}

func (f *fakeInsightsApi) StartInsightsQuery(
	ctx context.Context, logGroups []string, startTime time.Time, endTime time.Time, query string,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, logGroups)
	f.queries = append(f.queries, query)
	return fmt.Sprintf("query-%d", len(f.started)), nil
}

func (f *fakeInsightsApi) StopInsightsQuery(ctx context.Context, queryId string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, queryId)
	return true, nil
}

func (f *fakeInsightsApi) GetInsightsQueryResults(
	ctx context.Context, queryId string,
) (awsapi.InsightsQueryResults, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.responses) == 0 {
		return completeWith(), nil
	}
	var response = f.responses[min(f.polls, len(f.responses)-1)]
	f.polls++
	return response, nil
}

func (f *fakeInsightsApi) GetInsightsLogRecord(ctx context.Context, recordPtr string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recordPtrs = append(f.recordPtrs, recordPtr)
	return f.record, f.recordErr
}

func (f *fakeInsightsApi) startedQueries() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.started)
}

func (f *fakeInsightsApi) stoppedQueries() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.stopped)
}

func (f *fakeInsightsApi) pollCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.polls
}

func newInsightsPageView(t *testing.T, api awsapi.CloudWatchLogsApi) *LogsInsightsPageView {
	var serviceCtx = uitest.NewServiceContextWith(api)
	var resultsTable = NewInsightsQueryResultsTable(serviceCtx)
	resultsTable.pollInterval = 5 * time.Millisecond
	var view = NewLogsInsightsPageView(
		NewLogGroupsTable(serviceCtx),
		NewSelectedGroupsTable(serviceCtx),
		resultsTable,
		serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestInsightsPageView__SelectingALogGroupAddsItToTheSelectedGroups(t *testing.T) {
	var api = &fakeInsightsApi{fakeCloudWatchLogsApi: fakeCloudWatchLogsApi{
		groups: []types.LogGroup{newLogGroup("/aws/lambda/orders")},
	}}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App

	view.LogGroupsTable.RefreshLogGroups(true)
	uitest.Eventually(t, app, func() bool { return view.LogGroupsTable.GetSelectedLogGroup() != "" })
	uitest.OnUI(t, app, func() { uitest.PressEnter(view.LogGroupsTable) })

	uitest.Eventually(t, app, func() bool {
		return slices.Equal(view.SelectedGroupsTable.GetAllLogGroups(), []string{"/aws/lambda/orders"}) &&
			view.SelectedGroupsTable.GetCellText(1, 0) == "/aws/lambda/orders"
	})
}

func TestInsightsPageView__RunningWithoutLogGroupsMovesToThemAndReportsIt(t *testing.T) {
	var api = &fakeInsightsApi{}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	view.ResultsTable.ErrorMessageCallback = callback
	uitest.OnUI(t, app, func() { view.TabView.SwitchToTab(InsightsTabQuery) })

	uitest.OnUI(t, app, func() { view.RunQuery() })

	if got := uitest.AwaitError(t, received); got != "No log groups selected" {
		t.Errorf("message = %q", got)
	}
	uitest.OnUI(t, app, func() {
		if name, _ := view.TabView.GetCurrentTab(); name != InsightsTabLogGroups {
			t.Errorf("tab = %q, want %q", name, InsightsTabLogGroups)
		}
	})
	if started := api.startedQueries(); len(started) != 0 {
		t.Errorf("queries started = %v, want none", started)
	}
}

func TestInsightsPageView__RunningQueriesTheSelectedGroupsAndShowsTheResults(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{completeWith(resultRow("@message", "hello", "@ptr", "ptr-1"))}}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	uitest.OnUI(t, app, func() {
		view.SelectedGroupsTable.AddLogGroup("/aws/lambda/orders")
		view.QueryView.queryTextArea.SetText("fields @message", false)
		view.RunQuery()
	})

	uitest.Eventually(t, app, func() bool { return view.ResultsTable.GetCellText(1, 0) == "hello" })

	if started := api.startedQueries(); len(started) != 1 || !slices.Equal(started[0], []string{"/aws/lambda/orders"}) {
		t.Errorf("queries started = %v, want one on /aws/lambda/orders", started)
	}
	if api.queries[0] != "fields @message" {
		t.Errorf("query = %q", api.queries[0])
	}
	uitest.OnUI(t, app, func() {
		if !view.ResultsTable.HasFocus() {
			t.Error("the results table does not have focus")
		}
	})
}

// runQuery runs a query on a group on the UI thread
func runQuery(t *testing.T, view *LogsInsightsPageView) {
	t.Helper()
	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.SelectedGroupsTable.AddLogGroup("/aws/lambda/orders")
		view.RunQuery()
	})
}

func TestInsightsPageView__ResultsAndStatusComeInWhileTheQueryRuns(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{
		runningWith(resultRow("@message", "one")),
		completeWith(resultRow("@message", "one"), resultRow("@message", "two")),
	}}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App

	runQuery(t, view)

	uitest.Eventually(t, app, func() bool { return view.ResultsTable.GetCellText(2, 0) == "two" })
	uitest.Eventually(t, app, func() bool { return strings.Contains(view.ResultsTable.GetTitle(), "Complete") })
}

func TestInsightsPageView__RunningStatusShowsMatchedAndScannedRecords(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{runningWith(resultRow("@message", "one"))}}
	var view = newInsightsPageView(t, api)

	runQuery(t, view)

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		var title = view.ResultsTable.GetTitle()
		return strings.Contains(title, "Running") && strings.Contains(title, "1 matched") &&
			strings.Contains(title, "2.5k scanned")
	})
}

func TestInsightsPageView__QueryIsPolledUntilItIsDoneHoweverLongItTakes(t *testing.T) {
	var responses []awsapi.InsightsQueryResults
	for range 30 {
		responses = append(responses, runningWith())
	}
	responses = append(responses, completeWith(resultRow("@message", "late")))
	var api = &fakeInsightsApi{responses: responses}
	var view = newInsightsPageView(t, api)

	runQuery(t, view)

	uitest.Eventually(t, view.serviceCtx.App, func() bool { return view.ResultsTable.GetCellText(1, 0) == "late" })
	if got := api.pollCount(); got != 31 {
		t.Errorf("polls = %d, want 31", got)
	}
}

func TestInsightsPageView__FailedQueryIsReportedWithItsStatus(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{
		{Status: types.QueryStatusFailed},
	}}
	var view = newInsightsPageView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.ResultsTable.ErrorMessageCallback = callback

	runQuery(t, view)

	if got := uitest.AwaitError(t, received); got != "Query failed with status Failed" {
		t.Errorf("message = %q", got)
	}
}

func TestInsightsPageView__StoppingStopsTheQueryAndKeepsTheResultsSoFar(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{runningWith(resultRow("@message", "one"))}}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	runQuery(t, view)
	uitest.Eventually(t, app, func() bool { return view.ResultsTable.GetCellText(1, 0) == "one" })

	uitest.OnUI(t, app, func() { view.ResultsTable.StopQuery() })

	uitest.Eventually(t, app, func() bool { return strings.Contains(view.ResultsTable.GetTitle(), "Cancelled") })
	if got := api.stoppedQueries(); !slices.Equal(got, []string{"query-1"}) {
		t.Errorf("queries stopped = %v, want [query-1]", got)
	}
	var polls = api.pollCount()
	time.Sleep(50 * time.Millisecond)
	if got := api.pollCount(); got > polls+1 {
		t.Errorf("the query was still polled after it was stopped: %d then %d", polls, got)
	}
	uitest.OnUI(t, app, func() {
		if got := view.ResultsTable.GetCellText(1, 0); got != "one" {
			t.Errorf("results after stopping = %q, want the ones so far", got)
		}
	})
}

func TestInsightsPageView__StartingAnotherQueryStopsTheRunningOne(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{runningWith()}}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	runQuery(t, view)
	uitest.Eventually(t, app, func() bool { return api.pollCount() > 0 })

	uitest.OnUI(t, app, func() { view.RunQuery() })

	uitest.Eventually(t, app, func() bool { return len(api.startedQueries()) == 2 })
	uitest.Eventually(t, app, func() bool { return slices.Equal(api.stoppedQueries(), []string{"query-1"}) })
}

func TestInsightsPageView__EmptyOrBackwardsQueriesAreNotRun(t *testing.T) {
	var api = &fakeInsightsApi{}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	view.ResultsTable.ErrorMessageCallback = callback
	uitest.OnUI(t, app, func() {
		view.SelectedGroupsTable.AddLogGroup("/aws/lambda/orders")
		view.QueryView.queryTextArea.SetText("  ", false)
		view.RunQuery()
	})

	if got := uitest.AwaitError(t, received); got != "The query is empty" {
		t.Errorf("message = %q", got)
	}
	if started := api.startedQueries(); len(started) != 0 {
		t.Errorf("queries started = %v, want none", started)
	}
}

func TestInsightsPageView__OpeningARecordGivesItsLogGroupAndStream(t *testing.T) {
	var api = &fakeInsightsApi{
		responses: []awsapi.InsightsQueryResults{completeWith(resultRow("@message", "hello", "@ptr", "ptr-1"))},
		record:    map[string]string{"@log": "123456789012:/aws/lambda/orders", "@logStream": "2024/01/02/[$LATEST]abc"},
	}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	runQuery(t, view)
	uitest.Eventually(t, app, func() bool { return view.ResultsTable.GetCellText(1, 0) == "hello" })
	var opened = make(chan [2]string, 1)

	uitest.OnUI(t, app, func() {
		view.OpenRecord(1, func(logGroup string, logStream string) { opened <- [2]string{logGroup, logStream} })
	})

	select {
	case got := <-opened:
		if want := [2]string{"/aws/lambda/orders", "2024/01/02/[$LATEST]abc"}; got != want {
			t.Errorf("opened = %v, want %v", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the record was not opened")
	}
	if !slices.Equal(api.recordPtrs, []string{"ptr-1"}) {
		t.Errorf("record pointers asked for = %v", api.recordPtrs)
	}
}

func TestInsightsPageView__RecordThatFailsToLoadIsReportedAndNotOpened(t *testing.T) {
	var api = &fakeInsightsApi{
		responses: []awsapi.InsightsQueryResults{completeWith(resultRow("@message", "hello", "@ptr", "ptr-1"))},
		recordErr: errors.New("access denied"),
	}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	view.ResultsTable.ErrorMessageCallback = callback
	runQuery(t, view)
	uitest.Eventually(t, app, func() bool { return view.ResultsTable.GetCellText(1, 0) == "hello" })

	uitest.OnUI(t, app, func() {
		view.OpenRecord(1, func(string, string) { t.Error("a record that failed to load was opened") })
	})

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q", got)
	}
}

func TestInsightsPageView__ResultWithoutARecordIsReportedAndNotLoaded(t *testing.T) {
	var api = &fakeInsightsApi{responses: []awsapi.InsightsQueryResults{completeWith(resultRow("count", "42"))}}
	var view = newInsightsPageView(t, api)
	var app = view.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	view.ResultsTable.ErrorMessageCallback = callback
	runQuery(t, view)
	uitest.Eventually(t, app, func() bool { return view.ResultsTable.GetCellText(1, 0) == "42" })

	uitest.OnUI(t, app, func() {
		view.OpenRecord(1, func(string, string) { t.Error("a result without a record was opened") })
	})

	if got := uitest.AwaitError(t, received); got != "This result has no log record to open" {
		t.Errorf("message = %q", got)
	}
	if len(api.recordPtrs) != 0 {
		t.Errorf("records loaded = %v, want none", api.recordPtrs)
	}
}
