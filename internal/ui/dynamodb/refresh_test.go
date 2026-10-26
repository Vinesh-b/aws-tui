package dynamodb

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gdamore/tcell/v2"
)

type fakeDynamoDBApi struct {
	awsapi.DynamoDBApi
	tables      []string
	description *types.TableDescription
	items       []map[string]any
	tags        []types.Tag
	err         error

	// queryFunc, when set, gives the result of each query by its call number
	queryFunc func(ctx context.Context, call int) ([]map[string]any, error)

	mu           sync.Mutex
	queryIndexes []string
}

func (f *fakeDynamoDBApi) ListTables(ctx context.Context, force bool) ([]string, error) {
	return f.tables, f.err
}

func (f *fakeDynamoDBApi) DescribeTable(ctx context.Context, tableName string) (*types.TableDescription, error) {
	return f.description, f.err
}

func (f *fakeDynamoDBApi) ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error) {
	return f.tags, f.err
}

func (f *fakeDynamoDBApi) ScanTable(
	ctx context.Context, tableName string, scanExpression expression.Expression, indexName string, force bool,
) ([]map[string]any, error) {
	return f.items, f.err
}

func (f *fakeDynamoDBApi) QueryTable(
	ctx context.Context, tableName string, queryExpression expression.Expression, indexName string, force bool,
) ([]map[string]any, error) {
	f.mu.Lock()
	f.queryIndexes = append(f.queryIndexes, indexName)
	var call = len(f.queryIndexes) - 1
	f.mu.Unlock()
	if f.queryFunc != nil {
		return f.queryFunc(ctx, call)
	}
	return f.items, f.err
}

func (f *fakeDynamoDBApi) queriedIndexes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.queryIndexes...)
}

func TestTablesTable__RefreshLoadsTableNamesFromApi(t *testing.T) {
	var api = &fakeDynamoDBApi{tables: []string{"orders", "users"}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBTablesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshTables(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name"}, {"orders"}, {"users"}})
}

func TestTablesTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeDynamoDBApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBTablesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshTables(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestDetailsTable__RefreshLoadsDescriptionOfSelectedTable(t *testing.T) {
	var api = &fakeDynamoDBApi{description: newDescribedTable()}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedTable("orders")

	table.RefreshDetails()

	uitest.AwaitDetailsRows(t, serviceCtx.App, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Status", "ACTIVE"},
		{"Created", "2024-01-02 03:04:05"},
		{"Item Count", "42"},
		{"Billing Mode", "PROVISIONED"},
		{"Delete Protection", "DISABLED"},
		{"ARN", "arn:aws:dynamodb:us-east-1:123456789012:table/orders"},
		{"Keys"},
		{"Partition Key", "id"},
		{"Sort Key", "ts"},
	})
}

func TestGenericTable__ScanShowsItemsFromApi(t *testing.T) {
	var api = &fakeDynamoDBApi{
		description: newTableDescription("users", hashKey("id")),
		items:       []map[string]any{{"id": "u1"}, {"id": "u2"}},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBGenericTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedTable("users")

	table.ExecuteSearch(DDBTableScan, expression.Expression{}, true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"id"}, {"u1"}, {"u2"}})
}

func TestGenericTable__QueryRunsAgainstSelectedGsi(t *testing.T) {
	var api = &fakeDynamoDBApi{
		description: newDescriptionWithGsi(),
		items:       []map[string]any{{"id": "u1"}},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBGenericTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var rowsAre = func(count int) func() bool {
		return func() bool { return table.GetTable().GetRowCount() == count }
	}
	table.SetSelectedTable("users")
	table.ExecuteSearch(DDBTableScan, expression.Expression{}, true)
	uitest.Eventually(t, serviceCtx.App, rowsAre(2))

	// Each load returns a different number of items so the test can wait for
	// it to finish before starting the next one.
	uitest.OnUI(t, serviceCtx.App, func() {
		api.items = []map[string]any{{"id": "u1"}, {"id": "u2"}}
		var input = table.queryInputView.Input
		input.indexDropDown.SetCurrentOption(1)
		input.pkInput.SetText("a@b.c")
		input.QueryDoneButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil)
	})
	uitest.Eventually(t, serviceCtx.App, rowsAre(3))

	uitest.OnUI(t, serviceCtx.App, func() {
		api.items = []map[string]any{{"id": "u1"}, {"id": "u2"}, {"id": "u3"}}
		table.ExecuteSearch(table.lastTableOp, table.lastSearchExpr, true)
	})
	uitest.Eventually(t, serviceCtx.App, rowsAre(4))

	var want = []string{"byEmail", "byEmail"}
	if got := api.queriedIndexes(); !slices.Equal(got, want) {
		t.Errorf("queried indexes = %v, want %v (query, then reload)", got, want)
	}
}

func newQueriedTable(t *testing.T, api *fakeDynamoDBApi) *DynamoDBGenericTable {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBGenericTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() { table.SetSelectedTable("users") })
	return table
}

func TestGenericTable__NewQueryCancelsQueryInFlight(t *testing.T) {
	var firstCancelled = make(chan error, 1)
	var api = &fakeDynamoDBApi{
		description: newTableDescription("users", hashKey("id")),
		queryFunc: func(ctx context.Context, call int) ([]map[string]any, error) {
			if call == 0 {
				<-ctx.Done()
				firstCancelled <- ctx.Err()
				return nil, ctx.Err()
			}
			return []map[string]any{{"id": "second"}}, nil
		},
	}
	var table = newQueriedTable(t, api)
	var app = table.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })
	uitest.Eventually(t, app, func() bool { return len(api.queriedIndexes()) == 1 })
	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })

	select {
	case err := <-firstCancelled:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("first query ctx error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first query was not cancelled")
	}
	uitest.AwaitRows(t, app, table, []core.TableRow{{"id"}, {"second"}})
	select {
	case msg := <-received:
		t.Errorf("cancelled query reported error %q", msg)
	default:
	}
}

func TestGenericTable__ResultsOfReplacedQueryAreDropped(t *testing.T) {
	var releaseFirst = make(chan struct{})
	var releaseSecond = make(chan struct{})
	var api = &fakeDynamoDBApi{
		description: newTableDescription("users", hashKey("id")),
		queryFunc: func(ctx context.Context, call int) ([]map[string]any, error) {
			if call == 0 {
				// Ignores the cancel, as a request that already finished would
				<-releaseFirst
				return []map[string]any{{"id": "stale-1"}, {"id": "stale-2"}}, nil
			}
			<-releaseSecond
			return []map[string]any{{"id": "latest"}}, nil
		},
	}
	var table = newQueriedTable(t, api)
	var app = table.serviceCtx.App

	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })
	uitest.Eventually(t, app, func() bool { return len(api.queriedIndexes()) == 1 })
	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })

	// The second query waits for the first to stop using the api
	time.Sleep(50 * time.Millisecond)
	if got := len(api.queriedIndexes()); got != 1 {
		t.Fatalf("queries started = %d, want 1 while the first is in flight", got)
	}
	close(releaseFirst)
	uitest.Eventually(t, app, func() bool { return len(api.queriedIndexes()) == 2 })

	// Give the first query's results time to reach the view
	time.Sleep(300 * time.Millisecond)
	uitest.OnUI(t, app, func() {
		if got := table.GetTable().GetRowCount(); got != 0 {
			t.Errorf("rows = %d before the latest query finished, want 0", got)
		}
	})

	close(releaseSecond)
	uitest.AwaitRows(t, app, table, []core.TableRow{{"id"}, {"latest"}})
}
