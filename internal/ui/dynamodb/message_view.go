package dynamodb

import (
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// editItemKey switches the message view to the editor for the selected item.
const editItemKey = 'e'

const (
	messageViewViewerPage = "VIEWER"
	messageViewEditorPage = "EDITOR"
)

// DynamoDBItemMessageView shows the selected item as highlighted json and, with
// e, switches to an editor for it that highlights the json as well. Back
// returns to the viewer, which keeps following the selection.
type DynamoDBItemMessageView struct {
	*tview.Pages
	viewer  *core.SearchableTextView
	editor  *DynamoDBUpdateItemPageView
	app     *tview.Application
	editing bool
}

func NewDynamoDBItemMessageView(
	appCtx *core.AppContext,
	viewer *core.SearchableTextView,
	editor *DynamoDBUpdateItemPageView,
) *DynamoDBItemMessageView {
	var view = &DynamoDBItemMessageView{
		Pages:  tview.NewPages(),
		viewer: viewer,
		editor: editor,
		app:    appCtx.App,
	}

	view.
		AddPage(messageViewViewerPage, viewer, true, true).
		AddPage(messageViewEditorPage, editor, true, false)

	viewer.HelpView.View.AddItem(string(editItemKey), "Edit the selected item", nil)
	editor.SetOnBack(view.ShowViewer)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Not while searching or reading the help, where e is typed text
		if !view.editing && !viewer.IsAnOverlayVisible() &&
			event.Key() == tcell.KeyRune && event.Rune() == editItemKey {
			view.EditItem()
			return nil
		}
		return event
	})

	return view
}

func (inst *DynamoDBItemMessageView) IsEditing() bool {
	return inst.editing
}

// EditItem switches to the editor with the selected item. Edits that were not
// saved are kept rather than replaced by the item.
func (inst *DynamoDBItemMessageView) EditItem() {
	if _, ok := inst.editor.selectedItem(); !ok && !inst.editor.hasUnsavedEdits() {
		inst.viewer.ErrorMessageCallback("No item selected")
		return
	}

	inst.editor.LoadSelectedItemIfUnedited()
	inst.editing = true
	inst.SwitchToPage(messageViewEditorPage)
	inst.app.SetFocus(inst.editor)
}

func (inst *DynamoDBItemMessageView) ShowViewer() {
	inst.editing = false
	inst.SwitchToPage(messageViewViewerPage)
	inst.app.SetFocus(inst.viewer)
}

func (inst *DynamoDBItemMessageView) NavigateWithin(key tcell.Key) bool {
	if inst.editing {
		return inst.editor.NavigateWithin(key)
	}
	return false
}

func (inst *DynamoDBItemMessageView) FocusFrom(key tcell.Key) {
	if inst.editing {
		inst.editor.FocusFrom(key)
		return
	}
	inst.app.SetFocus(inst.viewer)
}
