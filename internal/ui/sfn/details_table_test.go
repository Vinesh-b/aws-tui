package sfn

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func newDetailsTable() *SfnDetailsTable {
	return NewSfnDetailsTable(uitest.NewServiceContext[awsapi.StateMachineApi]())
}

const testRoleArn = "arn:aws:iam::123456789012:role/service-role/orders-sfn"

func newStateMachineDescription() *sfn.DescribeStateMachineOutput {
	return &sfn.DescribeStateMachineOutput{
		Name:            aws.String("orders"),
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:orders"),
		Type:            types.StateMachineTypeStandard,
		Description:     aws.String("Processes orders"),
		Status:          types.StateMachineStatusActive,
		CreationDate:    at(0),
		RoleArn:         aws.String(testRoleArn),
	}
}

func TestDetailsTable__NoStateMachineShowsNothing(t *testing.T) {
	var table = newDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestDetailsTable__ShowsStateMachineSummary(t *testing.T) {
	var table = newDetailsTable()
	table.data = newStateMachineDescription()

	table.populateTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"ARN", "arn:aws:states:us-east-1:123456789012:stateMachine:orders"},
		{"Type", "STANDARD"},
		{"Description", "Processes orders"},
		{"Status", "ACTIVE"},
		{"Created", "2024-01-02 03:04:05"},
		{"Role", testRoleArn},
	})
	if link, _ := table.GetLink(6); link != testRoleArn {
		t.Errorf("role link = %q, want %q", link, testRoleArn)
	}
}

func TestDetailsTable__ShowsLoggingConfiguration(t *testing.T) {
	var table = newDetailsTable()
	table.data = newStateMachineDescription()
	table.data.LoggingConfiguration = &types.LoggingConfiguration{
		IncludeExecutionData: true,
		Level:                types.LogLevelAll,
		Destinations: []types.LogDestination{{
			CloudWatchLogsLogGroup: &types.CloudWatchLogsLogGroup{
				LogGroupArn: aws.String("arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/orders:*"),
			},
		}},
	}

	table.populateTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"ARN", "arn:aws:states:us-east-1:123456789012:stateMachine:orders"},
		{"Type", "STANDARD"},
		{"Description", "Processes orders"},
		{"Status", "ACTIVE"},
		{"Created", "2024-01-02 03:04:05"},
		{"Role", testRoleArn},
		{"Logging"},
		{"Include Execution Data", "true"},
		{"Log Level", "ALL"},
		{"Log Group", "/aws/states/orders"},
	})
}

func TestDetailsTable__LogGroupLinksToLogGroup(t *testing.T) {
	var table = newDetailsTable()
	table.data = newStateMachineDescription()
	table.data.LoggingConfiguration = &types.LoggingConfiguration{
		Destinations: []types.LogDestination{{
			CloudWatchLogsLogGroup: &types.CloudWatchLogsLogGroup{
				LogGroupArn: aws.String("arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/orders:*"),
			},
		}},
	}

	table.populateTable()

	const logGroupRow = 10
	if got := table.GetCellText(logGroupRow, 0); got != "Log Group" {
		t.Fatalf("row %d = %q, want Log Group", logGroupRow, got)
	}
	if link, _ := table.GetLink(logGroupRow); link != "arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/orders:*" {
		t.Errorf("link = %q", link)
	}
}
