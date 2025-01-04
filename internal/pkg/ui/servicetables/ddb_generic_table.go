package servicetables

import (
	"fmt"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
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

type DynamoDBGenericTable struct {
	*core.SelectableTable[any]
	*DynamoDBTableSearchView
	rootView             core.View
	table                *tview.Table
	ErrorMessageCallback func(text string, a ...any)
	data                 []map[string]any
	tableDescription     *types.TableDescription
	selectedTable        string
	pkQueryString        string
	skQueryString        string
	searchIndexName      string
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.DynamoDBApi
	attributeIdxMap      map[string]int
	queryExpr            expression.Expression
	pkName               string
	skName               string
	lastTableOp          DDBTableOp
	lastSearchExpr       expression.Expression
	lastSelectedRowIdx   int
}

func NewDynamoDBGenericTable(
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBGenericTable {
	var selectableTable = core.NewSelectableTable[any]("", nil)
	var searchView = NewDynamoDBTableSearchView(selectableTable, app, logger)

	var table = &DynamoDBGenericTable{
		SelectableTable:         selectableTable,
		DynamoDBTableSearchView: searchView,
		rootView:                selectableTable.Box,
		table:                   selectableTable.GetTable(),
		ErrorMessageCallback:    func(text string, a ...any) {},
		data:                    []map[string]any{},
		attributeIdxMap:         map[string]int{},
		selectedTable:           "",
		pkQueryString:           "",
		skQueryString:           "",
		searchIndexName:         "",
		lastTableOp:             DDBTableScan,
		lastSelectedRowIdx:      0,
		logger:                  logger,
		app:                     app,
		api:                     api,
	}

	table.HighlightSearch = true
	table.populateDynamoDBTable(false)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			table.ExecuteSearch(table.lastTableOp, table.lastSearchExpr, true)
		case core.APP_KEY_BINDINGS.NextPage:
			table.ExecuteSearch(table.lastTableOp, table.lastSearchExpr, false)
		}
		return event
	})

	table.QueryDoneButton.SetSelectedFunc(func() {
		table.SetPartitionKeyName(table.pkName)
		table.SetSortKeyName(table.skName)

		var expr, err = table.GenerateQueryExpression()
		if err != nil {
			table.logger.Println(err.Error())
			table.ErrorMessageCallback(err.Error())
			return
		}
		table.ExecuteSearch(DDBTableQuery, expr, true)
	})

	table.ScanDoneButton.SetSelectedFunc(func() {
		var expr, err = table.GenerateScanExpression()
		if err != nil {
			table.logger.Println(err.Error())
			table.ErrorMessageCallback(err.Error())
			return
		}
		table.ExecuteSearch(DDBTableScan, expr, true)
	})

	return table
}

func (inst *DynamoDBGenericTable) populateDynamoDBTable(extend bool) {
	if inst.tableDescription == nil {
		return
	}

	inst.table.Clear()
	var tableTitle = fmt.Sprintf("%s (%d)",
		aws.ToString(inst.tableDescription.TableName),
		len(inst.data),
	)
	inst.rootView.SetTitle(tableTitle)

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
			var cellData = fmt.Sprintf("%v", rowData[heading])
			var previewText = core.ClampStringLen(&cellData, 100)
			var newCell = tview.NewTableCell(previewText).
				SetAlign(tview.AlignLeft)

			// Store the ref to the full row data in the first cell. It will
			// always exist as a PK is required for all tables
			if colIdx == 0 {
				newCell.SetReference(rowData)
			} else {
				newCell.SetReference(rowData[heading])
			}

			inst.table.SetCell(rowIdx+1, colIdx, newCell)
		}
	}

	for heading, colIdx := range inst.attributeIdxMap {
		inst.table.SetCell(0, colIdx, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(core.SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(core.ContrastBackgroundColor),
		)
	}

	if len(inst.data) > 0 {
		inst.table.SetSelectable(true, true).SetSelectedStyle(
			tcell.Style{}.Background(core.MoreContrastBackgroundColor),
		)
	}

	inst.table.Select(inst.lastSelectedRowIdx, 0)
}

func (inst *DynamoDBGenericTable) ExecuteSearch(operation DDBTableOp, expr expression.Expression, reset bool) {
	inst.lastTableOp = operation
	inst.lastSearchExpr = expr
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(inst.selectedTable) <= 0 {
			inst.data = make([]map[string]interface{}, 0)
			return
		}

		var err error = nil
		if reset || inst.tableDescription == nil {
			inst.tableDescription, err = inst.api.DescribeTable(inst.selectedTable)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
				return
			}
		}

		var data []map[string]any

		switch operation {
		case DDBTableScan:
			data, err = inst.api.ScanTable(inst.selectedTable, expr, "", reset)
		case DDBTableQuery:
			data, err = inst.api.QueryTable(inst.selectedTable, expr, "", reset)
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
	inst.DynamoDBQueryInputView.SetSelectedTable(tableName)
	inst.DynamoDBScanInputView.SetSelectedTable(tableName)
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
