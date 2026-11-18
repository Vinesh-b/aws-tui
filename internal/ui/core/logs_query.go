package core

import (
	"errors"
	"time"
)

// LogsQuery is a CloudWatch Logs Insights query that one service asks the
// logs to run, such as on the logs of an X-Ray trace
type LogsQuery struct {
	Query      string
	LogGroups  []string
	Start, End time.Time
}

// LogsQueryOpener is a service page that runs the Logs Insights queries that
// other services open
type LogsQueryOpener interface {
	OpenLogsQuery(query LogsQuery)
}

// SetLogsQueryOpener is called by the app with how to run a Logs Insights
// query in the page of the logs
func (inst *AppContext) SetLogsQueryOpener(open func(query LogsQuery) error) {
	inst.openLogsQuery = open
}

// OpenLogsQuery runs the Logs Insights query in the page of the logs and shows
// it there
func (inst *AppContext) OpenLogsQuery(query LogsQuery) error {
	if inst.openLogsQuery == nil {
		return errors.New("can't open Logs Insights queries")
	}
	return inst.openLogsQuery(query)
}
