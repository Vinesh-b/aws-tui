package cloudwatch

import (
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func newLogGroup(name string) types.LogGroup {
	return types.LogGroup{LogGroupName: aws.String(name)}
}

func newPopulatedLogGroupsTable(names ...string) *LogGroupsTable {
	var table = NewLogGroupsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	var groups []types.LogGroup
	for _, name := range names {
		groups = append(groups, newLogGroup(name))
	}
	table.data = groups
	table.populateLogGroupsTable(groups)
	return table
}

func TestLogGroupsTable__ShowsHeadingAndOneRowPerGroup(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders", "/aws/lambda/billing")

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"", "Name"},
		{"", "/aws/lambda/orders"},
		{"", "/aws/lambda/billing"},
	})
}

func TestLogGroupsTable__EmptyDataShowsOnlyHeading(t *testing.T) {
	var table = newPopulatedLogGroupsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"", "Name"}})
}

func TestLogGroupsTable__MarkKeyMarksTheGroupAndKeepsTheSelection(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders", "/aws/lambda/billing")
	table.Select(2, 0)

	uitest.PressTableRune(table, logGroupMarkRune)

	if got := table.Marks().All(); !slices.Equal(got, []string{"/aws/lambda/billing"}) {
		t.Errorf("marks = %v, want [/aws/lambda/billing]", got)
	}
	uitest.AssertTableRows(t, table, []core.TableRow{
		{"", "Name"},
		{"", "/aws/lambda/orders"},
		{logGroupMarker, "/aws/lambda/billing"},
	})
	if row, _ := table.GetTable().GetSelection(); row != 2 {
		t.Errorf("selected row = %d, want 2", row)
	}
	if title := table.GetTitle(); !strings.Contains(title, "1 marked") {
		t.Errorf("title = %q, want the marked count", title)
	}
}

func TestLogGroupsTable__MarkKeyOnAMarkedGroupUnmarksIt(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders")
	table.Select(1, 0)

	uitest.PressTableRune(table, logGroupMarkRune)
	uitest.PressTableRune(table, logGroupMarkRune)

	if got := table.Marks().Len(); got != 0 {
		t.Errorf("marks = %v, want none", table.Marks().All())
	}
	uitest.AssertTableRows(t, table, []core.TableRow{{"", "Name"}, {"", "/aws/lambda/orders"}})
	if title := table.GetTitle(); strings.Contains(title, "marked") {
		t.Errorf("title = %q, want no marked count", title)
	}
}

func TestLogGroupsTable__MarksAreShownAgainWhenTheGroupsAreShown(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders", "/aws/lambda/billing")
	table.Marks().Add("/aws/lambda/orders")

	table.populateLogGroupsTable(table.data)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"", "Name"},
		{logGroupMarker, "/aws/lambda/orders"},
		{"", "/aws/lambda/billing"},
	})
}

func TestLogGroupsTable__UnmarkingElsewhereClearsTheMarker(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders")
	table.Marks().Add("/aws/lambda/orders")

	table.Marks().Clear()

	uitest.AssertTableRows(t, table, []core.TableRow{{"", "Name"}, {"", "/aws/lambda/orders"}})
}

func TestLogGroupsTable__QueryKeyCallsTheQueryFunc(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders")
	var queried = false
	table.SetQueryFunc(func() { queried = true })

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.TableQuery)

	if !queried {
		t.Error("the query key did not call the query func")
	}
}

func TestLogGroupsTable__SelectedFuncTracksSelectedGroup(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders", "/aws/lambda/billing")
	table.SetSelectedFunc(func(row, column int) {})

	table.Select(2, 0)
	uitest.PressEnter(table)

	if got := table.GetSelectedLogGroup(); got != "/aws/lambda/billing" {
		t.Errorf("selected group = %q, want %q", got, "/aws/lambda/billing")
	}
}

func TestLogGroupsTable__LogGroupDetailForSelectedGroup(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders", "/aws/lambda/billing")
	table.data[1].RetentionInDays = aws.Int32(14)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.Select(2, 0)

	var detail = table.GetLogGroupDetail()

	if aws.ToString(detail.LogGroupName) != "/aws/lambda/billing" || aws.ToInt32(detail.RetentionInDays) != 14 {
		t.Errorf("detail = %+v, want the billing group", detail)
	}
}

func TestLogGroupsTable__LogGroupDetailIsEmptyWhenNothingSelected(t *testing.T) {
	var table = newPopulatedLogGroupsTable("/aws/lambda/orders")

	if detail := table.GetLogGroupDetail(); detail.LogGroupName != nil {
		t.Errorf("detail = %+v, want empty", detail)
	}
}
