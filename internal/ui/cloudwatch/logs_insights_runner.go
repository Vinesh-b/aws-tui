package cloudwatch

import (
	"context"
	"fmt"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/rivo/tview"
)

// defaultInsightsPollInterval is the time between asking for the results of a
// query that is still running
const defaultInsightsPollInterval = time.Second

func isInsightsQueryRunning(status types.QueryStatus) bool {
	return status == types.QueryStatusRunning || status == types.QueryStatusScheduled
}

// insightsFailure says why a query that is no longer running did not give its
// results, which is empty if it did or was cancelled.
func insightsFailure(status types.QueryStatus) string {
	switch status {
	case types.QueryStatusFailed, types.QueryStatusTimeout, types.QueryStatusUnknown:
		return fmt.Sprintf("Query failed with status %s", status)
	}
	return ""
}

// pollInsightsQuery gets the results of the query until it is no longer
// running, calling onUpdate, which can be nil, with those it has until then.
// It stops with the error of the context if that is done first.
func pollInsightsQuery(
	ctx context.Context,
	api awsapi.CloudWatchLogsApi,
	queryId string,
	interval time.Duration,
	onUpdate func(awsapi.InsightsQueryResults),
) (awsapi.InsightsQueryResults, error) {
	for {
		var results, err = api.GetInsightsQueryResults(ctx, queryId)
		if err != nil || !isInsightsQueryRunning(results.Status) {
			return results, err
		}
		if onUpdate != nil {
			onUpdate(results)
		}

		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// InsightsQueryRunner runs a query on log groups and gives its results once
// it is no longer running.
type InsightsQueryRunner struct {
	ctx                  context.Context
	app                  *tview.Application
	api                  awsapi.CloudWatchLogsApi
	pollInterval         time.Duration
	ErrorMessageCallback func(text string, a ...any)
}

func NewInsightsQueryRunner(
	ctx context.Context,
	app *tview.Application,
	api awsapi.CloudWatchLogsApi,
) *InsightsQueryRunner {

	return &InsightsQueryRunner{
		ctx:                  ctx,
		app:                  app,
		api:                  api,
		pollInterval:         defaultInsightsPollInterval,
		ErrorMessageCallback: func(text string, a ...any) {},
	}
}

// ExecuteInsightsQuery runs the query in the background and sends its results
// once, which are empty if it failed.
func (inst *InsightsQueryRunner) ExecuteInsightsQuery(
	query InsightsQuery, logGroups []string, resultChan chan [][]types.ResultField,
) {
	if len(logGroups) == 0 {
		inst.ErrorMessageCallback("No log groups selected")
		return
	}

	go func() {
		var results [][]types.ResultField
		defer func() { resultChan <- results }()

		var queryId, err = inst.api.StartInsightsQuery(
			inst.ctx, logGroups, query.StartTime, query.EndTime, query.Query,
		)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}

		var final awsapi.InsightsQueryResults
		if final, err = pollInsightsQuery(inst.ctx, inst.api, queryId, inst.pollInterval, nil); err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}
		if failure := insightsFailure(final.Status); len(failure) > 0 {
			inst.ErrorMessageCallback("%s", failure)
			return
		}
		results = final.Results
	}()
}
