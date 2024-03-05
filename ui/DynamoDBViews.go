package ui

import (
	"fmt"
	"log"
	"time"

	"aws-tui/dynamodb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func populateDynamoDBTabelsTable(table *tview.Table, data []string) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{row})
	}

	initSelectableTable(table, "DynamoDB Tables",
		tableRow{"Name"},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(1, 0)
}

func populateDynamoDBTabelDetailsTable(table *tview.Table, data *types.TableDescription) {
	var tableData []tableRow
	var partitionKey = ""
	var sortKey = ""

	if data != nil {
		for _, atter := range data.KeySchema {
			switch atter.KeyType {
			case types.KeyTypeHash:
				partitionKey = *atter.AttributeName
			case types.KeyTypeRange:
				sortKey = *atter.AttributeName
			}
		}

		tableData = []tableRow{
			{"Name", aws.ToString(data.TableName)},
			{"Status", fmt.Sprintf("%s", data.TableStatus)},
			{"CreationDate", data.CreationDateTime.Format(time.DateTime)},
			{"PartitionKey", partitionKey},
			{"SortKey", sortKey},
			{"ItemCount", fmt.Sprintf("%d", aws.ToInt64(data.ItemCount))},
			{"GSIs", fmt.Sprintf("%v", data.GlobalSecondaryIndexes)},
		}
	}

	initBasicTable(table, "Table Details", tableData, false)
	table.Select(0, 0)
}

type dynamoDBGenericTable struct {
	Table           *tview.Table
	attributeIdxMap map[string]int
}

func (inst *dynamoDBGenericTable) populateDynamoDBTable(
	description *types.TableDescription,
	data []map[string]interface{},
	extend bool,
) {
	inst.Table.
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	if description == nil || len(data) == 0 {
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
	for _, atter := range description.KeySchema {
		switch atter.KeyType {
		case types.KeyTypeHash:
			inst.attributeIdxMap[*atter.AttributeName] = 0
			headingIdx++
		case types.KeyTypeRange:
			inst.attributeIdxMap[*atter.AttributeName] = 1
			headingIdx++
		}
	}

	inst.Table.SetFixed(1, headingIdx)

	var tableTitle = fmt.Sprintf("%s (%d)",
		aws.ToString(description.TableName),
		len(data)+rowIdxOffset,
	)
	inst.Table.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for heading := range rowData {
			var colIdx, ok = inst.attributeIdxMap[heading]
			if !ok {
				inst.attributeIdxMap[heading] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellData = fmt.Sprintf("%v", rowData[heading])
			var previewText = clampStringLen(&cellData, 100)
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
			SetTextColor(secondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(contrastBackgroundColor),
		)
	}

	if len(data) > 0 {
		inst.Table.SetSelectable(true, false).SetSelectedStyle(
			tcell.Style{}.Background(moreContrastBackgroundColor),
		)
	}
	inst.Table.Select(1, 0)
}

type DynamoDBDetailsView struct {
	TablesTable  *tview.Table
	DetailsTable *tview.Table
	SearchInput  *tview.InputField
	RootView     *tview.Flex
	app          *tview.Application
	api          *dynamodb.DynamoDBApi
}

func NewDynamoDBDetailsView(
	app *tview.Application,
	api *dynamodb.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBDetailsView {
	var tablesTable = tview.NewTable()
	populateDynamoDBTabelsTable(tablesTable, make([]string, 0))

	var detailsTable = tview.NewTable()
	populateDynamoDBTabelDetailsTable(detailsTable, nil)

	var inputField = createSearchInput("Tables")

	const detailsSize = 3000
	const tablesSize = 5000

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(detailsTable, 0, detailsSize, false).
		AddItem(tablesTable, 0, tablesSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		detailsTable, tablesTable,
		detailsSize, tablesSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			tablesTable,
			detailsTable,
		},
	)

	return &DynamoDBDetailsView{
		TablesTable:  tablesTable,
		DetailsTable: detailsTable,
		SearchInput:  inputField,
		RootView:     serviceView.RootView,
		app:          app,
		api:          api,
	}
}

func (inst *DynamoDBDetailsView) RefreshTables(search string, force bool) {
	var data []string
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			data = inst.api.FilterByName(search)
		} else {
			data = inst.api.ListTables(force)
		}
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.TablesTable.Box, resultChannel, func() {
		populateDynamoDBTabelsTable(inst.TablesTable, data)
	})
}

func (inst *DynamoDBDetailsView) RefreshDetails(tableName string) {
	var data *types.TableDescription = nil
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.DescribeTable(tableName)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.DetailsTable.Box, resultChannel, func() {
		populateDynamoDBTabelDetailsTable(inst.DetailsTable, data)
	})
}

func (inst *DynamoDBDetailsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshTables(inst.SearchInput.GetText(), true)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})

	inst.TablesTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshTables("", true)
		}

		return event
	})
}

type ddbTableOp int

const (
	DDB_TABLE_SCAN ddbTableOp = iota
	DDB_TABLE_QUERY
)

type DynamoDBTableItemsView struct {
	ItemsTable       *dynamoDBGenericTable
	SearchInput      *tview.InputField
	RootView         *tview.Flex
	app              *tview.Application
	api              *dynamodb.DynamoDBApi
	tableName        string
	tableDescription *types.TableDescription
	searchPositions  []int
	queryPkInput     *tview.InputField
	querySkInput     *tview.InputField
	runQueryBtn      *tview.Button
	lastTableOp      ddbTableOp
}

func NewDynamoDBTableItemsView(
	app *tview.Application,
	api *dynamodb.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTableItemsView {
	var itemsTable = tview.NewTable()
	var genericTable = dynamoDBGenericTable{Table: itemsTable}
	genericTable.populateDynamoDBTable(nil, nil, false)

	var inputField = createSearchInput("Item")

	var expandItemView = createExpandedLogView(app, itemsTable, 0, DATA_TYPE_MAP_STRING_ANY)

	var pkQueryValInput = tview.NewInputField().
		SetFieldWidth(0).
		SetLabel(" Partition Key ")
	var skQueryValInput = tview.NewInputField().
		SetFieldWidth(0).
		SetLabel(" Sort Key      ")
	var runQueryBtn = tview.NewButton("Run")

	var queryView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pkQueryValInput, 1, 0, false).
		AddItem(skQueryValInput, 1, 0, false).
		AddItem(runQueryBtn, 1, 0, false)
	queryView.
		SetBorder(true).
		SetTitle("Query").
		SetTitleAlign(tview.AlignLeft)

	const expandItemViewSize = 3
	const itemsTableSize = 7

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(expandItemView, 0, expandItemViewSize, false).
		AddItem(itemsTable, 0, itemsTableSize, false).
		AddItem(queryView, 5, 0, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		expandItemView, itemsTable,
		expandItemViewSize, itemsTableSize,
	)

	serviceView.InitViewTabNavigation(
		queryView,
		[]view{
			pkQueryValInput,
			skQueryValInput,
			runQueryBtn,
		},
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			queryView,
			itemsTable,
			expandItemView,
		},
	)

	return &DynamoDBTableItemsView{
		ItemsTable:       &genericTable,
		SearchInput:      inputField,
		RootView:         serviceView.RootView,
		app:              app,
		api:              api,
		tableDescription: nil,
		queryPkInput:     pkQueryValInput,
		querySkInput:     skQueryValInput,
		runQueryBtn:      runQueryBtn,
	}
}

func (inst *DynamoDBTableItemsView) RefreshItems(tableName string, force bool) {
	inst.tableName = tableName

	var data []map[string]interface{}
	var descData *types.TableDescription = nil
	var resultChannel = make(chan struct{})

	go func() {
		if len(tableName) <= 0 {
			data = make([]map[string]interface{}, 0)
			return
		}
		if force || inst.tableDescription == nil {
			inst.tableDescription = inst.api.DescribeTable(inst.tableName)
		}
		descData = inst.tableDescription
		data = inst.api.ScanTable(inst.tableDescription, force)
		inst.lastTableOp = DDB_TABLE_SCAN

		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.ItemsTable.Table.Box, resultChannel, func() {
		inst.ItemsTable.populateDynamoDBTable(descData, data, !force)
	})
}

func (inst *DynamoDBTableItemsView) RefreshItemsForQuery(tableName string, force bool) {
	var data []map[string]interface{}
	var descData *types.TableDescription = nil
	var resultChannel = make(chan struct{})

	go func() {
		if len(tableName) <= 0 {
			data = make([]map[string]interface{}, 0)
			return
		}

		if force || inst.tableDescription == nil {
			inst.tableDescription = inst.api.DescribeTable(inst.tableName)
		}

		descData = inst.tableDescription
		data = inst.api.QueryTable(
			inst.tableDescription,
			inst.queryPkInput.GetText(),
			inst.querySkInput.GetText(),
			force,
		)
		inst.lastTableOp = DDB_TABLE_QUERY

		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.ItemsTable.Table.Box, resultChannel, func() {
		inst.ItemsTable.populateDynamoDBTable(descData, data, !force)
	})
}

func (inst *DynamoDBTableItemsView) InitInputCapture() *DynamoDBTableItemsView {
	inst.SearchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Broken as the table stores a ref to a map and not a string
			// which is what search currently supports
			break
			inst.searchPositions = highlightTableSearch(
				inst.app,
				inst.ItemsTable.Table,
				inst.SearchInput.GetText(),
				[]int{0},
			)
			inst.app.SetFocus(inst.ItemsTable.Table)
		case tcell.KeyCtrlR:
			inst.SearchInput.SetText("")
			clearSearchHighlights(inst.ItemsTable.Table)
			inst.searchPositions = nil
		}
		return event
	})

	inst.runQueryBtn.SetSelectedFunc(func() {
		inst.RefreshItemsForQuery(inst.tableName, true)
	})

	var nextSearch = 0
	inst.ItemsTable.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			const forceRefresh = true
			switch inst.lastTableOp {
			case DDB_TABLE_QUERY:
				inst.RefreshItemsForQuery(inst.tableName, forceRefresh)
			case DDB_TABLE_SCAN:
				inst.RefreshItems(inst.tableName, forceRefresh)
			}
		case tcell.KeyCtrlN:
			const forceRefresh = false
			switch inst.lastTableOp {
			case DDB_TABLE_QUERY:
				inst.RefreshItemsForQuery(inst.tableName, forceRefresh)
			case DDB_TABLE_SCAN:
				inst.RefreshItems(inst.tableName, forceRefresh)
			}
		}

		var searchCount = len(inst.searchPositions)
		if searchCount > 0 {
			switch event.Rune() {
			case rune('n'):
				nextSearch = (nextSearch + 1) % searchCount
				inst.ItemsTable.Table.Select(inst.searchPositions[nextSearch], 0)
			case rune('N'):
				nextSearch = (nextSearch - 1 + searchCount) % searchCount
				inst.ItemsTable.Table.Select(inst.searchPositions[nextSearch], 0)
			}
		}

		return event
	})

	return inst
}

func (inst *DynamoDBTableItemsView) SetTableName(tableName string,
) *DynamoDBTableItemsView {
	inst.tableName = tableName
	return inst
}

func createDynamoDBHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0x003388))
	defer resetGlobalStyle()

	var (
		api = dynamodb.NewDynamoDBApi(config, logger)

		ddbDetailsView = NewDynamoDBDetailsView(app, api, logger)
		ddbItemsView   = NewDynamoDBTableItemsView(app, api, logger)
	)

	var pages = tview.NewPages()
	pages.
		AddPage("Items", ddbItemsView.RootView, true, true).
		AddPage("Tables", ddbDetailsView.RootView, true, true)

	var orderedPages = []string{
		"Tables",
		"Items",
	}

	var serviceRootView = NewServiceRootView(
		app, string(DYNAMODB), pages, orderedPages).Init()

	var selectedTableName = ""
	ddbDetailsView.TablesTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		selectedTableName = ddbDetailsView.TablesTable.GetCell(row, 0).Text
		ddbDetailsView.RefreshDetails(selectedTableName)
	})

	ddbDetailsView.TablesTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		selectedTableName = ddbDetailsView.TablesTable.GetCell(row, 0).Text
		ddbItemsView.RefreshItems(selectedTableName, true)
		serviceRootView.ChangePage(1, ddbItemsView.ItemsTable.Table)
	})

	ddbDetailsView.InitInputCapture()

	ddbItemsView.
		SetTableName(selectedTableName).
		InitInputCapture()

	return serviceRootView.RootView
}
