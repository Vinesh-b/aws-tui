package cloudwatch

import (
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func eventRow(message string, seconds int) core.TableRow {
	return core.TableRow{
		baseTime.Add(time.Duration(seconds) * time.Second).Format("2006-01-02 15:04:05.000"),
		message,
	}
}

// newFollowedEventsTable has the events of the stream loaded, with the app running
func newFollowedEventsTable(t *testing.T, api *cloudwatchfake.Api) *LogEventsTable {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogEventsTable(serviceCtx)
	table.followInterval = time.Millisecond
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() {
		table.SetSelectedLogGroup("group")
		table.SetSelectedLogStream("stream")
		table.RefreshLogEvents(true)
	})
	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Timestamp", "Message"}, eventRow("START", 0)})
	return table
}

func pressFollow(t *testing.T, table *LogEventsTable) {
	t.Helper()
	uitest.OnUI(t, table.serviceCtx.App, func() {
		uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Follow)
	})
}

func TestLogEventsTable__FollowAddsTheEventsThatArrive(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)

	pressFollow(t, table)
	api.QueueFollowedEvents(newLogEvent("one", 1), newLogEvent("two", 2))

	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{
		{"Timestamp", "Message"}, eventRow("START", 0), eventRow("one", 1), eventRow("two", 2),
	})
}

func TestLogEventsTable__FollowKeepsTheCursorOnTheLastRow(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	uitest.OnUI(t, table.serviceCtx.App, func() { table.Select(1, 0) })

	pressFollow(t, table)
	api.QueueFollowedEvents(newLogEvent("one", 1))

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		var row, _ = table.GetTable().GetSelection()
		return table.GetTable().GetRowCount() == 3 && row == 2
	}, "the cursor to follow the new event")
}

func TestLogEventsTable__FollowLeavesTheCursorWhereItIsAwayFromTheLastRow(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0), newLogEvent("MIDDLE", 1)))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogEventsTable(serviceCtx)
	table.followInterval = time.Millisecond
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		table.SetSelectedLogGroup("group")
		table.SetSelectedLogStream("stream")
		table.RefreshLogEvents(true)
	})
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 }, "the events")
	uitest.OnUI(t, serviceCtx.App, func() { table.Select(1, 0) })

	pressFollow(t, table)
	api.QueueFollowedEvents(newLogEvent("one", 2))

	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 4 }, "the new event")
	uitest.OnUI(t, serviceCtx.App, func() {
		if row, _ := table.GetTable().GetSelection(); row != 1 {
			t.Errorf("row = %d, want the cursor to stay on 1", row)
		}
	})
}

func TestLogEventsTable__FollowKeyAgainStopsFollowing(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	pressFollow(t, table)
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return len(api.CallsWithPrefix("follow ")) > 0 }, "following to start")

	pressFollow(t, table)
	api.QueueFollowedEvents(newLogEvent("late", 1))

	uitest.Consistently(t, table.serviceCtx.App, 100*time.Millisecond, func() bool {
		return table.GetTable().GetRowCount() == 2 && !table.IsFollowing()
	}, "the table to stay as it was")
}

func TestLogEventsTable__FollowIsStoppedByReloadingTheEvents(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	pressFollow(t, table)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.RefreshLogEvents(true) })

	uitest.OnUI(t, table.serviceCtx.App, func() {
		if table.IsFollowing() {
			t.Error("still following after the events were loaded again")
		}
	})
}

func TestLogEventsTable__FollowIsStoppedByChangingTheStream(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	pressFollow(t, table)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetSelectedLogStream("other") })

	uitest.OnUI(t, table.serviceCtx.App, func() {
		if table.IsFollowing() {
			t.Error("still following after the stream changed")
		}
	})
}

func TestLogEventsTable__FollowStopsAndReportsAnError(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback
	api.SetFollowErr(errors.New("throttled"))

	pressFollow(t, table)

	if got := uitest.AwaitError(t, received); got != "throttled" {
		t.Errorf("message = %q, want %q", got, "throttled")
	}
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return !table.IsFollowing() }, "following to stop")
}
