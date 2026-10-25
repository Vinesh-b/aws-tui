package dynamodb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

type DynamoDBAddItemPageView struct {
	*core.ServicePageView
	selectedTable string
	addButton     *tview.Button
	copyButton    *tview.Button
	formatButton  *tview.Button
	itemInput     *core.TextArea
	resultOutput  *core.SearchableTextView
	serviceCtx    *core.ServiceContext[awsapi.DynamoDBApi]
	// onItemAdded is called on the UI goroutine after an item was written.
	onItemAdded func(tableName string)
	// selectedItem returns the item selected in the items table, if any.
	selectedItem func() (map[string]any, bool)
}

func NewDynamoDBAddItemPageView(
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBAddItemPageView {
	var itemInput = core.NewTextArea("Item", serviceCtx.Theme)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var addButton = tview.NewButton("Add Item")
	var copyButton = tview.NewButton("Copy Selected Item")
	var formatButton = tview.NewButton("Format Item")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(addButton, 0, 1, false).
		AddItem(copyButton, 0, 1, false).
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
			{addButton, copyButton, formatButton},
		},
	)

	var view = &DynamoDBAddItemPageView{
		ServicePageView: serviceView,
		addButton:       addButton,
		copyButton:      copyButton,
		formatButton:    formatButton,
		itemInput:       itemInput,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
		onItemAdded:     func(string) {},
		selectedItem:    func() (map[string]any, bool) { return nil, false },
	}

	addButton.SetSelectedFunc(func() { view.AddItem() })
	copyButton.SetSelectedFunc(func() { view.CopySelectedItem() })
	formatButton.SetSelectedFunc(func() { view.FormatItem() })

	return view
}

func (inst *DynamoDBAddItemPageView) SetSelectedTable(tableName string) {
	inst.selectedTable = tableName
	inst.itemInput.SetTitleExtra(tableName)
}

func (inst *DynamoDBAddItemPageView) SetOnItemAdded(handler func(tableName string)) {
	inst.onItemAdded = handler
}

// SetSelectedItemFunc sets how the item selected in the items table is found,
// for use as a template.
func (inst *DynamoDBAddItemPageView) SetSelectedItemFunc(handler func() (map[string]any, bool)) {
	inst.selectedItem = handler
}

// CopySelectedItem replaces the contents of the editor with the selected item
// so it can be edited into a new one.
func (inst *DynamoDBAddItemPageView) CopySelectedItem() {
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
	inst.itemInput.SetText(string(formatted), false)
}

// decodeItemJson reads a JSON object, keeping numbers as written so large
// values aren't rounded by a float conversion.
func decodeItemJson(text string) (map[string]any, error) {
	var decoder = json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.UseNumber()

	var item map[string]any
	if err := decoder.Decode(&item); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("unexpected data after the item")
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("item must be a JSON object with at least one attribute")
	}
	return item, nil
}

func (inst *DynamoDBAddItemPageView) FormatItem() {
	var item, err = decodeItemJson(inst.itemInput.GetText())
	if err != nil {
		inst.itemInput.ErrorMessageCallback("%s", err.Error())
		return
	}

	var formatted, _ = json.MarshalIndent(item, "", "  ")
	inst.itemInput.SetText(string(formatted), false)
}

func (inst *DynamoDBAddItemPageView) AddItem() {
	var tableName = inst.selectedTable
	if tableName == "" {
		inst.itemInput.ErrorMessageCallback("No table selected")
		return
	}

	var item, err = decodeItemJson(inst.itemInput.GetText())
	if err != nil {
		inst.itemInput.ErrorMessageCallback("%s", err.Error())
		return
	}

	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		apiErr = inst.serviceCtx.Api.PutItem(inst.serviceCtx.Ctx, tableName, item)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		inst.resultOutput.SetText(
			fmt.Sprintf("Item added to %s at %s", tableName, time.Now().Format(time.TimeOnly)),
			false,
		)
		inst.onItemAdded(tableName)
	})
}
