package cloudwatch

import (
	"slices"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func pressTimeRange(t *testing.T, table *LogEventsTable) {
	t.Helper()
	uitest.OnUI(t, table.serviceCtx.App, func() {
		uitest.PressTableRune(table, core.APP_KEY_BINDINGS.TimeRange)
	})
}

func listCallFrom(since time.Duration) string {
	return "list group stream from " + baseTime.Add(-since).UTC().Format(time.RFC3339)
}

func awaitListCalls(t *testing.T, table *LogEventsTable, api *cloudwatchfake.Api, want ...string) {
	t.Helper()
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return slices.Equal(api.CallsWithPrefix("list "), want)
	}, "the events to be listed as expected")
}

func TestLogEventsTable__TimeRangeKeyLoadsTheEventsOfEachRangeInTurn(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	table.now = func() time.Time { return baseTime }
	var want = []string{"list group stream"}

	for _, span := range []time.Duration{
		15 * time.Minute, time.Hour, 3 * time.Hour, 24 * time.Hour, 7 * 24 * time.Hour,
	} {
		pressTimeRange(t, table)
		want = append(want, listCallFrom(span))
		awaitListCalls(t, table, api, want...)
	}

	pressTimeRange(t, table)
	want = append(want, "list group stream")
	awaitListCalls(t, table, api, want...)
}

func TestLogEventsTable__TimeRangeKeyWorksWhenTheRangeHasNoEvents(t *testing.T) {
	var api = cloudwatchfake.New()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogEventsTable(serviceCtx)
	table.now = func() time.Time { return baseTime }
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		table.SetSelectedLogGroup("group")
		table.SetSelectedLogStream("stream")
	})

	pressTimeRange(t, table)
	awaitListCalls(t, table, api, listCallFrom(15*time.Minute))
	pressTimeRange(t, table)

	awaitListCalls(t, table, api, listCallFrom(15*time.Minute), listCallFrom(time.Hour))
}

func TestLogEventsTable__TimeRangeStaysWhenAnotherStreamIsOpened(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	table.now = func() time.Time { return baseTime }
	pressTimeRange(t, table)
	awaitListCalls(t, table, api, "list group stream", listCallFrom(15*time.Minute))

	uitest.OnUI(t, table.serviceCtx.App, func() {
		table.SetSelectedLogStream("other")
		table.RefreshLogEvents(true)
	})

	awaitListCalls(t, table, api,
		"list group stream", listCallFrom(15*time.Minute),
		"list group other from "+baseTime.Add(-15*time.Minute).UTC().Format(time.RFC3339),
	)
}

func TestLogEventsTable__TimeRangeKeyStopsFollowing(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	pressFollow(t, table)

	pressTimeRange(t, table)

	uitest.OnUI(t, table.serviceCtx.App, func() {
		if table.IsFollowing() {
			t.Error("still following after the time range changed")
		}
	})
}
