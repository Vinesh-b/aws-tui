package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"
)

type fakePutItemApi struct {
	fakeDynamoDBApi
	mtx    sync.Mutex
	table  string
	item   map[string]any
	calls  int
	putErr error
}

func (f *fakePutItemApi) PutItem(ctx context.Context, tableName string, item map[string]any) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls++
	f.table = tableName
	f.item = item
	return f.putErr
}

func (f *fakePutItemApi) putCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.calls
}

func newAddItemView(t *testing.T, api *fakePutItemApi) *DynamoDBAddItemPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var view = NewDynamoDBAddItemPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestAddItemView__AddsTheItemToTheSelectedTable(t *testing.T) {
	var api = &fakePutItemApi{}
	var view = newAddItemView(t, api)
	var added = make(chan string, 1)
	view.SetOnItemAdded(func(table string) { added <- table })
	view.SetSelectedTable("orders")
	view.itemInput.SetText(`{"id": "a1", "big": 12345678901234567890}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

	var table = <-added
	if table != "orders" || api.table != "orders" {
		t.Fatalf("Expected table orders but got %q / %q", table, api.table)
	}
	if api.item["id"] != "a1" || api.item["big"] != json.Number("12345678901234567890") {
		t.Fatalf("Unexpected item %v", api.item)
	}
}

func TestAddItemView__ReportsApiError(t *testing.T) {
	var api = &fakePutItemApi{putErr: errors.New("access denied")}
	var view = newAddItemView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnItemAdded(func(string) { t.Error("item added callback must not run on failure") })
	view.SetSelectedTable("orders")
	view.itemInput.SetText(`{"id": "a1"}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestAddItemView__RejectsInvalidInputWithoutCallingTheApi(t *testing.T) {
	for name, input := range map[string]string{
		"not json":    `{"id":`,
		"not object":  `["a"]`,
		"empty":       `{}`,
		"trailing":    `{"id": "a"} {"id": "b"}`,
		"blank input": ``,
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakePutItemApi{}
			var view = newAddItemView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.itemInput.ErrorMessageCallback = callback
			view.SetSelectedTable("orders")
			view.itemInput.SetText(input, false)

			uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

			uitest.AwaitError(t, received)
			if api.putCalls() != 0 {
				t.Fatalf("Expected no api calls but got %d", api.putCalls())
			}
		})
	}
}

func TestAddItemView__RequiresASelectedTable(t *testing.T) {
	var api = &fakePutItemApi{}
	var view = newAddItemView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.itemInput.ErrorMessageCallback = callback
	view.itemInput.SetText(`{"id": "a1"}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "No table") {
		t.Fatalf("Unexpected error %q", message)
	}
	if api.putCalls() != 0 {
		t.Fatalf("Expected no api calls but got %d", api.putCalls())
	}
}

func TestAddItemView__FormatKeepsLargeNumbers(t *testing.T) {
	var view = newAddItemView(t, &fakePutItemApi{})
	view.itemInput.SetText(`{"big":12345678901234567890}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.FormatItem)

	var text = view.itemInput.GetText()
	if !strings.Contains(text, "\n  \"big\": 12345678901234567890") {
		t.Fatalf("Unexpected formatted text %q", text)
	}
}

func TestItemsPageView__HasAnAddItemTab(t *testing.T) {
	var serviceCtx = uitest.NewServiceContext[awsapi.DynamoDBApi]()
	var view = NewDynamoDBItemsPageView(NewDynamoDBGenericTable(serviceCtx), serviceCtx)

	view.AddItemView.SetSelectedTable("orders")

	if view.AddItemView.selectedTable != "orders" {
		t.Fatalf("Unexpected selected table %q", view.AddItemView.selectedTable)
	}
	if view.TabView.GetTab(DdbTabNameAddItem) == nil {
		t.Fatalf("Expected a %q tab", DdbTabNameAddItem)
	}
}
