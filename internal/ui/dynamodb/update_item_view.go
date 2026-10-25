package dynamodb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// updateItemKey opens the update view for the selected item on the items table.
const updateItemKey = 'e'

const updateItemOverlayId = "UPDATE_ITEM"

type DynamoDBUpdateItemView struct {
	*tview.Flex
	itemInput    *core.TextArea
	message      *tview.TextView
	saveButton   *core.Button
	closeButton  *core.Button
	tabNavigator *core.ViewNavigation1D
	orderedViews []core.View

	// source gives the item selected in the items table
	source   func() (map[string]any, bool)
	original map[string]any
	onSave   func(original map[string]any, edited map[string]any)
}

func NewDynamoDBUpdateItemView(appCtx *core.AppContext) *DynamoDBUpdateItemView {
	var layout = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &DynamoDBUpdateItemView{
		Flex:        layout,
		itemInput:   core.NewTextArea("Item", appCtx.Theme),
		message:     tview.NewTextView().SetLabel("Status ").SetWordWrap(true),
		saveButton:  core.NewButton("Save", appCtx.Theme),
		closeButton: core.NewButton("Close", appCtx.Theme),
		source:      func() (map[string]any, bool) { return nil, false },
		onSave:      func(map[string]any, map[string]any) {},
	}

	layout.
		AddItem(view.itemInput, 0, 1, true).
		AddItem(view.message, 2, 0, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.saveButton, 0, 1, true).
				AddItem(view.closeButton, 0, 1, true),
			1, 0, true,
		)

	view.itemInput.ErrorMessageCallback = func(text string, a ...any) {
		view.SetStatusMessage(fmt.Sprintf(text, a...))
	}

	view.orderedViews = []core.View{view.itemInput, view.saveButton, view.closeButton}
	view.tabNavigator = core.NewViewNavigation1D(layout, view.orderedViews, appCtx.App)
	// Up and Down move the cursor in the editor, so Tab changes the focus
	view.tabNavigator.SetNavigationKeys(tcell.KeyTab, tcell.KeyBacktab)

	view.saveButton.SetSelectedFunc(view.save)

	return view
}

func (inst *DynamoDBUpdateItemView) SetSourceFunc(source func() (map[string]any, bool)) {
	inst.source = source
}

// SetOnSaveFunc sets what happens when Save is pressed with a valid item. It
// is given the item as it was when the view was shown and as it was edited.
func (inst *DynamoDBUpdateItemView) SetOnSaveFunc(handler func(original map[string]any, edited map[string]any)) {
	inst.onSave = handler
}

func (inst *DynamoDBUpdateItemView) SetOnCloseFunc(handler func()) {
	inst.closeButton.SetSelectedFunc(func() {
		inst.message.SetText("")
		handler()
	})
}

func (inst *DynamoDBUpdateItemView) SetStatusMessage(msg string) {
	inst.message.SetText(msg)
}

func (inst *DynamoDBUpdateItemView) save() {
	if inst.original == nil {
		inst.SetStatusMessage("No item selected")
		return
	}

	var edited, err = decodeItemJson(inst.itemInput.GetText())
	if err != nil {
		inst.SetStatusMessage(err.Error())
		return
	}

	inst.SetStatusMessage("Saving...")
	inst.onSave(inst.original, edited)
}

// OnShow fills the editor with the selected item. The item is kept as it was
// then so a changing table can't change what is compared against.
func (inst *DynamoDBUpdateItemView) OnShow() {
	inst.tabNavigator.UpdateOrderedViews(inst.orderedViews, 0)
	inst.message.SetText("")

	var item, ok = inst.source()
	if !ok {
		inst.original = nil
		inst.itemInput.SetText("", false)
		inst.SetStatusMessage("No item selected")
		return
	}

	var formatted, err = json.MarshalIndent(item, "", "  ")
	if err != nil {
		inst.original = nil
		inst.itemInput.SetText("", false)
		inst.SetStatusMessage(err.Error())
		return
	}

	inst.original = item
	inst.itemInput.SetText(string(formatted), false)
}

func (inst *DynamoDBUpdateItemView) GetLastFocusedView() tview.Primitive {
	return inst.tabNavigator.GetLastFocusedView()
}

type FloatingDDBUpdateItemView struct {
	*tview.Flex
	Input *DynamoDBUpdateItemView
}

func NewFloatingDDBUpdateItemView(appCtx *core.AppContext) *FloatingDDBUpdateItemView {
	var input = NewDynamoDBUpdateItemView(appCtx)

	return &FloatingDDBUpdateItemView{
		Flex:  core.FloatingViewRelative("Update Item", input, 80, 80),
		Input: input,
	}
}

func (inst *FloatingDDBUpdateItemView) OnShow() {
	inst.Input.OnShow()
}

func (inst *FloatingDDBUpdateItemView) GetLastFocusedView() tview.Primitive {
	return inst.Input.GetLastFocusedView()
}

// canonicalJson gives a value the way it is shown in the editor, so binary
// values and sets of the item compare equal to what is read back from it.
func canonicalJson(value any) (any, error) {
	var data, err = json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return decodeJsonValue(data)
}

// sameJsonValue compares values read from JSON. Numbers are compared by value,
// so 1 and 1.0 are the same.
func sameJsonValue(a any, b any) bool {
	switch av := a.(type) {
	case json.Number:
		var bv, ok = b.(json.Number)
		if !ok {
			return false
		}
		if av == bv {
			return true
		}
		var af, aErr = strconv.ParseFloat(string(av), 64)
		var bf, bErr = strconv.ParseFloat(string(bv), 64)
		return aErr == nil && bErr == nil && af == bf
	case map[string]any:
		var bv, ok = b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if other, found := bv[k]; !found || !sameJsonValue(v, other) {
				return false
			}
		}
		return true
	case []any:
		var bv, ok = b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !sameJsonValue(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

// diffItem finds what an edit changed: the attributes to write and the ones to
// remove. Attributes that were not touched are left out, so they keep their
// exact types and values. The key can't change, as that would be another item.
func diffItem(
	original map[string]any, edited map[string]any, keyNames []string,
) (set map[string]any, remove []string, err error) {
	var canonical = make(map[string]any, len(original))
	for name, value := range original {
		if canonical[name], err = canonicalJson(value); err != nil {
			return nil, nil, fmt.Errorf("attribute %q: %w", name, err)
		}
	}

	for _, keyName := range keyNames {
		var editedValue, ok = edited[keyName]
		if !ok {
			return nil, nil, fmt.Errorf("key attribute %q can't be removed", keyName)
		}
		if !sameJsonValue(canonical[keyName], editedValue) {
			return nil, nil, fmt.Errorf("key attribute %q can't be changed, add a new item instead", keyName)
		}
	}

	set = map[string]any{}
	for name, value := range edited {
		if previous, existed := canonical[name]; !existed || !sameJsonValue(previous, value) {
			set[name] = value
		}
	}
	for name := range canonical {
		if _, kept := edited[name]; !kept {
			remove = append(remove, name)
		}
	}
	return set, remove, nil
}
