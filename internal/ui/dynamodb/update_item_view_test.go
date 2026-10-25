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
	"aws-tui/internal/ui/core"
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

func (f *updateItemFixture) view() *DynamoDBUpdateItemView { return f.table.UpdateView.Input }

func (f *updateItemFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app(), func() {})
}

func (f *updateItemFixture) open(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyRune, updateItemKey)
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

// save moves focus from the editor to the Save button and presses it
func (f *updateItemFixture) save(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyTab, 0)
	f.press(t, tcell.KeyEnter, 0)
}

func (f *updateItemFixture) editor(t *testing.T) string {
	t.Helper()
	var text string
	uitest.OnUI(t, f.app(), func() { text = f.view().itemInput.GetText() })
	return text
}

func (f *updateItemFixture) statusNow() string { return f.view().message.GetText(true) }

func TestUpdateItemView__ShowsTheSelectedItemAndSavesOnlyTheChanges(t *testing.T) {
	var f = newUpdateItemFixture(t, &fakeUpdateItemApi{})
	uitest.OnUI(t, f.app(), func() { f.table.Select(1, 0) })

	f.open(t)
	var shown = jsonItem(t, f.editor(t))
	if shown["id"] != "a1" || shown["status"] != "new" || shown["note"] != "x" {
		t.Fatalf("editor shows %v", shown)
	}
	uitest.OnUI(t, f.app(), func() {
		f.view().itemInput.SetText(`{"id":"a1","status":"shipped","extra":1}`, false)
	})
	f.save(t)

	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
	if f.api.tableName != "orders" || !reflectEqualJson(f.api.key, `{"id":"a1"}`) {
		t.Errorf("Unexpected target %q %v", f.api.tableName, f.api.key)
	}
	if !reflectEqualJson(f.api.set, `{"status":"shipped","extra":1}`) || !slices.Equal(f.api.remove, []string{"note"}) {
		t.Errorf("Unexpected update set=%v remove=%v", f.api.set, f.api.remove)
	}
}

func TestUpdateItemView__NoChangesDoesNotCallTheApi(t *testing.T) {
	var f = newUpdateItemFixture(t, &fakeUpdateItemApi{})
	f.open(t)

	f.save(t)

	uitest.Eventually(t, f.app(), func() bool { return f.statusNow() == "No changes to save" })
	if f.api.updateCalls() != 0 {
		t.Errorf("Expected no api calls but got %d", f.api.updateCalls())
	}
}

func TestUpdateItemView__InvalidEditsAreReportedWithoutCallingTheApi(t *testing.T) {
	for name, tc := range map[string]struct{ text, want string }{
		"not json":    {`{"id":`, "unexpected EOF"},
		"key changed": {`{"id":"zzz","status":"new","note":"x"}`, "can't be changed"},
		"key removed": {`{"status":"new"}`, "can't be removed"},
	} {
		t.Run(name, func(t *testing.T) {
			var f = newUpdateItemFixture(t, &fakeUpdateItemApi{})
			f.open(t)
			uitest.OnUI(t, f.app(), func() { f.view().itemInput.SetText(tc.text, false) })

			f.save(t)

			uitest.Eventually(t, f.app(), func() bool { return strings.Contains(f.statusNow(), tc.want) })
			if f.api.updateCalls() != 0 {
				t.Errorf("Expected no api calls but got %d", f.api.updateCalls())
			}
			uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
		})
	}
}

func TestUpdateItemView__ReportsApiErrorAndStaysOpen(t *testing.T) {
	var f = newUpdateItemFixture(t, &fakeUpdateItemApi{updateErr: errors.New("ConditionalCheckFailed")})
	f.open(t)
	uitest.OnUI(t, f.app(), func() { f.view().itemInput.SetText(`{"id":"a1","status":"x"}`, false) })

	f.save(t)

	uitest.Eventually(t, f.app(), func() bool { return f.statusNow() == "ConditionalCheckFailed" })
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

func TestUpdateItemView__NothingSelectedReportsAnError(t *testing.T) {
	var f = newUpdateItemFixture(t, &fakeUpdateItemApi{fakeDynamoDBApi: fakeDynamoDBApi{items: []map[string]any{}}})
	f.open(t)

	f.save(t)

	uitest.Eventually(t, f.app(), func() bool { return f.statusNow() == "No item selected" })
	if f.api.updateCalls() != 0 {
		t.Errorf("Expected no api calls but got %d", f.api.updateCalls())
	}
}

func TestUpdateItemView__SavedChangesReloadTheItems(t *testing.T) {
	var f = newUpdateItemFixture(t, &fakeUpdateItemApi{})
	f.api.items = []map[string]any{{"id": "a1", "status": "shipped"}}
	f.open(t)
	uitest.OnUI(t, f.app(), func() { f.view().itemInput.SetText(`{"id":"a1","status":"shipped"}`, false) })

	f.save(t)

	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{{"id", "status"}, {"a1", "shipped"}})
}
