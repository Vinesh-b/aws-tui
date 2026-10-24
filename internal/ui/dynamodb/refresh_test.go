package dynamodb

import (
	"context"
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamoDBApi struct {
	awsapi.DynamoDBApi
	tables      []string
	description *types.TableDescription
	items       []map[string]any
	err         error
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
