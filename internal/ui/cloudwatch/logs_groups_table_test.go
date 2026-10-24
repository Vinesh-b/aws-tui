package cloudwatch

import (
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
		{"Name"},
		{"/aws/lambda/orders"},
		{"/aws/lambda/billing"},
	})
}

func TestLogGroupsTable__EmptyDataShowsOnlyHeading(t *testing.T) {
	var table = newPopulatedLogGroupsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}})
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
