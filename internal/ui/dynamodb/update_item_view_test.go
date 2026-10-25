package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func jsonItem(t *testing.T, text string) map[string]any {
	t.Helper()
	var item, err = decodeItemJson(text)
	if err != nil {
		t.Fatalf("bad test item %s: %v", text, err)
	}
	return item
}

func TestDiffItem__WritesOnlyWhatChanged(t *testing.T) {
	var original = map[string]any{
		"id": "a1", "status": "new", "count": float64(30), "note": "x",
		"nested": map[string]any{"a": float64(1)},
	}
	var edited = jsonItem(t, `{"id":"a1","status":"shipped","count":30.0,"added":true,"nested":{"a":2}}`)

	var set, remove, err = diffItem(original, edited, []string{"id"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set) != 3 || set["status"] != "shipped" || set["added"] != true ||
		!reflectEqualJson(set["nested"], `{"a":2}`) {
		t.Errorf("set = %v", set)
	}
	if !slices.Equal(remove, []string{"note"}) {
		t.Errorf("remove = %v, want [note]", remove)
	}
}

func reflectEqualJson(value any, text string) bool {
	var want, _ = decodeJsonValue([]byte(text))
	return sameJsonValue(value, want)
}

func TestDiffItem__UntouchedItemHasNoChanges(t *testing.T) {
	// Binary values, sets and big numbers are shown in the editor in their JSON
	// form and read back from it, which must not count as a change.
	var original = map[string]any{
		"id":     "a1",
		"blob":   []byte("hello"),
		"tags":   []string{"a", "b"},
		"scores": []float64{1, 2.5},
		"big":    float64(12345678901234567890),
		"nested": map[string]any{"blobs": [][]byte{[]byte("x")}},
	}
	var shown, _ = json.MarshalIndent(original, "", "  ")

	var set, remove, err = diffItem(original, jsonItem(t, string(shown)), []string{"id"})

	if err != nil || len(set) != 0 || len(remove) != 0 {
		t.Errorf("diff = %v, %v, %v, want no changes", set, remove, err)
	}
}

func TestDiffItem__KeyCannotChangeOrBeRemoved(t *testing.T) {
	var original = map[string]any{"id": "a1", "ts": float64(5), "x": "y"}

	for name, tc := range map[string]struct{ edited, want string }{
		"partition key changed": {`{"id":"a2","ts":5,"x":"y"}`, `"id" can't be changed`},
		"sort key changed":      {`{"id":"a1","ts":6,"x":"y"}`, `"ts" can't be changed`},
		"partition key removed": {`{"ts":5,"x":"y"}`, `"id" can't be removed`},
		"sort key removed":      {`{"id":"a1","x":"y"}`, `"ts" can't be removed`},
	} {
		t.Run(name, func(t *testing.T) {
			var _, _, err = diffItem(original, jsonItem(t, tc.edited), []string{"id", "ts"})

			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %s", err, tc.want)
			}
		})
	}
}

func TestDiffItem__SameKeyNumberWrittenDifferentlyIsNotAChange(t *testing.T) {
	var _, _, err = diffItem(map[string]any{"ts": float64(5)}, jsonItem(t, `{"ts":5.0}`), []string{"ts"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

type fakeUpdateItemApi struct {
	fakeDynamoDBApi
	mtx       sync.Mutex
	tableName string
	key       map[string]any
	set       map[string]any
	remove    []string
	calls     int
	updateErr error

	deleted   []itemRef
	deleteErr error
}

func (f *fakeUpdateItemApi) DeleteItem(ctx context.Context, tableName string, key map[string]any) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.deleted = append(f.deleted, itemRef{tableName: tableName, key: key})
	return f.deleteErr
}

func (f *fakeUpdateItemApi) deletedItems() []itemRef {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.deleted)
}

func (f *fakeUpdateItemApi) UpdateItem(
	ctx context.Context, tableName string, key map[string]any, set map[string]any, remove []string,
) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls++
	f.tableName, f.key, f.set, f.remove = tableName, key, set, remove
	return f.updateErr
}

func (f *fakeUpdateItemApi) updateCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.calls
}

type updateItemFixture struct {
	table *DynamoDBGenericTable
	api   *fakeUpdateItemApi
}

func newUpdateItemFixture(t *testing.T, api *fakeUpdateItemApi) *updateItemFixture {
	t.Helper()
	api.description = newTableDescription("orders", hashKey("id"))
	if api.items == nil {
		api.items = []map[string]any{{"id": "a1", "status": "new", "note": "x"}, {"id": "a2", "status": "old"}}
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBGenericTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(table, true)
		serviceCtx.App.SetFocus(table)
	})
	table.SetSelectedTable("orders")
	table.ExecuteSearch(DDBTableScan, expression.Expression{}, true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == len(api.items)+1 })
	return &updateItemFixture{table: table, api: api}
}

func (f *updateItemFixture) app() *tview.Application { return f.table.serviceCtx.App }

func (f *updateItemFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app(), func() {})
}

func newUpdateItemPageView(t *testing.T, api *fakeUpdateItemApi, item map[string]any) *DynamoDBUpdateItemPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var view = NewDynamoDBUpdateItemPageView(serviceCtx)
	view.SetSelectedTable("orders")
	view.SetKeyNamesFunc(func() []string { return []string{"id"} })
	view.SetSelectedItemFunc(func() (map[string]any, bool) { return item, item != nil })
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func (v *DynamoDBUpdateItemPageView) editorText(t *testing.T) string {
	t.Helper()
	var text string
	uitest.OnUI(t, v.serviceCtx.App, func() { text = v.itemInput.GetText() })
	return text
}

func TestUpdateItemPageView__LoadsTheSelectedItemAndSavesOnlyTheChanges(t *testing.T) {
	var api = &fakeUpdateItemApi{}
	var view = newUpdateItemPageView(t, api, map[string]any{"id": "a1", "status": "new", "note": "x"})
	var updated = make(chan string, 1)
	view.SetOnItemUpdated(func(table string) { updated <- table })

	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItem)
	var shown = jsonItem(t, view.editorText(t))
	if shown["id"] != "a1" || shown["status"] != "new" || shown["note"] != "x" {
		t.Fatalf("editor shows %v", shown)
	}
	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.itemInput.SetText(`{"id":"a1","status":"shipped","extra":1}`, false)
		view.UpdateItem()
	})

	if table := <-updated; table != "orders" {
		t.Errorf("updated table = %q", table)
	}
	if api.tableName != "orders" || !reflectEqualJson(api.key, `{"id":"a1"}`) {
		t.Errorf("Unexpected target %q %v", api.tableName, api.key)
	}
	if !reflectEqualJson(api.set, `{"status":"shipped","extra":1}`) || !slices.Equal(api.remove, []string{"note"}) {
		t.Errorf("Unexpected update set=%v remove=%v", api.set, api.remove)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Item updated in orders")
	})
}

func TestUpdateItemPageView__FurtherEditsAreComparedAgainstWhatWasSaved(t *testing.T) {
	var api = &fakeUpdateItemApi{}
	var view = newUpdateItemPageView(t, api, map[string]any{"id": "a1", "status": "new"})
	var updated = make(chan string, 2)
	view.SetOnItemUpdated(func(table string) { updated <- table })
	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.LoadSelectedItem()
		view.itemInput.SetText(`{"id":"a1","status":"shipped"}`, false)
		view.UpdateItem()
	})
	<-updated

	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.itemInput.SetText(`{"id":"a1","status":"shipped","note":"x"}`, false)
		view.UpdateItem()
	})

	<-updated
	if !reflectEqualJson(api.set, `{"note":"x"}`) || len(api.remove) != 0 {
		t.Errorf("Unexpected update set=%v remove=%v", api.set, api.remove)
	}
}

func TestUpdateItemPageView__NoChangesDoesNotCallTheApi(t *testing.T) {
	var api = &fakeUpdateItemApi{}
	var view = newUpdateItemPageView(t, api, map[string]any{"id": "a1", "status": "new"})

	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.LoadSelectedItem()
		view.UpdateItem()
	})

	uitest.Eventually(t, view.serviceCtx.App, func() bool { return view.resultOutput.GetText() == "No changes to save" })
	if api.updateCalls() != 0 {
		t.Errorf("Expected no api calls but got %d", api.updateCalls())
	}
}

func TestUpdateItemPageView__InvalidEditsAreReportedWithoutCallingTheApi(t *testing.T) {
	for name, tc := range map[string]struct{ text, want string }{
		"not json":    {`{"id":`, "unexpected EOF"},
		"key changed": {`{"id":"zzz","status":"new"}`, "can't be changed"},
		"key removed": {`{"status":"new"}`, "can't be removed"},
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakeUpdateItemApi{}
			var view = newUpdateItemPageView(t, api, map[string]any{"id": "a1", "status": "new"})
			var callback, received = uitest.CaptureErrors()
			view.itemInput.ErrorMessageCallback = callback

			uitest.OnUI(t, view.serviceCtx.App, func() {
				view.LoadSelectedItem()
				view.itemInput.SetText(tc.text, false)
				view.UpdateItem()
			})

			if message := uitest.AwaitError(t, received); !strings.Contains(message, tc.want) {
				t.Errorf("error = %q, want one containing %q", message, tc.want)
			}
			if api.updateCalls() != 0 {
				t.Errorf("Expected no api calls but got %d", api.updateCalls())
			}
		})
	}
}

func TestUpdateItemPageView__ReportsApiError(t *testing.T) {
	var api = &fakeUpdateItemApi{updateErr: errors.New("ConditionalCheckFailed")}
	var view = newUpdateItemPageView(t, api, map[string]any{"id": "a1", "status": "new"})
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnItemUpdated(func(string) { t.Error("updated callback must not run on failure") })

	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.LoadSelectedItem()
		view.itemInput.SetText(`{"id":"a1","status":"x"}`, false)
		view.UpdateItem()
	})

	if message := uitest.AwaitError(t, received); message != "ConditionalCheckFailed" {
		t.Errorf("error = %q", message)
	}
}

func TestUpdateItemPageView__RequiresATableAndALoadedItem(t *testing.T) {
	for name, tc := range map[string]struct {
		setup func(view *DynamoDBUpdateItemPageView)
		want  string
	}{
		"no table":       {func(v *DynamoDBUpdateItemPageView) { v.SetSelectedTable("") }, "No table selected"},
		"nothing loaded": {func(v *DynamoDBUpdateItemPageView) {}, "No item loaded"},
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakeUpdateItemApi{}
			var view = newUpdateItemPageView(t, api, map[string]any{"id": "a1"})
			var callback, received = uitest.CaptureErrors()
			view.itemInput.ErrorMessageCallback = callback
			tc.setup(view)

			uitest.OnUI(t, view.serviceCtx.App, view.UpdateItem)

			if message := uitest.AwaitError(t, received); !strings.Contains(message, tc.want) {
				t.Errorf("error = %q, want one containing %q", message, tc.want)
			}
			if api.updateCalls() != 0 {
				t.Errorf("Expected no api calls but got %d", api.updateCalls())
			}
		})
	}
}

func TestUpdateItemPageView__LoadWithoutASelectedItemReportsAnError(t *testing.T) {
	var view = newUpdateItemPageView(t, &fakeUpdateItemApi{}, nil)
	var callback, received = uitest.CaptureErrors()
	view.itemInput.ErrorMessageCallback = callback

	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItem)

	if message := uitest.AwaitError(t, received); message != "No item selected" {
		t.Errorf("error = %q", message)
	}
}

func TestUpdateItemPageView__ShowingTheTabKeepsUnsavedEdits(t *testing.T) {
	var item = map[string]any{"id": "a1", "status": "new"}
	var view = newUpdateItemPageView(t, &fakeUpdateItemApi{}, item)

	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItemIfUnedited)
	if shown := jsonItem(t, view.editorText(t)); shown["status"] != "new" {
		t.Fatalf("editor shows %v", shown)
	}

	// The selection moves on while the editor is untouched
	view.SetSelectedItemFunc(func() (map[string]any, bool) { return map[string]any{"id": "a2"}, true })
	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItemIfUnedited)
	if shown := jsonItem(t, view.editorText(t)); shown["id"] != "a2" {
		t.Fatalf("editor shows %v, want the newly selected item", shown)
	}

	// With edits made, it is left alone
	uitest.OnUI(t, view.serviceCtx.App, func() { view.itemInput.SetText(`{"id":"a2","mine":true}`, false) })
	view.SetSelectedItemFunc(func() (map[string]any, bool) { return map[string]any{"id": "a3"}, true })
	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItemIfUnedited)
	if shown := jsonItem(t, view.editorText(t)); shown["mine"] != true {
		t.Fatalf("editor shows %v, want the edits kept", shown)
	}

	// The explicit load always replaces them
	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItem)
	if shown := jsonItem(t, view.editorText(t)); shown["id"] != "a3" {
		t.Fatalf("editor shows %v, want the selected item", shown)
	}
}

func TestUpdateItemPageView__EmptySelectionLeavesTheEditorAloneWhenShown(t *testing.T) {
	var view = newUpdateItemPageView(t, &fakeUpdateItemApi{}, nil)
	var callback, received = uitest.CaptureErrors()
	view.itemInput.ErrorMessageCallback = callback

	uitest.OnUI(t, view.serviceCtx.App, view.LoadSelectedItemIfUnedited)

	select {
	case message := <-received:
		t.Fatalf("Unexpected error %q", message)
	default:
	}
}
