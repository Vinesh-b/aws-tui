package cloudwatchfake

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// insightsState is what the Insights queries of an Api were asked and give.
// It is guarded by the mutex of the Api.
type insightsState struct {
	// results are given one for each time the results are asked for, and the
	// last of them after that
	results   []awsapi.InsightsQueryResults
	record    map[string]string
	recordErr error

	logGroups  [][]string
	queries    []string
	windows    []QueryWindow
	stopped    []string
	polls      int
	recordPtrs []string
}

// WithInsightsResults sets what GetInsightsQueryResults gives: the first
// results the first time it is asked, and so on, then the last of them every
// time after that. With none it gives a complete query with no rows.
func WithInsightsResults(results ...awsapi.InsightsQueryResults) Option {
	return func(f *Api) { f.insights.results = results }
}

// WithInsightsRecord sets the log record GetInsightsLogRecord gives
func WithInsightsRecord(record map[string]string) Option {
	return func(f *Api) { f.insights.record = maps.Clone(record) }
}

// WithInsightsRecordErr makes GetInsightsLogRecord fail
func WithInsightsRecordErr(err error) Option {
	return func(f *Api) { f.insights.recordErr = err }
}

// QueryWindow is the times a query was started between
type QueryWindow struct{ Start, End time.Time }

// QueryWindows gives the times of each query started, oldest first
func (f *Api) QueryWindows() []QueryWindow {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.insights.windows)
}

// QueryLogGroups gives the log groups of each query started, oldest first
func (f *Api) QueryLogGroups() [][]string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var groups = slices.Clone(f.insights.logGroups)
	for i := range groups {
		groups[i] = slices.Clone(groups[i])
	}
	return groups
}

// Queries gives the text of each query started, oldest first
func (f *Api) Queries() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.insights.queries)
}

// StoppedQueries gives the ids of the queries StopInsightsQuery was asked to
// stop, oldest first
func (f *Api) StoppedQueries() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.insights.stopped)
}

// ResultPolls is how many times the results of a query were asked for
func (f *Api) ResultPolls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.insights.polls
}

// RecordPointers gives the pointers GetInsightsLogRecord was asked for, oldest
// first
func (f *Api) RecordPointers() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.insights.recordPtrs)
}

// StartInsightsQuery starts the query, which is given the id query-1 for the
// first query, query-2 for the next and so on
func (f *Api) StartInsightsQuery(
	ctx context.Context, logGroups []string, startTime time.Time, endTime time.Time, query string,
) (string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.insights.logGroups = append(f.insights.logGroups, slices.Clone(logGroups))
	f.insights.queries = append(f.insights.queries, query)
	f.insights.windows = append(f.insights.windows, QueryWindow{startTime, endTime})
	return fmt.Sprintf("query-%d", len(f.insights.queries)), nil
}

func (f *Api) StopInsightsQuery(ctx context.Context, queryId string) (bool, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.insights.stopped = append(f.insights.stopped, queryId)
	return true, nil
}

func (f *Api) GetInsightsQueryResults(ctx context.Context, queryId string) (awsapi.InsightsQueryResults, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var results = f.insights.results
	if len(results) == 0 {
		return awsapi.InsightsQueryResults{Status: types.QueryStatusComplete}, nil
	}
	var response = results[min(f.insights.polls, len(results)-1)]
	f.insights.polls++
	return response, nil
}

func (f *Api) GetInsightsLogRecord(ctx context.Context, recordPtr string) (map[string]string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.insights.recordPtrs = append(f.insights.recordPtrs, recordPtr)
	return maps.Clone(f.insights.record), f.insights.recordErr
}
