package dynamodb

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type messageViewFixture struct {
	page *DynamoDBItemsPageView
	view *DynamoDBItemMessageView
	app  *tview.Application
}

func newMessageViewFixture(t *testing.T, items ...map[string]any) *messageViewFixture {
	t.Helper()
	var api = &fakeUpdateItemApi{fakeDynamoDBApi: fakeDynamoDBApi{
		description: newTableDescription("orders", hashKey("id")),
		items:       items,
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var page = NewDynamoDBItemsPageView(NewDynamoDBGenericTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(page, true)
		serviceCtx.App.SetFocus(page.MessageView.viewer)
	})
	page.ItemsTable.SetSelectedTable("orders")
	page.UpdateItemView.SetSelectedTable("orders")
	page.ItemsTable.ExecuteSearch(DDBTableScan, expression.Expression{}, true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return page.ItemsTable.GetTable().GetRowCount() == len(items)+1 })
	return &messageViewFixture{page: page, view: page.MessageView, app: serviceCtx.App}
}

func (f *messageViewFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the key is handled by a parent view
	f.app.QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app, func() {})
}

func (f *messageViewFixture) editing(t *testing.T) bool {
	t.Helper()
	var editing bool
	uitest.OnUI(t, f.app, func() { editing = f.view.IsEditing() })
	return editing
}

func TestMessageView__ShowsTheSelectedItemHighlightedAsJson(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1", "status": "new"})

	if !f.view.viewer.JsonHighlighting() {
		t.Error("expected the viewer to highlight json")
	}
	uitest.Eventually(t, f.app, func() bool { return f.view.viewer.GetText() != "" })
}

func TestMessageView__EditorHighlightsJsonToo(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1"})

	if !f.page.UpdateItemView.itemInput.JsonHighlighting() {
		t.Error("expected the editor to highlight json")
	}
}

func TestMessageView__EditKeySwitchesToTheEditorWithTheSelectedItem(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1", "status": "new"})

	f.press(t, tcell.KeyRune, editItemKey)

	if !f.editing(t) {
		t.Fatal("expected the editor to be shown")
	}
	var shown = jsonItem(t, f.page.UpdateItemView.editorText(t))
	if shown["id"] != "a1" || shown["status"] != "new" {
		t.Errorf("editor shows %v", shown)
	}
	uitest.OnUI(t, f.app, func() {
		if !f.page.UpdateItemView.itemInput.HasFocus() {
			t.Error("expected the editor to be focused")
		}
	})
}

func TestMessageView__BackReturnsToTheViewer(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1"})
	f.press(t, tcell.KeyRune, editItemKey)

	uitest.OnUI(t, f.app, f.page.UpdateItemView.onBack)

	if f.editing(t) {
		t.Fatal("expected the viewer to be shown")
	}
	uitest.OnUI(t, f.app, func() {
		if !f.view.viewer.HasFocus() {
			t.Error("expected the viewer to be focused")
		}
	})
}

func TestMessageView__EditKeyIsTypedTextWhileSearching(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1"})
	uitest.OnUI(t, f.app, func() { f.view.viewer.ToggleOverlay(core.SEARCH_OVERLAY_NAME, false) })

	f.press(t, tcell.KeyRune, editItemKey)

	if f.editing(t) {
		t.Error("expected typing e in the search to leave the viewer shown")
	}
}

func TestMessageView__EditKeyIsTypedTextInTheEditor(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1"})
	f.press(t, tcell.KeyRune, editItemKey)
	uitest.OnUI(t, f.app, func() { f.page.UpdateItemView.itemInput.SetText("", false) })

	f.press(t, tcell.KeyRune, editItemKey)

	if got := f.page.UpdateItemView.editorText(t); got != "e" {
		t.Errorf("editor text = %q, want the typed e", got)
	}
	if !f.editing(t) {
		t.Error("expected to stay in the editor")
	}
}

func TestMessageView__EditingWithoutASelectedItemReportsAnError(t *testing.T) {
	var f = newMessageViewFixture(t)
	var callback, received = uitest.CaptureErrors()
	f.view.viewer.ErrorMessageCallback = callback

	f.press(t, tcell.KeyRune, editItemKey)

	if got := uitest.AwaitError(t, received); got != "No item selected" {
		t.Errorf("error = %q", got)
	}
	if f.editing(t) {
		t.Error("expected the viewer to stay shown")
	}
}

func TestMessageView__UnsavedEditsAreKeptWhenReturningToTheEditor(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1"})
	f.press(t, tcell.KeyRune, editItemKey)
	uitest.OnUI(t, f.app, func() {
		f.page.UpdateItemView.itemInput.SetText(`{"id":"a1","mine":true}`, false)
		f.page.UpdateItemView.onBack()
	})

	f.press(t, tcell.KeyRune, editItemKey)

	if shown := jsonItem(t, f.page.UpdateItemView.editorText(t)); shown["mine"] != true {
		t.Errorf("editor shows %v, want the edits kept", shown)
	}
}

func TestMessageView__NestedNavigationFollowsTheMode(t *testing.T) {
	var f = newMessageViewFixture(t, map[string]any{"id": "a1"})

	uitest.OnUI(t, f.app, func() {
		if f.view.NavigateWithin(tcell.KeyCtrlJ) {
			t.Error("the viewer has nothing to navigate within")
		}
		f.app.SetFocus(f.page.ItemsTable)
		f.view.FocusFrom(tcell.KeyCtrlJ)
		if !f.view.viewer.HasFocus() {
			t.Error("expected the viewer to be focused")
		}
	})

	f.press(t, tcell.KeyRune, editItemKey)
	uitest.OnUI(t, f.app, func() {
		f.app.SetFocus(f.page.ItemsTable)
		f.view.FocusFrom(tcell.KeyCtrlJ)
		if !f.page.UpdateItemView.HasFocus() {
			t.Error("expected the editor page to be focused")
		}
	})
}
