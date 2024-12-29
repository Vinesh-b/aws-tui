package serviceviews

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
	DDB_TABLE_SCAN DDBTableOp = iota
	DDB_TABLE_QUERY
)

type DynamoDBGenericTable struct {
	*DynamoDBTableSearchView
	Table            *tview.Table
	data             []map[string]interface{}
	tableDescription *types.TableDescription
	selectedTable    string
	pkQueryString    string
	skQueryString    string
	searchIndexName  string
	logger           *log.Logger
	app              *tview.Application
	api              *awsapi.DynamoDBApi
	attributeIdxMap  map[string]int

	queryExpr expression.Expression
	pkName    string
	skName    string
}

func NewDynamoDBGenericTable(
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBGenericTable {
	var t = tview.NewTable()

	var table = &DynamoDBGenericTable{
		DynamoDBTableSearchView: NewDynamoDBTableSearchView(t, app, logger),
		Table:                   t,
		data:                    nil,
		selectedTable:           "",
		pkQueryString:           "",
		skQueryString:           "",
		searchIndexName:         "",
		logger:                  logger,
		app:                     app,
		api:                     api,
	}

	table.populateDynamoDBTable(false)

	return table
}

func (inst *DynamoDBGenericTable) populateDynamoDBTable(extend bool) {
	inst.Table.
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	if inst.tableDescription == nil {
		return
	}

	var rowIdxOffset = 0
	if extend {
		rowIdxOffset = inst.Table.GetRowCount() - 1
	} else {
		inst.Table.Clear()
		inst.attributeIdxMap = make(map[string]int)
	}

	var headingIdx = 0
	for _, atter := range inst.tableDescription.KeySchema {
		switch atter.KeyType {
		case types.KeyTypeHash:
			inst.pkName = *atter.AttributeName
			inst.attributeIdxMap[*atter.AttributeName] = 0
			headingIdx++
		case types.KeyTypeRange:
			inst.skName = *atter.AttributeName
			inst.attributeIdxMap[*atter.AttributeName] = 1
			headingIdx++
		}
	}

	inst.Table.SetFixed(1, headingIdx)

	var tableTitle = fmt.Sprintf("%s (%d)",
		aws.ToString(inst.tableDescription.TableName),
		len(inst.data)+rowIdxOffset,
	)
	inst.Table.SetTitle(tableTitle)

	for rowIdx, rowData := range inst.data {
		for heading := range rowData {
			var colIdx, ok = inst.attributeIdxMap[heading]
			if !ok {
				inst.attributeIdxMap[heading] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellData = fmt.Sprintf("%v", rowData[heading])
			var previewText = core.ClampStringLen(&cellData, 100)
			var newCell = tview.NewTableCell(previewText).
				SetAlign(tview.AlignLeft)

			// Store the ref to the full row data in the first cell. It will
			// always exist as a PK is required for all tables
			if colIdx == 0 {
				newCell.SetReference(rowData)
			}

			inst.Table.SetCell(rowIdx+rowIdxOffset+1, colIdx, newCell)
		}
	}

	for heading, colIdx := range inst.attributeIdxMap {
		inst.Table.SetCell(0, colIdx, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(core.SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(core.ContrastBackgroundColor),
		)
	}

	if len(inst.data) > 0 {
		inst.Table.SetSelectable(true, false).SetSelectedStyle(
			tcell.Style{}.Background(core.MoreContrastBackgroundColor),
		)
	}
	inst.Table.Select(1, 0)
}

func (inst *DynamoDBGenericTable) RefreshScan(expr expression.Expression, reset bool) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(inst.selectedTable) <= 0 {
			inst.data = make([]map[string]interface{}, 0)
			return
		}
		if reset || inst.tableDescription == nil {
			inst.tableDescription = inst.api.DescribeTable(inst.selectedTable)
		}
		inst.data = inst.api.ScanTable(inst.selectedTable, expr, "", reset)

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateDynamoDBTable(!reset)
	})
}

func (inst *DynamoDBGenericTable) RefreshQuery(expr expression.Expression, reset bool) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(inst.selectedTable) <= 0 {
			inst.data = make([]map[string]interface{}, 0)
			resultChannel <- struct{}{}
			return
		}

		if reset || inst.tableDescription == nil {
			inst.tableDescription = inst.api.DescribeTable(inst.selectedTable)
		}

		inst.data = inst.api.QueryTable(
			inst.selectedTable,
			expr,
			"",
			reset,
		)

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateDynamoDBTable(!reset)
	})
}

func (inst *DynamoDBGenericTable) SetSelectedTable(tableName string) {
	inst.DynamoDBQueryInputView.SetSelectedTable(tableName)
	inst.DynamoDBScanInputView.SetSelectedTable(tableName)
	inst.selectedTable = tableName
}
