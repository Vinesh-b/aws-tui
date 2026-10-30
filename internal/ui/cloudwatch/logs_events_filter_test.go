package cloudwatch

import (
	"testing"
	"time"

	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newFilteredEventsTable(t *testing.T) (*LogEventsTable, *cloudwatchfake.Api) {
	t.Helper()
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(
		newLogEvent("START", 0), newLogEvent("ERROR one", 1), newLogEvent("INFO", 2), newLogEvent("ERROR two", 3),
	))
	var table = newFollowedEventsTable(t, api)
	return table, api
}

func TestLogEventsTable__FilterPatternShowsTheEventsThatMatch(t *testing.T) {
	var table, _ = newFilteredEventsTable(t)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetFilterPattern("ERROR") })

	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{
		{"Timestamp", "Message"}, eventRow("ERROR one", 1), eventRow("ERROR two", 3),
	})
}

func TestLogEventsTable__EmptyFilterPatternShowsAllTheEventsAgain(t *testing.T) {
	var table, api = newFilteredEventsTable(t)
	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetFilterPattern("ERROR") })
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 }, "the matches")

	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetFilterPattern("") })

	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 5 }, "all the events")
	if got := api.CallsWithPrefix("filter "); len(got) != 1 {
		t.Errorf("filter calls = %v, want only the one with the pattern", got)
	}
}

func TestLogEventsTable__FilterPatternIsSearchedForInTheTimeRange(t *testing.T) {
	var table, api = newFilteredEventsTable(t)
	table.now = func() time.Time { return baseTime }
	pressTimeRange(t, table)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetFilterPattern("ERROR") })

	want := "filter group stream ERROR from " + baseTime.Add(-15*time.Minute).UTC().Format(time.RFC3339)
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		var calls = api.CallsWithPrefix("filter ")
		return len(calls) == 1 && calls[0] == want
	}, "the filter to be searched for from the start of the range")
}

func TestLogEventsTable__FilterKeyAsksForThePatternAndEnterAppliesIt(t *testing.T) {
	var table, _ = newFilteredEventsTable(t)

	uitest.OnUI(t, table.serviceCtx.App, func() {
		table.InputHandler()(
			tcell.NewEventKey(tcell.KeyRune, core.APP_KEY_BINDINGS.Filter, tcell.ModNone), func(tview.Primitive) {},
		)
		if table.IsOverlayHidden(logFilterOverlayId) {
			t.Fatal("filter input not shown by the filter key")
		}
		table.filterView.InputField.SetText("ERROR")
		table.filterView.InputField.InputHandler()(
			tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {},
		)
		if !table.IsOverlayHidden(logFilterOverlayId) {
			t.Error("filter input still shown after enter")
		}
	})

	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{
		{"Timestamp", "Message"}, eventRow("ERROR one", 1), eventRow("ERROR two", 3),
	})
}

func TestLogEventsTable__FilterInputShowsThePatternInUse(t *testing.T) {
	var table, _ = newFilteredEventsTable(t)
	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetFilterPattern("ERROR") })
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 }, "the matches")
	uitest.OnUI(t, table.serviceCtx.App, func() { table.filterView.InputField.SetText("half typed") })

	uitest.OnUI(t, table.serviceCtx.App, func() {
		table.ToggleOverlay(logFilterOverlayId, false)
		if got := table.filterView.InputField.GetText(); got != "ERROR" {
			t.Errorf("filter input = %q, want %q", got, "ERROR")
		}
	})
}

func TestLogEventsTable__FilterIsNotFollowed(t *testing.T) {
	var table, _ = newFilteredEventsTable(t)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback
	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetFilterPattern("ERROR") })

	pressFollow(t, table)

	if got := uitest.AwaitError(t, received); got != "Clear the filter to follow the log stream" {
		t.Errorf("message = %q", got)
	}
	uitest.OnUI(t, table.serviceCtx.App, func() {
		if table.IsFollowing() {
			t.Error("following a filtered log stream")
		}
	})
}
