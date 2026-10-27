package dynamodb

import (
	"aws-tui/internal/ui/dynamodb/dynamodbfake"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/gdamore/tcell/v2"
)

func TestTablesTable__RefreshLoadsTableNamesFromApi(t *testing.T) {
	var api = dynamodbfake.New(dynamodbfake.WithTables([]string{"orders", "users"}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBTablesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshTables(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name"}, {"orders"}, {"users"}})
}

func TestTablesTable__RefreshReportsApiError(t *testing.T) {
	var api = dynamodbfake.New(dynamodbfake.WithErr(errors.New("access denied")))
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
	var api = dynamodbfake.New(dynamodbfake.WithDescription(newDescribedTable()))
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
	var api = dynamodbfake.New(dynamodbfake.WithDescription(newTableDescription("users", hashKey("id"))), dynamodbfake.WithItems([]map[string]any{{"id": "u1"}, {"id": "u2"}}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBGenericTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedTable("users")

	table.ExecuteSearch(DDBTableScan, expression.Expression{}, true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"id"}, {"u1"}, {"u2"}})
}

func TestGenericTable__QueryRunsAgainstSelectedGsi(t *testing.T) {
	var api = dynamodbfake.New(dynamodbfake.WithDescription(newDescriptionWithGsi()), dynamodbfake.WithItems([]map[string]any{{"id": "u1"}}))
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
		api.SetItems([]map[string]any{{"id": "u1"}, {"id": "u2"}})
		var input = table.queryInputView.Input
		input.indexDropDown.SetCurrentOption(1)
		input.pkInput.SetText("a@b.c")
		input.QueryDoneButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil)
	})
	uitest.Eventually(t, serviceCtx.App, rowsAre(3))

	uitest.OnUI(t, serviceCtx.App, func() {
		api.SetItems([]map[string]any{{"id": "u1"}, {"id": "u2"}, {"id": "u3"}})
		table.ExecuteSearch(table.lastTableOp, table.lastSearchExpr, true)
	})
	uitest.Eventually(t, serviceCtx.App, rowsAre(4))

	var want = []string{"byEmail", "byEmail"}
	if got := api.QueriedIndexes(); !slices.Equal(got, want) {
		t.Errorf("queried indexes = %v, want %v (query, then reload)", got, want)
	}
}

func newQueriedTable(t *testing.T, api *dynamodbfake.Api) *DynamoDBGenericTable {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBGenericTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() { table.SetSelectedTable("users") })
	return table
}

func TestGenericTable__NewQueryCancelsQueryInFlight(t *testing.T) {
	var firstCancelled = make(chan error, 1)
	var api = dynamodbfake.New(dynamodbfake.WithDescription(newTableDescription("users", hashKey("id"))), dynamodbfake.WithQueryFunc(func(ctx context.Context, call int) ([]map[string]any, error) {
		if call == 0 {
			<-ctx.Done()
			firstCancelled <- ctx.Err()
			return nil, ctx.Err()
		}
		return []map[string]any{{"id": "second"}}, nil
	}))
	var table = newQueriedTable(t, api)
	var app = table.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })
	uitest.Eventually(t, app, func() bool { return len(api.QueriedIndexes()) == 1 })
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
	var api = dynamodbfake.New(dynamodbfake.WithDescription(newTableDescription("users", hashKey("id"))), dynamodbfake.WithQueryFunc(func(ctx context.Context, call int) ([]map[string]any, error) {
		if call == 0 {
			// Ignores the cancel, as a request that already finished would
			<-releaseFirst
			return []map[string]any{{"id": "stale-1"}, {"id": "stale-2"}}, nil
		}
		<-releaseSecond
		return []map[string]any{{"id": "latest"}}, nil
	}))
	var table = newQueriedTable(t, api)
	var app = table.serviceCtx.App

	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })
	uitest.Eventually(t, app, func() bool { return len(api.QueriedIndexes()) == 1 })
	uitest.OnUI(t, app, func() { table.ExecuteSearch(DDBTableQuery, expression.Expression{}, true) })

	// The second query waits for the first to stop using the api
	uitest.Consistently(t, app, 50*time.Millisecond, func() bool { return len(api.QueriedIndexes()) == 1 },
		"the second query started while the first was in flight")
	close(releaseFirst)
	uitest.Eventually(t, app, func() bool { return len(api.QueriedIndexes()) == 2 })

	// Give the first query's results time to reach the view
	uitest.Consistently(t, app, 300*time.Millisecond, func() bool { return table.GetTable().GetRowCount() == 0 },
		"rows were shown before the latest query finished")

	close(releaseSecond)
	uitest.AwaitRows(t, app, table, []core.TableRow{{"id"}, {"latest"}})
}
