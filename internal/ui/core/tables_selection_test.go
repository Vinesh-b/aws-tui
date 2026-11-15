package core

import (
	"slices"
	"testing"
	"time"

	"aws-tui/internal/safetest"
)

// newSettleTest is a table of four rows in a running app, whose settled
// selection waits on timers that the test fires
func newSettleTest(t *testing.T) (*SelectableTable[string], *fakeTimers, func(func())) {
	t.Helper()
	var appCtx, _, _ = newLoaderTest(t)
	var timers = &fakeTimers{}
	var table = NewSelectableTable[string]("Queues", TableRow{"Name"}, appCtx)
	table.settleDebouncer = newDebouncer(appCtx.App, time.Second, timers.afterFunc)

	var names = []string{"orders", "payments", "invoices", "refunds"}
	var rows []TableRow
	for _, name := range names {
		rows = append(rows, TableRow{name})
	}
	if err := table.SetData(rows, names, 0); err != nil {
		t.Fatal(err)
	}
	return table, timers, onUIOf(t, appCtx.App)
}

func TestSelectableTable__SettledRunsStraightAwayForOneMove(t *testing.T) {
	var table, _, onUI = newSettleTest(t)
	var changed, settled safetest.Recorder[string]
	table.SetSelectionChangedFunc(func(row, column int) { changed.Record(table.GetPrivateData(row, 0)) })
	table.SetSelectionSettledFunc(func(row, column int) { settled.Record(table.GetPrivateData(row, 0)) })

	onUI(func() { table.Select(2, 0) })

	if got := settled.Calls(); !slices.Equal(got, []string{"payments"}) {
		t.Errorf("settled = %v, want [payments]", got)
	}
	if got := changed.Calls(); !slices.Equal(got, []string{"payments"}) {
		t.Errorf("changed = %v, want [payments]", got)
	}
}

func TestSelectableTable__SettledSkipsRowsMovedThroughQuickly(t *testing.T) {
	var table, timers, onUI = newSettleTest(t)
	var changed, settled safetest.Recorder[string]
	table.SetSelectionChangedFunc(func(row, column int) { changed.Record(table.GetPrivateData(row, 0)) })
	table.SetSelectionSettledFunc(func(row, column int) { settled.Record(table.GetPrivateData(row, 0)) })

	onUI(func() {
		for row := 2; row <= 4; row++ {
			table.Select(row, 0)
		}
	})
	timers.fireLatest(t)
	onUI(func() {})

	if got := settled.Calls(); !slices.Equal(got, []string{"payments", "refunds"}) {
		t.Errorf("settled = %v, want [payments refunds]", got)
	}
	// What follows the selection straight away sees every row
	if got := changed.Calls(); !slices.Equal(got, []string{"payments", "invoices", "refunds"}) {
		t.Errorf("changed = %v, want [payments invoices refunds]", got)
	}
}

func TestSelectableTable__SettledIsNotRunForTheHeadings(t *testing.T) {
	var table, _, onUI = newSettleTest(t)
	var settled safetest.Recorder[int]
	table.SetSelectionSettledFunc(func(row, column int) { settled.Record(row) })

	onUI(func() { table.Select(0, 0) })

	if got := settled.Count(); got != 0 {
		t.Errorf("settled ran %d times for the headings, want 0", got)
	}
}
