package dynamodb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

type DynamoDBUpdateItemPageView struct {
	*core.ServicePageView
	selectedTable string
	updateButton  *tview.Button
	reloadButton  *tview.Button
	formatButton  *tview.Button
	itemInput     *core.TextArea
	resultOutput  *core.SearchableTextView
	serviceCtx    *core.ServiceContext[awsapi.DynamoDBApi]

	// original is the item as it was loaded into the editor, which edits are
	// compared against, and loadedText is how it was shown
	original   map[string]any
	loadedText string

	// selectedItem returns the item selected in the items table, if any.
	selectedItem func() (map[string]any, bool)
	// keyNames returns the names of the key attributes of the table.
	keyNames func() []string
	// onItemUpdated is called on the UI goroutine after an item was written.
	onItemUpdated func(tableName string)
}

func NewDynamoDBUpdateItemPageView(
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBUpdateItemPageView {
	var itemInput = core.NewTextArea("Item", serviceCtx.Theme)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var updateButton = tview.NewButton("Update Item")
	var reloadButton = tview.NewButton("Load Selected Item")
	var formatButton = tview.NewButton("Format Item")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(updateButton, 0, 1, false).
		AddItem(reloadButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	itemInput.SetTitleExtra("NO TABLE SELECTED")

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(itemInput, 0, 1, true).
		AddItem(buttonsView, 3, 0, false)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		SetDirection(tview.FlexColumn).
		AddItem(inputColumn, 0, 1, true).
		AddItem(resultOutput, 0, 1, false)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	resultOutput.ErrorMessageCallback = errorHandler
	itemInput.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{itemInput, resultOutput},
			{updateButton, reloadButton, formatButton},
		},
	)

	var view = &DynamoDBUpdateItemPageView{
		ServicePageView: serviceView,
		updateButton:    updateButton,
		reloadButton:    reloadButton,
		formatButton:    formatButton,
		itemInput:       itemInput,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
		selectedItem:    func() (map[string]any, bool) { return nil, false },
		keyNames:        func() []string { return nil },
		onItemUpdated:   func(string) {},
	}

	updateButton.SetSelectedFunc(func() { view.UpdateItem() })
	reloadButton.SetSelectedFunc(func() { view.LoadSelectedItem() })
	formatButton.SetSelectedFunc(func() { view.FormatItem() })

	return view
}

func (inst *DynamoDBUpdateItemPageView) SetSelectedTable(tableName string) {
	inst.selectedTable = tableName
	inst.itemInput.SetTitleExtra(tableName)
}

// SetSelectedItemFunc sets how the item selected in the items table is found.
func (inst *DynamoDBUpdateItemPageView) SetSelectedItemFunc(handler func() (map[string]any, bool)) {
	inst.selectedItem = handler
}

// SetKeyNamesFunc sets how the names of the key attributes of the table are
// found.
func (inst *DynamoDBUpdateItemPageView) SetKeyNamesFunc(handler func() []string) {
	inst.keyNames = handler
}

func (inst *DynamoDBUpdateItemPageView) SetOnItemUpdated(handler func(tableName string)) {
	inst.onItemUpdated = handler
}

// hasUnsavedEdits tells if the editor was changed since the item was loaded
func (inst *DynamoDBUpdateItemPageView) hasUnsavedEdits() bool {
	return inst.itemInput.GetText() != inst.loadedText
}

// LoadSelectedItem replaces the contents of the editor with the selected item,
// which is kept as it is now to compare the edits against.
func (inst *DynamoDBUpdateItemPageView) LoadSelectedItem() {
	var item, ok = inst.selectedItem()
	if !ok {
		inst.itemInput.ErrorMessageCallback("No item selected")
		return
	}

	var formatted, err = json.MarshalIndent(item, "", "  ")
	if err != nil {
		inst.itemInput.ErrorMessageCallback("%s", err.Error())
		return
	}

	inst.original = item
	inst.loadedText = string(formatted)
	inst.itemInput.SetText(inst.loadedText, false)
}

// LoadSelectedItemIfUnedited loads the selected item, unless that would throw
// away edits. It is used when the tab is shown, and does nothing without a
// selected item.
func (inst *DynamoDBUpdateItemPageView) LoadSelectedItemIfUnedited() {
	if inst.hasUnsavedEdits() {
		return
	}
	if _, ok := inst.selectedItem(); ok {
		inst.LoadSelectedItem()
	}
}

func (inst *DynamoDBUpdateItemPageView) FormatItem() {
	var item, err = decodeItemJson(inst.itemInput.GetText())
	if err != nil {
		inst.itemInput.ErrorMessageCallback("%s", err.Error())
		return
	}

	var formatted, _ = json.MarshalIndent(item, "", "  ")
	inst.itemInput.SetText(string(formatted), false)
}

// UpdateItem writes what was edited to the item, without blocking the UI. Only
// the attributes that changed are written.
func (inst *DynamoDBUpdateItemPageView) UpdateItem() {
	var tableName = inst.selectedTable
	if tableName == "" {
		inst.itemInput.ErrorMessageCallback("No table selected")
		return
	}
	if inst.original == nil {
		inst.itemInput.ErrorMessageCallback("No item loaded, select an item and load it first")
		return
	}

	var edited, err = decodeItemJson(inst.itemInput.GetText())
	if err != nil {
		inst.itemInput.ErrorMessageCallback("%s", err.Error())
		return
	}

	var keyNames = inst.keyNames()
	var set, remove, diffErr = diffItem(inst.original, edited, keyNames)
	if diffErr != nil {
		inst.itemInput.ErrorMessageCallback("%s", diffErr.Error())
		return
	}
	if len(set) == 0 && len(remove) == 0 {
		inst.resultOutput.SetText("No changes to save", false)
		return
	}

	var key = map[string]any{}
	for _, name := range keyNames {
		key[name] = edited[name]
	}

	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		apiErr = inst.serviceCtx.Api.UpdateItem(inst.serviceCtx.Ctx, tableName, key, set, remove)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		// Further edits are compared against what was just saved
		inst.original = edited
		inst.loadedText = inst.itemInput.GetText()

		inst.resultOutput.SetText(
			fmt.Sprintf("Item updated in %s at %s", tableName, time.Now().Format(time.TimeOnly)),
			false,
		)
		inst.onItemUpdated(tableName)
	})
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
