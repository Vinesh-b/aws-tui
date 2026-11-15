package cloudwatch

import (
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gdamore/tcell/v2"
)

func newAlarm(name string, state types.StateValue) types.MetricAlarm {
	return types.MetricAlarm{AlarmName: aws.String(name), StateValue: state}
}

func newPopulatedAlarmListTable(alarms ...types.MetricAlarm) *AlarmListTable {
	var table = NewAlarmListTable(uitest.NewServiceContext[awsapi.CloudWatchAlarmsApi]())
	table.populateAlarmsTable(alarms)
	return table
}

func TestAlarmListTable__ShowsHeadingsAndOneRowPerAlarm(t *testing.T) {
	var table = newPopulatedAlarmListTable(
		newAlarm("cpu-high", types.StateValueAlarm),
		newAlarm("disk-ok", types.StateValueOk),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Name", "State"},
		{"cpu-high", "ALARM"},
		{"disk-ok", "OK"},
	})
}

func TestAlarmListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedAlarmListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name", "State"}})
}

func TestAlarmListTable__SelectingRowTracksSelectedAlarm(t *testing.T) {
	var table = newPopulatedAlarmListTable(
		newAlarm("cpu-high", types.StateValueAlarm),
		newAlarm("disk-ok", types.StateValueOk),
	)

	table.Select(2, 0)

	if got := table.GetSelectedAlarmName(); got != "disk-ok" {
		t.Errorf("selected alarm = %q, want %q", got, "disk-ok")
	}
}

func TestAlarmListTable__SelectedFuncReceivesRowAndColumn(t *testing.T) {
	var table = newPopulatedAlarmListTable(newAlarm("cpu-high", types.StateValueAlarm))
	var gotRow, gotCol = -1, -1
	table.SetSelectedFunc(func(row, column int) { gotRow, gotCol = row, column })

	table.Select(1, 0)
	uitest.PressEnter(table)

	if gotRow != 1 || gotCol != 0 {
		t.Errorf("handler got (%d,%d), want (1,0)", gotRow, gotCol)
	}
}

// drawnForeground draws the table and returns the colour the text is drawn in,
// which is what the user sees whichever field of the cell holds the colour.
// The selected row is drawn in the selection colours, so the text must be on
// another row.
func drawnForeground(t *testing.T, table *AlarmListTable, text string) tcell.Color {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(40, 10)
	table.SetRect(0, 0, 40, 10)
	table.Draw(screen)
	screen.Show()

	var cells, width, height = screen.GetContents()
	for row := 0; row < height; row++ {
		var line strings.Builder
		for col := 0; col < width; col++ {
			line.WriteString(string(cells[row*width+col].Runes))
		}
		if idx := strings.Index(line.String(), text); idx >= 0 {
			var fg, _, _ = cells[row*width+idx].Style.Decompose()
			return fg
		}
	}
	t.Fatalf("%q is not on the screen", text)
	return 0
}

func TestAlarmListTable__ColoursOnlyAlarmsInTheAlarmState(t *testing.T) {
	var table = newPopulatedAlarmListTable(
		newAlarm("cpu-high", types.StateValueAlarm),
		newAlarm("disk-ok", types.StateValueOk),
		newAlarm("no-data", types.StateValueInsufficientData),
	)

	// Select a row that is not in the alarm state, then the alarm row
	table.Select(3, 0)
	for _, text := range []string{"cpu-high", "ALARM"} {
		if got := drawnForeground(t, table, text); got != tcell.ColorRed {
			t.Errorf("%q is drawn in %v, want red", text, got)
		}
	}
	table.Select(1, 0)
	for _, text := range []string{"disk-ok", "OK", "no-data", "INSUFFICIENT_DATA"} {
		if got := drawnForeground(t, table, text); got == tcell.ColorRed {
			t.Errorf("%q is drawn red but is not in the alarm state", text)
		}
	}
}
