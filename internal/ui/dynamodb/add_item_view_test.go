package dynamodb

import (
	"aws-tui/internal/ui/dynamodb/dynamodbfake"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"
)

func newAddItemView(t *testing.T, api *dynamodbfake.Api) *DynamoDBAddItemPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var view = NewDynamoDBAddItemPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestAddItemView__AddsTheItemToTheSelectedTable(t *testing.T) {
	var api = dynamodbfake.New()
	var view = newAddItemView(t, api)
	var added = make(chan string, 1)
	view.SetOnItemAdded(func(table string) { added <- table })
	view.SetSelectedTable("orders")
	view.itemInput.SetText(`{"id": "a1", "big": 12345678901234567890}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

	var table = <-added
	var puts = api.Puts()
	if len(puts) != 1 {
		t.Fatalf("Expected one put but got %v", puts)
	}
	if table != "orders" || puts[0].TableName != "orders" {
		t.Fatalf("Expected table orders but got %q / %q", table, puts[0].TableName)
	}
	if item := puts[0].Item; item["id"] != "a1" || item["big"] != json.Number("12345678901234567890") {
		t.Fatalf("Unexpected item %v", item)
	}
}

func TestAddItemView__ReportsApiError(t *testing.T) {
	var api = dynamodbfake.New(dynamodbfake.WithPutErr(errors.New("access denied")))
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
			var api = dynamodbfake.New()
			var view = newAddItemView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.itemInput.ErrorMessageCallback = callback
			view.SetSelectedTable("orders")
			view.itemInput.SetText(input, false)

			uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

			uitest.AwaitError(t, received)
			if len(api.Puts()) != 0 {
				t.Fatalf("Expected no api calls but got %d", len(api.Puts()))
			}
		})
	}
}

func TestAddItemView__RequiresASelectedTable(t *testing.T) {
	var api = dynamodbfake.New()
	var view = newAddItemView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.itemInput.ErrorMessageCallback = callback
	view.itemInput.SetText(`{"id": "a1"}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.AddItem)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "No table") {
		t.Fatalf("Unexpected error %q", message)
	}
	if len(api.Puts()) != 0 {
		t.Fatalf("Expected no api calls but got %d", len(api.Puts()))
	}
}

func TestAddItemView__FormatKeepsLargeNumbers(t *testing.T) {
	var view = newAddItemView(t, dynamodbfake.New())
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

func TestAddItemView__CopiesTheSelectedItemAsATemplate(t *testing.T) {
	var view = newAddItemView(t, dynamodbfake.New())
	view.SetSelectedItemFunc(func() (map[string]any, bool) {
		return map[string]any{"id": "u1", "age": float64(30)}, true
	})
	view.itemInput.SetText("old", false)

	uitest.OnUI(t, view.serviceCtx.App, view.CopySelectedItem)

	var item, err = decodeItemJson(view.itemInput.GetText())
	if err != nil || item["id"] != "u1" || item["age"] != json.Number("30") {
		t.Fatalf("Unexpected editor contents %q (%v)", view.itemInput.GetText(), err)
	}
}

func TestAddItemView__CopyWithoutSelectedItemReportsError(t *testing.T) {
	var view = newAddItemView(t, dynamodbfake.New())
	var callback, received = uitest.CaptureErrors()
	view.itemInput.ErrorMessageCallback = callback
	view.itemInput.SetText("keep", false)

	uitest.OnUI(t, view.serviceCtx.App, view.CopySelectedItem)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "No item selected") {
		t.Fatalf("Unexpected error %q", message)
	}
	if view.itemInput.GetText() != "keep" {
		t.Fatalf("Expected the editor to be untouched")
	}
}
