package sfn

import (
	"context"
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

const failedStatusOption = 3

type fakeStateMachineApi struct {
	awsapi.StateMachineApi
	machines   []types.StateMachineListItem
	executions []types.ExecutionListItem
	execution  *sfn.DescribeExecutionOutput
	history    *sfn.GetExecutionHistoryOutput
	err        error
}

func (f *fakeStateMachineApi) ListStateMachines(ctx context.Context, force bool) ([]types.StateMachineListItem, error) {
	return f.machines, f.err
}

func (f *fakeStateMachineApi) ListExecutions(
	ctx context.Context, stateMachineArn string, start time.Time, end time.Time, reset bool,
) ([]types.ExecutionListItem, error) {
	return f.executions, f.err
}

func (f *fakeStateMachineApi) DescribeExecution(
	ctx context.Context, executionArn string,
) (*sfn.DescribeExecutionOutput, error) {
	return f.execution, f.err
}

func (f *fakeStateMachineApi) GetExecutionHistory(
	ctx context.Context, executionArn string,
) (*sfn.GetExecutionHistoryOutput, error) {
	return f.history, f.err
}

func TestListTable__RefreshLoadsStateMachinesFromApi(t *testing.T) {
	var api = &fakeStateMachineApi{machines: []types.StateMachineListItem{
		newStateMachine("orders", types.StateMachineTypeStandard),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var table = NewSfnListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshStateMachines(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		stateMachineHeadings,
		{"orders", "STANDARD", "2024-01-02 03:04:05"},
	})
}

func TestListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeStateMachineApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var table = NewSfnListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshStateMachines(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestListTable__SearchShowsMatchingStateMachines(t *testing.T) {
	var api = &fakeStateMachineApi{machines: []types.StateMachineListItem{
		newStateMachine("orders", types.StateMachineTypeStandard),
		newStateMachine("billing", types.StateMachineTypeExpress),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var table = NewSfnListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshStateMachines(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("bill") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		stateMachineHeadings,
		{"billing", "EXPRESS", "2024-01-02 03:04:05"},
	})
}

func TestSummaryTable__RefreshLoadsExecutionFromApi(t *testing.T) {
	var api = &fakeStateMachineApi{execution: newExecutionDescription()}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var table = NewSfnExecutionSummaryTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshExecutionDetails("arn:aws:states:us-east-1:123456789012:execution:orders:run-1", true)

	uitest.AwaitDetailsRows(t, serviceCtx.App, table.DetailsTable, []core.TableRow{
		{"Name", "run-1"},
		{"Execution ARN", "arn:aws:states:us-east-1:123456789012:execution:orders:run-1"},
		{"State Machine ARN", "arn:aws:states:us-east-1:123456789012:stateMachine:orders"},
		{"Status", "SUCCEEDED"},
		{"Started", "2024-01-02 03:04:05"},
		{"Stopped", "2024-01-02 03:05:35"},
	})
}

func TestExecutionsTable__RefreshShowsStandardExecutionsFilteredByStatus(t *testing.T) {
	var api = &fakeStateMachineApi{executions: []types.ExecutionListItem{
		*newExecution("run-1", types.ExecutionStatusSucceeded, 0, 90).ExecutionListItem,
		*newExecution("run-2", types.ExecutionStatusFailed, 10, 15).ExecutionListItem,
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var table = NewSfnExecutionsTable(serviceCtx.AppContext, api, nil)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedFunction(newStateMachine("orders", types.StateMachineTypeStandard))
	table.queryView.Input.statusDropDown.SetCurrentOption(failedStatusOption)

	table.RefreshExecutions(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		executionHeadings,
		{"run-2", "FAILED", "2024-01-02 03:04:15", "2024-01-02 03:04:20", "5s"},
	})
}

func TestStatesTable__RefreshBuildsStatesFromExecutionHistory(t *testing.T) {
	var api = &fakeStateMachineApi{history: &sfn.GetExecutionHistoryOutput{Events: []types.HistoryEvent{
		stateEntered(2, types.HistoryEventTypeTaskStateEntered, 10, "Charge", "{}"),
		stateExited(3, types.HistoryEventTypeTaskStateExited, 100, "Charge", "{}"),
	}}}
	var appCtx = uitest.NewAppContext()
	var table = NewSfnExecutionDetailsTable(appCtx, api, nil)
	uitest.RunApp(t, appCtx.App)

	table.RefreshExecutionStates("arn:aws:states:us-east-1:123456789012:execution:orders:run-1", true)

	uitest.AwaitRows(t, appCtx.App, table, []core.TableRow{
		{"Name", "Type", "Duration"},
		{"Charge", "Task", "1m30s"},
	})
}
