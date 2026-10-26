package eventbridge

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type fakeEventBridgeApi struct {
	awsapi.EventBridgeApi
	buses      []types.EventBus
	tags       []types.Tag
	rules      []types.Rule
	targets    []types.Target
	err        error
	targetsErr error
	// Per rule targets; falls back to targets when the rule is not present
	targetsByRule map[string][]types.Target
	targetCalls   atomic.Int32
}

func (f *fakeEventBridgeApi) ListEventBuses(ctx context.Context, force bool) ([]types.EventBus, error) {
	return f.buses, f.err
}

func (f *fakeEventBridgeApi) ListRules(ctx context.Context, force bool, busArn string) ([]types.Rule, error) {
	return f.rules, f.err
}

func (f *fakeEventBridgeApi) ListTargetsByRule(
	ctx context.Context, force bool, busArn string, ruleName string,
) ([]types.Target, error) {
	f.targetCalls.Add(1)
	if targets, ok := f.targetsByRule[ruleName]; ok {
		return targets, f.targetsErr
	}
	return f.targets, f.targetsErr
}

func (f *fakeEventBridgeApi) ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error) {
	return f.tags, nil
}

func TestEventBusListTable__RefreshLoadsBusesFromApi(t *testing.T) {
	var api = &fakeEventBridgeApi{buses: []types.EventBus{newEventBus("orders")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var table = NewEventBusListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshEventBuss(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		eventBusHeadings,
		{"orders", "orders bus", "2024-01-02 03:04:05"},
	})
}

func TestEventBusListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeEventBridgeApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var table = NewEventBusListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshEventBuss(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestEventBusListTable__FilterByNameShowsMatchingBuses(t *testing.T) {
	var api = &fakeEventBridgeApi{buses: []types.EventBus{newEventBus("orders"), newEventBus("billing")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var table = NewEventBusListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshEventBuss(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterByName("bill")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		eventBusHeadings,
		{"billing", "billing bus", "2024-01-02 03:04:05"},
	})
}
