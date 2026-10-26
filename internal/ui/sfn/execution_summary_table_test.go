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

func newSummaryTable() *SfnExecutionSummaryTable {
	return NewSfnExecutionSummaryTable(uitest.NewServiceContext[awsapi.StateMachineApi]())
}

func newExecutionDescription() *sfn.DescribeExecutionOutput {
	return &sfn.DescribeExecutionOutput{
		Name:            aws.String("run-1"),
		ExecutionArn:    aws.String("arn:aws:states:us-east-1:123456789012:execution:orders:run-1"),
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:orders"),
		Status:          types.ExecutionStatusSucceeded,
		StartDate:       at(0),
		StopDate:        at(90),
	}
}

func TestSummaryTable__NoExecutionShowsNothing(t *testing.T) {
	var table = newSummaryTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestSummaryTable__ShowsExecutionSummary(t *testing.T) {
	var table = newSummaryTable()
	table.data = newExecutionDescription()

	table.populateTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "run-1"},
		{"Execution ARN", "arn:aws:states:us-east-1:123456789012:execution:orders:run-1"},
		{"State Machine ARN", "arn:aws:states:us-east-1:123456789012:stateMachine:orders"},
		{"Status", "SUCCEEDED"},
		{"Started", "2024-01-02 03:04:05"},
		{"Stopped", "2024-01-02 03:05:35"},
	})
}

func TestSummaryTable__RunningExecutionHasNoStopDate(t *testing.T) {
	var table = newSummaryTable()
	table.data = newExecutionDescription()
	table.data.Status = types.ExecutionStatusRunning
	table.data.StopDate = nil

	table.populateTable()

	if got := table.GetCellText(5, 1); got != "" {
		t.Errorf("stop date = %q, want empty", got)
	}
}
