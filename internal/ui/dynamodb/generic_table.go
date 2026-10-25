package dynamodb

import (
	"encoding/json"
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DDBTableOp int

const (
	DDBTableScan DDBTableOp = iota
	DDBTableQuery
)

const (
	QUERY_PAGE_NAME = "QUERY"
	SCAN_PAGE_NAME  = "SCAN"
)

type DynamoDBGenericTable struct {
	*core.SelectableTable[map[string]any]
	rootView             core.View
	DeleteView           *core.FloatingConfirmView
	toDelete             itemRef
	table                *tview.Table
	ErrorMessageCallback func(text string, a ...any)
	data                 []map[string]any
	tableDescription     *types.TableDescription
	scanInputView        *FloatingDDBScanInputView
	queryInputView       *FloatingDDBQueryInputView
	selectedTable        string
	pkQueryString        string
	skQueryString        string
	searchIndexName      string
	serviceCtx           *core.ServiceContext[awsapi.DynamoDBApi]
	attributeIdxMap      map[string]int
	queryExpr            expression.Expression
	pkName               string
	skName               string
	lastTableOp          DDBTableOp
	lastSearchExpr       expression.Expression
	lastSelectedRowIdx   int
}

func NewDynamoDBGenericTable(
	serviceContext *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBGenericTable {
	var selectableTable = core.NewSelectableTable[map[string]any]("Results", nil, serviceContext.AppContext)

	var queryView = NewFloatingDDBQueryInputView(serviceContext.AppContext)
	var scanView = NewFloatingDDBScanInputView(serviceContext.AppContext)

	selectableTable.AddRuneToggleOverlay(QUERY_PAGE_NAME, queryView, core.APP_KEY_BINDINGS.TableQuery, false)
	selectableTable.AddRuneToggleOverlay(SCAN_PAGE_NAME, scanView, core.APP_KEY_BINDINGS.TableScan, false)

	var deleteView = core.NewFloatingConfirmView(serviceContext.AppContext, "Delete Item", "Delete")
	selectableTable.AddRuneToggleOverlay(deleteItemOverlayId, deleteView, deleteItemKey, false)

	var table = &DynamoDBGenericTable{
		SelectableTable:      selectableTable,
		DeleteView:           deleteView,
		rootView:             selectableTable.Box,
		table:                selectableTable.GetTable(),
		ErrorMessageCallback: func(text string, a ...any) {},
		data:                 []map[string]any{},
		attributeIdxMap:      map[string]int{},
		scanInputView:        scanView,
		queryInputView:       queryView,
		selectedTable:        "",
		pkQueryString:        "",
		skQueryString:        "",
		searchIndexName:      "",
		lastTableOp:          DDBTableScan,
		lastSelectedRowIdx:   0,
		serviceCtx:           serviceContext,
	}

	table.HighlightSearch = true
	table.populateDynamoDBTable(false)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.ExecuteSearch(table.lastTableOp, table.lastSearchExpr, true)
			return nil
		case core.APP_KEY_BINDINGS.LoadMoreData:
			table.ExecuteSearch(table.lastTableOp, table.lastSearchExpr, false)
			return nil
		}
		return event
	})

	queryView.Input.QueryDoneButton.SetSelectedFunc(func() {
		queryView.Input.SetPartitionKeyName(table.pkName)
		queryView.Input.SetSortKeyName(table.skName)

		var expr, err = queryView.Input.GenerateQueryExpression()
		if err != nil {
			table.serviceCtx.Logger.Println(err.Error())
			table.ErrorMessageCallback("%s", err.Error())
			return
		}
		table.ExecuteSearch(DDBTableQuery, expr, true)
	})

	scanView.Input.ScanDoneButton.SetSelectedFunc(func() {
		var expr, err = scanView.Input.GenerateScanExpression()
		if err != nil {
			table.serviceCtx.Logger.Println(err.Error())
			table.ErrorMessageCallback("%s", err.Error())
			return
		}
		table.ExecuteSearch(DDBTableScan, expr, true)
	})

	deleteView.Input.SetOnShowFunc(table.prepareDeleteItem)
	deleteView.Input.SetOnConfirmFunc(func() {
		table.ToggleOverlay(deleteItemOverlayId, true)
		table.deleteItem(table.toDelete)
	})
	deleteView.Input.SetOnCancelFunc(func() {
		table.ToggleOverlay(deleteItemOverlayId, true)
	})

	table.HelpView.View.
		AddItem(string(deleteItemKey), "Delete the selected item", nil).
		AddItem("f", "Jump to next search result", nil).
		AddItem("F", "Jump to previous search result", nil).
		AddItem("q", "To show query view", nil).
		AddItem("s", "To show scan view", nil)

	return table
}

func (inst *DynamoDBGenericTable) populateDynamoDBTable(extend bool) {
	if inst.tableDescription == nil {
		return
	}

	inst.table.Clear()

	if !extend {
		inst.attributeIdxMap = make(map[string]int)
		inst.lastSelectedRowIdx = 1
	}

	var fixedCols = 0
	for _, atter := range inst.tableDescription.KeySchema {
		switch atter.KeyType {
		case types.KeyTypeHash:
			inst.pkName = *atter.AttributeName
			inst.attributeIdxMap[*atter.AttributeName] = 0
			fixedCols++
		case types.KeyTypeRange:
			inst.skName = *atter.AttributeName
			inst.attributeIdxMap[*atter.AttributeName] = 1
			fixedCols++
		}
	}
	inst.table.SetFixed(1, fixedCols)

	for _, rowData := range inst.data {
		for heading := range rowData {
			var headingIdx = len(inst.attributeIdxMap)
			var _, ok = inst.attributeIdxMap[heading]
			if !ok {
				inst.attributeIdxMap[heading] = headingIdx
			}
		}
	}

	for rowIdx, rowData := range inst.data {
		for heading, colIdx := range inst.attributeIdxMap {
			var text = fmt.Sprintf("%v", rowData[heading])
			var cell *tview.TableCell

			if colIdx == 0 {
				// Store the ref to the full row data in the first cell. It will
				// always exist as a PK is required for all tables
				cell = core.NewTableCell(text, &rowData)
			} else {
				cell = core.NewTableCell(text, &map[string]any{heading: rowData[heading]})
			}

			inst.table.SetCell(rowIdx+1, colIdx, cell)
		}
	}

	for heading, colIdx := range inst.attributeIdxMap {
		core.SetTableHeading(inst.table, inst.serviceCtx.Theme, heading, colIdx)
	}

	inst.table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(inst.serviceCtx.Theme.MoreContrastBackgroundColor),
	)

	var clampedName = utils.ClampStringLen(inst.tableDescription.TableName, 100)
	inst.SetTitleExtra(clampedName)
	inst.RefreshTitle(len(inst.data))

	inst.table.Select(inst.lastSelectedRowIdx, 0)
}

func (inst *DynamoDBGenericTable) ExecuteSearch(operation DDBTableOp, expr expression.Expression, reset bool) {
	inst.lastTableOp = operation
	inst.lastSearchExpr = expr
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		if len(inst.selectedTable) <= 0 {
			inst.data = make([]map[string]any, 0)
			return
		}

		var err error = nil
		if reset || inst.tableDescription == nil {
			inst.tableDescription, err = inst.serviceCtx.Api.DescribeTable(inst.serviceCtx.Ctx, inst.selectedTable)
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}
		}

		var data []map[string]any

		switch operation {
		case DDBTableScan:
			data, err = inst.serviceCtx.Api.ScanTable(inst.serviceCtx.Ctx, inst.selectedTable, expr, "", reset)
		case DDBTableQuery:
			data, err = inst.serviceCtx.Api.QueryTable(inst.serviceCtx.Ctx, inst.selectedTable, expr, "", reset)
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}

		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.table.Box, func() {
		inst.populateDynamoDBTable(!reset)
	})
}

// GetSelectedItem returns all the attributes of the item in the selected row.
func (inst *DynamoDBGenericTable) GetSelectedItem() (map[string]any, bool) {
	var row, _ = inst.table.GetSelection()
	if row < 1 || row >= inst.table.GetRowCount() {
		return nil, false
	}

	// The first column is the partition key, which holds the full item
	var item = inst.GetPrivateData(row, 0)
	return item, len(item) > 0
}

func (inst *DynamoDBGenericTable) SetSelectedTable(tableName string) {
	inst.queryInputView.Input.SetSelectedTable(tableName)
	inst.scanInputView.Input.SetSelectedTable(tableName)
	inst.selectedTable = tableName
}

func (inst *DynamoDBGenericTable) SetSelectionChangedFunc(
	handler func(row int, column int),
) *DynamoDBGenericTable {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.lastSelectedRowIdx = row
		handler(row, column)
	})
	return inst
}

// deleteItemKey opens the delete confirmation for the selected item.
const deleteItemKey = 'x'

const deleteItemOverlayId = "DELETE_ITEM"

// itemRef is an item to delete, by its key
type itemRef struct {
	tableName string
	key       map[string]any
}

// keyNames are the names of the key attributes of the table
func (inst *DynamoDBGenericTable) keyNames() []string {
	var names []string
	for _, name := range []string{inst.pkName, inst.skName} {
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// prepareDeleteItem notes the key of the selected item and asks about it. It is
// kept so the table changing underneath can't change what is deleted.
func (inst *DynamoDBGenericTable) prepareDeleteItem() {
	var view = inst.DeleteView.Input
	inst.toDelete = itemRef{}

	var item, ok = inst.GetSelectedItem()
	if !ok {
		view.SetMessage("No item selected")
		return
	}

	var key = map[string]any{}
	for _, name := range inst.keyNames() {
		var value, found = item[name]
		if !found {
			view.SetMessage(fmt.Sprintf("The key attribute %q isn't in the selected item", name))
			return
		}
		key[name] = value
	}
	if len(key) == 0 {
		view.SetMessage("The key of the table is unknown")
		return
	}

	inst.toDelete = itemRef{tableName: inst.selectedTable, key: key}

	var keyText, _ = json.Marshal(key)
	view.SetMessage(fmt.Sprintf(
		"Delete the item %s from %s?\n\nThis can't be undone.", keyText, inst.selectedTable,
	))
}

// deleteItem deletes the item without blocking the UI, then reloads the items.
// Failures are shown as errors.
func (inst *DynamoDBGenericTable) deleteItem(item itemRef) {
	if item.tableName == "" {
		inst.ErrorMessageCallback("No item selected")
		return
	}

	go func() {
		var err = inst.serviceCtx.Api.DeleteItem(inst.serviceCtx.Ctx, item.tableName, item.key)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}

			if inst.selectedTable == item.tableName {
				inst.ExecuteSearch(inst.lastTableOp, inst.lastSearchExpr, true)
			}
		})
	}()
}
