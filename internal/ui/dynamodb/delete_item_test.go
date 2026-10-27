package dynamodb

import (
	"aws-tui/internal/ui/dynamodb/dynamodbfake"
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
)

func (f *updateItemFixture) openDelete(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyRune, deleteItemKey)
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

// confirmDelete moves focus from Cancel to Delete and presses it
func (f *updateItemFixture) confirmDelete(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyDown, 0)
	f.press(t, tcell.KeyEnter, 0)
}

func (f *updateItemFixture) deleteMessage(t *testing.T) string {
	t.Helper()
	var message string
	uitest.OnUI(t, f.app(), func() { message = f.table.DeleteView.Input.GetMessage() })
	return message
}

func TestDeleteItem__ConfirmingDeletesTheSelectedItemAndReloads(t *testing.T) {
	var f = newUpdateItemFixture(t, dynamodbfake.New())
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.openDelete(t)
	if got := f.deleteMessage(t); !strings.Contains(got, `Delete the item {"id":"a2"} from orders?`) {
		t.Errorf("message = %q", got)
	}
	f.api.SetItems(f.api.Items()[:1])
	f.confirmDelete(t)

	// The order of the other columns isn't fixed, so only the key is checked
	uitest.Eventually(t, f.app(), func() bool {
		return f.table.GetTable().GetRowCount() == 2 && f.table.GetCellText(1, 0) == "a1"
	})
	var deleted = f.api.Deleted()
	if len(deleted) != 1 || deleted[0].TableName != "orders" || !reflectEqualJson(deleted[0].Key, `{"id":"a2"}`) {
		t.Errorf("Unexpected deletes %v", deleted)
	}
	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
}

func TestDeleteItem__CancellingDeletesNothing(t *testing.T) {
	var f = newUpdateItemFixture(t, dynamodbfake.New())

	f.openDelete(t)
	// Focus starts on Cancel, so Enter alone must not delete
	f.press(t, tcell.KeyEnter, 0)

	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
	if deleted := f.api.Deleted(); len(deleted) != 0 {
		t.Errorf("Unexpected deletes %v", deleted)
	}
}

func TestDeleteItem__TheKeyIncludesTheSortKey(t *testing.T) {
	var api = dynamodbfake.New(dynamodbfake.WithItems([]map[string]any{{"id": "a1", "ts": float64(5), "x": "y"}}))
	var f = newUpdateItemFixture(t, api)
	api.SetDescription(newTableDescription("orders", hashKey("id"), rangeKey("ts")))
	f.table.ExecuteSearch(DDBTableScan, f.table.lastSearchExpr, true)
	uitest.Eventually(t, f.app(), func() bool { return f.table.skName == "ts" })

	f.openDelete(t)
	f.confirmDelete(t)

	uitest.Eventually(t, f.app(), func() bool { return len(api.Deleted()) == 1 })
	if key := api.Deleted()[0].Key; len(key) != 2 || key["id"] != "a1" || key["ts"] != float64(5) {
		t.Errorf("Unexpected key %v", key)
	}
}

func TestDeleteItem__ReportsApiError(t *testing.T) {
	var f = newUpdateItemFixture(t, dynamodbfake.New(dynamodbfake.WithDeleteErr(errors.New("ConditionalCheckFailed"))))
	var callback, received = uitest.CaptureErrors()
	f.table.ErrorMessageCallback = callback

	f.openDelete(t)
	f.confirmDelete(t)

	if got := uitest.AwaitError(t, received); got != "ConditionalCheckFailed" {
		t.Errorf("error = %q", got)
	}
}

func TestDeleteItem__NothingSelectedReportsAnError(t *testing.T) {
	var f = newUpdateItemFixture(t, dynamodbfake.New(dynamodbfake.WithItems([]map[string]any{})))
	var callback, received = uitest.CaptureErrors()
	f.table.ErrorMessageCallback = callback

	f.openDelete(t)
	if got := f.deleteMessage(t); got != "No item selected" {
		t.Errorf("message = %q", got)
	}
	f.confirmDelete(t)

	if got := uitest.AwaitError(t, received); got != "No item selected" {
		t.Errorf("error = %q", got)
	}
	if deleted := f.api.Deleted(); len(deleted) != 0 {
		t.Errorf("Unexpected deletes %v", deleted)
	}
}
