package dynamodb

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

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
	err         error

	mu           sync.Mutex
	queryIndexes []string
}

func (f *fakeDynamoDBApi) ListTables(ctx context.Context, force bool) ([]string, error) {
	return f.tables, f.err
}

func (f *fakeDynamoDBApi) DescribeTable(ctx context.Context, tableName string) (*types.TableDescription, error) {
	return f.description, f.err
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
	f.mu.Unlock()
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
		{"Creation Date", "2024-01-02 03:04:05"},
		{"Item Count", "42"},
		{"Billing Mode", "PROVISIONED"},
		{"Delete Protection", "DISABLED"},
		{"Arn", "arn:aws:dynamodb:us-east-1:123456789012:table/orders"},
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
