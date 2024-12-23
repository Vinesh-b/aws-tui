package serviceviews

import (
	"fmt"
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func populateDynamoDBTabelsTable(table *tview.Table, data []string) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{row})
	}

	core.InitSelectableTable(table, "DynamoDB Tables",
		core.TableRow{"Name"},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(1, 0)
}

func populateDynamoDBTabelDetailsTable(table *tview.Table, data *types.TableDescription) {
	var tableData []core.TableRow
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

		tableData = []core.TableRow{
			{"Name", aws.ToString(data.TableName)},
			{"Status", fmt.Sprintf("%s", data.TableStatus)},
			{"CreationDate", data.CreationDateTime.Format(time.DateTime)},
			{"PartitionKey", partitionKey},
			{"SortKey", sortKey},
			{"ItemCount", fmt.Sprintf("%d", aws.ToInt64(data.ItemCount))},
			{"GSIs", fmt.Sprintf("%v", data.GlobalSecondaryIndexes)},
		}
	}

	core.InitBasicTable(table, "Table Details", tableData, false)
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

	if len(data) > 0 {
		inst.Table.SetSelectable(true, false).SetSelectedStyle(
			tcell.Style{}.Background(core.MoreContrastBackgroundColor),
		)
	}
	inst.Table.Select(1, 0)
}

type DynamoDBDetailsView struct {
	TablesTable    *tview.Table
	DetailsTable   *tview.Table
	RootView       *tview.Flex
	searchableView *core.SearchableView
	app            *tview.Application
	api            *awsapi.DynamoDBApi
}

func NewDynamoDBDetailsView(
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBDetailsView {
	var tablesTable = tview.NewTable()
	populateDynamoDBTabelsTable(tablesTable, make([]string, 0))

	var detailsTable = tview.NewTable()
	populateDynamoDBTabelDetailsTable(detailsTable, nil)

	const detailsSize = 3000
	const tablesSize = 5000

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(detailsTable, 0, detailsSize, false).
		AddItem(tablesTable, 0, tablesSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		detailsTable, tablesTable,
		detailsSize, tablesSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			tablesTable,
			detailsTable,
		},
	)

	return &DynamoDBDetailsView{
		TablesTable:    tablesTable,
		DetailsTable:   detailsTable,
		RootView:       serviceView.RootView,
		searchableView: serviceView.SearchableView,
		app:            app,
		api:            api,
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

	go core.LoadData(inst.app, inst.TablesTable.Box, resultChannel, func() {
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

	go core.LoadData(inst.app, inst.DetailsTable.Box, resultChannel, func() {
		populateDynamoDBTabelDetailsTable(inst.DetailsTable, data)
	})
}

func (inst *DynamoDBDetailsView) InitInputCapture() {
	inst.searchableView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshTables(inst.searchableView.GetText(), true)
		case tcell.KeyEsc:
			inst.searchableView.SetText("")
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
	RootView         *tview.Flex
	app              *tview.Application
	api              *awsapi.DynamoDBApi
	tableName        string
	tableDescription *types.TableDescription
	searchPositions  []int
	queryPkInput     *tview.InputField
	querySkInput     *tview.InputField
	runQueryBtn      *tview.Button
	lastTableOp      ddbTableOp
	searchableView   *core.SearchableView
}

func NewDynamoDBTableItemsView(
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTableItemsView {
	var itemsTable = tview.NewTable()
	var genericTable = dynamoDBGenericTable{Table: itemsTable}
	genericTable.populateDynamoDBTable(nil, nil, false)

	var expandItemView = core.CreateExpandedLogView(app, itemsTable, 0, core.DATA_TYPE_MAP_STRING_ANY)

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

	var atterNameInput = tview.NewInputField().
		SetFieldWidth(0).
		SetLabel(" Attribute Name  ")
	var atterValueInput = tview.NewInputField().
		SetFieldWidth(0).
		SetLabel(" Attribute Value ")
	var runScanBtn = tview.NewButton("Run")
	var scanView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(atterNameInput, 1, 0, false).
		AddItem(atterValueInput, 1, 0, false).
		AddItem(runScanBtn, 1, 0, false)
	scanView.
		SetBorder(true).
		SetTitle("Scan").
		SetTitleAlign(tview.AlignLeft)

	const expandItemViewSize = 3
	const itemsTableSize = 7

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandItemView, 0, expandItemViewSize, false).
		AddItem(itemsTable, 0, itemsTableSize, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(queryView, 0, 1, false).
			AddItem(scanView, 0, 1, false),
			5, 0, true,
		)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		expandItemView, itemsTable,
		expandItemViewSize, itemsTableSize,
	)

	serviceView.InitViewTabNavigation(
		scanView,
		[]core.View{
			atterNameInput,
			atterValueInput,
			runScanBtn,
		},
	)

	serviceView.InitViewTabNavigation(
		queryView,
		[]core.View{
			pkQueryValInput,
			skQueryValInput,
			runQueryBtn,
		},
	)

	serviceView.InitViewNavigation(
		[]core.View{
			queryView,
			scanView,
			itemsTable,
			expandItemView,
		},
	)

	return &DynamoDBTableItemsView{
		ItemsTable:       &genericTable,
		RootView:         serviceView.RootView,
		app:              app,
		api:              api,
		tableDescription: nil,
		queryPkInput:     pkQueryValInput,
		querySkInput:     skQueryValInput,
		runQueryBtn:      runQueryBtn,
		searchableView:   serviceView.SearchableView,
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

	go core.LoadData(inst.app, inst.ItemsTable.Table.Box, resultChannel, func() {
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

	go core.LoadData(inst.app, inst.ItemsTable.Table.Box, resultChannel, func() {
		inst.ItemsTable.populateDynamoDBTable(descData, data, !force)
	})
}

func (inst *DynamoDBTableItemsView) InitInputCapture() *DynamoDBTableItemsView {
	inst.searchableView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			// Broken as the table stores a ref in col 0
			inst.searchPositions = core.HighlightTableSearch(
				inst.ItemsTable.Table,
				inst.searchableView.GetText(),
				[]int{0},
			)
		}
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

func CreateDynamoDBHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x003388))
	defer core.ResetGlobalStyle()

	var (
		api = awsapi.NewDynamoDBApi(config, logger)

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

	var serviceRootView = core.NewServiceRootView(
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
