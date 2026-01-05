package servicetables

import (
	"fmt"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"aws-tui/internal/pkg/utils"

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

	var table = &DynamoDBGenericTable{
		SelectableTable:      selectableTable,
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
			table.ErrorMessageCallback(err.Error())
			return
		}
		table.ExecuteSearch(DDBTableQuery, expr, true)
	})

	scanView.Input.ScanDoneButton.SetSelectedFunc(func() {
		var expr, err = scanView.Input.GenerateScanExpression()
		if err != nil {
			table.serviceCtx.Logger.Println(err.Error())
			table.ErrorMessageCallback(err.Error())
			return
		}
		table.ExecuteSearch(DDBTableScan, expr, true)
	})

	table.HelpView.View.
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
			inst.tableDescription, err = inst.serviceCtx.Api.DescribeTable(inst.selectedTable)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
				return
			}
		}

		var data []map[string]any

		switch operation {
		case DDBTableScan:
			data, err = inst.serviceCtx.Api.ScanTable(inst.selectedTable, expr, "", reset)
		case DDBTableQuery:
			data, err = inst.serviceCtx.Api.QueryTable(inst.selectedTable, expr, "", reset)
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}

		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.table.Box, func() {
		inst.populateDynamoDBTable(!reset)
	})
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
