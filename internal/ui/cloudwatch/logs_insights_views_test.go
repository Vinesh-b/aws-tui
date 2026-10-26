package cloudwatch

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type fakeInsightsApi struct {
	fakeCloudWatchLogsApi
	results [][]types.ResultField

	mu      sync.Mutex
	started [][]string
	queries []string
}

func (f *fakeInsightsApi) StartInsightsQuery(
	ctx context.Context, logGroups []string, startTime time.Time, endTime time.Time, query string,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, logGroups)
	f.queries = append(f.queries, query)
	return "query-1", nil
}

func (f *fakeInsightsApi) GetInsightsQueryResults(
	ctx context.Context, queryId string,
) ([][]types.ResultField, types.QueryStatus, error) {
	return f.results, types.QueryStatusComplete, nil
}

func (f *fakeInsightsApi) startedQueries() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.started)
}

func newInsightsPageView(t *testing.T, api awsapi.CloudWatchLogsApi) *LogsInsightsPageView {
	var serviceCtx = uitest.NewServiceContextWith(api)
	var view = NewLogsInsightsPageView(
		NewLogGroupsTable(serviceCtx),
		NewSelectedGroupsTable(serviceCtx),
		NewInsightsQueryResultsTable(serviceCtx),
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
	var api = &fakeInsightsApi{results: [][]types.ResultField{resultRow("@message", "hello", "@ptr", "ptr-1")}}
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
