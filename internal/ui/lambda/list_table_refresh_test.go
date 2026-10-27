package lambda

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type fakeLambdaApi struct {
	awsapi.LambdaApi
	functions []types.FunctionConfiguration
	err       error
	listCalls atomic.Int32

	// mappings, policies, versions and aliases are by lambda ARN, and delays
	// hold them back
	mappings  map[string][]types.EventSourceMappingConfiguration
	versions  map[string][]types.FunctionConfiguration
	aliases   map[string][]types.AliasConfiguration
	tags      map[string]map[string]string
	tagsErr   error
	policies  map[string]string
	policyErr error
	delays    map[string]time.Duration

	mu    sync.Mutex
	asked []string
}

// askedFor records the calls made for a lambda, as "<call> <arn>"
func (f *fakeLambdaApi) askedFor(call string, lambdaArn string) {
	f.mu.Lock()
	f.asked = append(f.asked, call+" "+lambdaArn)
	f.mu.Unlock()
}

func (f *fakeLambdaApi) wasAsked(call string, lambdaArn string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Contains(f.asked, call+" "+lambdaArn)
}

func (f *fakeLambdaApi) ListEventSourceMappings(
	ctx context.Context, lambdaArn string,
) ([]types.EventSourceMappingConfiguration, error) {
	f.askedFor("ListEventSourceMappings", lambdaArn)
	time.Sleep(f.delays[lambdaArn]) // sleep-ok: simulated api latency
	return f.mappings[lambdaArn], f.err
}

func (f *fakeLambdaApi) GetPolicy(ctx context.Context, lambdaArn string) (string, error) {
	f.askedFor("GetPolicy", lambdaArn)
	time.Sleep(f.delays[lambdaArn]) // sleep-ok: simulated api latency
	return f.policies[lambdaArn], f.policyErr
}

func (f *fakeLambdaApi) ListLambdas(ctx context.Context, force bool) ([]types.FunctionConfiguration, error) {
	f.listCalls.Add(1)
	return f.functions, f.err
}

func newRunningListTable(t *testing.T, api awsapi.LambdaApi) *LambdaListTable {
	var serviceCtx = uitest.NewServiceContextWith(api)
	var table = NewLambdasListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return table
}

func TestListTable__RefreshLoadsFunctionsFromApi(t *testing.T) {
	var api = &fakeLambdaApi{functions: []types.FunctionConfiguration{newFunction("alpha"), newFunction("beta")}}
	var table = newRunningListTable(t, api)

	table.RefreshLambdas(true)

	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, table.serviceCtx.App, func() {
		uitest.AssertTableRows(t, table, []core.TableRow{
			{"Name", "LastModified"},
			{"alpha", "2024-01-02T03:04:05.000+0000"},
			{"beta", "2024-01-02T03:04:05.000+0000"},
		})
	})
}

func TestListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeLambdaApi{err: errors.New("access denied")}
	var table = newRunningListTable(t, api)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshLambdas(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestListTable__FilterByNameShowsMatchingFunctions(t *testing.T) {
	var api = &fakeLambdaApi{functions: []types.FunctionConfiguration{newFunction("orders"), newFunction("billing")}}
	var table = newRunningListTable(t, api)
	table.RefreshLambdas(true)
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterByName("bill")

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return table.GetTable().GetRowCount() == 2 && table.GetCellText(1, 0) == "billing"
	})
}

func (f *fakeLambdaApi) ListTags(ctx context.Context, lambdaArn string) (map[string]string, error) {
	f.askedFor("ListTags", lambdaArn)
	time.Sleep(f.delays[lambdaArn]) // sleep-ok: simulated api latency
	return f.tags[lambdaArn], f.tagsErr
}

func (f *fakeLambdaApi) ListVersions(ctx context.Context, lambdaArn string) ([]types.FunctionConfiguration, error) {
	f.askedFor("ListVersions", lambdaArn)
	time.Sleep(f.delays[lambdaArn]) // sleep-ok: simulated api latency
	return f.versions[lambdaArn], f.err
}

func (f *fakeLambdaApi) ListAliases(ctx context.Context, lambdaArn string) ([]types.AliasConfiguration, error) {
	f.askedFor("ListAliases", lambdaArn)
	time.Sleep(f.delays[lambdaArn]) // sleep-ok: simulated api latency
	return f.aliases[lambdaArn], f.err
}
