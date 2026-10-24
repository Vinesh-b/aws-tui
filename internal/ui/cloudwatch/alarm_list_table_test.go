package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
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
