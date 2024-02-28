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

type DynamoDBDetailsView struct {
	DDBTablesTable     *tview.Table
	DetailsTable       *tview.Table
	SearchInput        *tview.InputField
	RefreshTablesTable func(search string)
	RefreshDetails     func(tableName string)
	RootView           *tview.Flex
}

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
	table.Select(0, 0)
	table.ScrollToBeginning()
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
	table.ScrollToBeginning()
}

func populateDynamoDBTable(
	table *tview.Table,
	description *types.TableDescription,
	data []map[string]interface{},
) {
	table.
		Clear().
		SetBorders(false).
		SetFixed(1, 2)
	table.
		SetTitle("Table").
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	if description == nil || len(data) == 0 {
		return
	}

	var headingIdx = 0
	var headingIdxMap = make(map[string]int)
	for _, atter := range description.KeySchema {
		switch atter.KeyType {
		case types.KeyTypeHash:
			headingIdxMap[*atter.AttributeName] = 0
			headingIdx++
		case types.KeyTypeRange:
			headingIdxMap[*atter.AttributeName] = 1
			headingIdx++
		}
	}

	for rowIdx, rowData := range data {
		for heading := range rowData {
			var colIdx, ok = headingIdxMap[heading]
			if !ok {
				headingIdxMap[heading] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellData = fmt.Sprintf("%v", rowData[heading])
			var previewText = clampStringLen(&cellData, 100)
			table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(previewText).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}

	for heading, colIdx := range headingIdxMap {
		table.SetCell(0, colIdx, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(secondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(contrastBackgroundColor),
		)
	}

	if len(data) > 0 {
		table.SetSelectable(true, false).SetSelectedStyle(
			tcell.Style{}.Background(moreContrastBackgroundColor),
		)
	}
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func createDynamoDBTablesTable(
	params tableCreationParams,
	api *dynamodb.DynamoDBApi,
) (*tview.Table, func(search string)) {
	var table = tview.NewTable()
	populateDynamoDBTabelsTable(table, make([]string, 0))

	var refreshViewsFunc = func(search string) {
		table.Clear()
		var data []string
		var dataChannel = make(chan []string)
		var resultChannel = make(chan struct{})

		go func() {
			if len(search) > 0 {
				//dataChannel <- api.FilterTablesByName(search)
			} else {
				dataChannel <- api.ListTables(false)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateDynamoDBTabelsTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func createDynamoDBTableDetailsTable(
	params tableCreationParams,
	api *dynamodb.DynamoDBApi,
) (*tview.Table, func(tableName string)) {
	var table = tview.NewTable()
	populateDynamoDBTabelDetailsTable(table, nil)

	var refreshViewsFunc = func(tableName string) {
		table.Clear()
		var data *types.TableDescription = nil
		var dataChannel = make(chan *types.TableDescription)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.DescribeTable(tableName)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateDynamoDBTabelDetailsTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func NewDynamoDBDetailsView(
	app *tview.Application,
	api *dynamodb.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBDetailsView {
	var (
		params = tableCreationParams{app, logger}

		tablesTable, refreshTablesTable   = createDynamoDBTablesTable(params, api)
		detailsTable, refreshDetailsTable = createDynamoDBTableDetailsTable(params, api)
	)

	var inputField = tview.NewInputField().
		SetLabel(" Search Tables: ").
		SetFieldWidth(64)
	inputField.SetBorder(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			go refreshTablesTable(inputField.GetText())
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var ddbDetailsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(detailsTable, 0, 1, false).
		AddItem(tablesTable, 0, 4, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var viewNavIdx = 0
	initViewNavigation(app, ddbDetailsView, &viewNavIdx,
		[]view{
			inputField,
			tablesTable,
			detailsTable,
		},
	)

	return &DynamoDBDetailsView{
		DDBTablesTable:     tablesTable,
		DetailsTable:       detailsTable,
		SearchInput:        inputField,
		RefreshTablesTable: refreshTablesTable,
		RefreshDetails:     refreshDetailsTable,
		RootView:           ddbDetailsView,
	}
}

type DynamoDBTableItemsView struct {
	DDBItemsTable *tview.Table
	SearchInput   *tview.InputField
	RefreshTable  func(search string)
	RootView      *tview.Flex
	app           *tview.Application
}

func createDynamoDBItemsTable(
	params tableCreationParams,
	api *dynamodb.DynamoDBApi,
) (*tview.Table, func(tableName string)) {
	var table = tview.NewTable()
	populateDynamoDBTable(table, nil, make([]map[string]interface{}, 0))

	var refreshViewsFunc = func(tableName string) {
		var data []map[string]interface{}
		var dataChannel = make(chan []map[string]interface{})
		var descData *types.TableDescription = nil
		var descDataChannel = make(chan *types.TableDescription)
		var resultChannel = make(chan struct{})

		go func() {
			if len(tableName) <= 0 {
				dataChannel <- make([]map[string]interface{}, 0)
				return
			}
			var description = api.DescribeTable(tableName)
			descDataChannel <- description
			dataChannel <- api.ScanTable(description)
		}()

		go func() {
			descData = <-descDataChannel
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateDynamoDBTable(table, descData, data)
		})
	}

	return table, refreshViewsFunc
}

func NewDynamoDBTableItemsView(
	app *tview.Application,
	api *dynamodb.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTableItemsView {
	var (
		params = tableCreationParams{app, logger}

		itemsTable, refreshItemsTable = createDynamoDBItemsTable(params, api)
	)

	var inputField = tview.NewInputField().
		SetLabel(" Search Tables: ").
		SetFieldWidth(64)
	inputField.SetBorder(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			go refreshItemsTable(inputField.GetText())
			app.SetFocus(itemsTable)
		case tcell.KeyEsc:
			inputField.SetText("")
			highlightTableSearch(app, itemsTable, "", []int{})
		default:
			return
		}
	})
	var ddbDetailsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(itemsTable, 0, 4, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var viewNavIdx = 0
	initViewNavigation(app, ddbDetailsView, &viewNavIdx,
		[]view{
			inputField,
			itemsTable,
		},
	)

	return &DynamoDBTableItemsView{
		DDBItemsTable: itemsTable,
		SearchInput:   inputField,
		RefreshTable:  refreshItemsTable,
		RootView:      ddbDetailsView,
		app:           app,
	}
}
func (inst *DynamoDBTableItemsView) InitSearchInputDoneCallback(search *string) {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			*search = inst.SearchInput.GetText()
			highlightTableSearch(inst.app, inst.DDBItemsTable, *search, []int{})
			inst.app.SetFocus(inst.DDBItemsTable)
		}
	})
}

func createDynamoDBHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	var (
		api = dynamodb.NewDynamoDBApi(config, logger)

		ddbDetailsView = NewDynamoDBDetailsView(app, api, logger)
		ddbItemsView   = NewDynamoDBTableItemsView(app, api, logger)
	)

	var pages = tview.NewPages()
	pages.
		AddPage("TableItems", ddbItemsView.RootView, true, true).
		AddPage("Home", ddbDetailsView.RootView, true, true)

	var pagesNavIdx = 0
	var orderedPages = []string{
		"Home",
		"TableItems",
	}

	var paginationView = createPaginatorView()
	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(paginationView.RootView, 1, 0, false)

	initPageNavigation(app, pages, &pagesNavIdx, orderedPages, paginationView.PageCounterView)

	var switchAndFocus = func(pageIdx int, view tview.Primitive) {
		pagesNavIdx = pageIdx
		pages.SwitchToPage(orderedPages[pageIdx])
		app.SetFocus(view)
	}

	ddbDetailsView.DDBTablesTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var selectedTableName = ddbDetailsView.DDBTablesTable.GetCell(row, 0).Text
		go ddbDetailsView.RefreshDetails(selectedTableName)
	})

	ddbDetailsView.DDBTablesTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var selectedTableName = ddbDetailsView.DDBTablesTable.GetCell(row, 0).Text
		go ddbItemsView.RefreshTable(selectedTableName)
		switchAndFocus(1, ddbItemsView.DDBItemsTable)
	})

	var searchString = ""
	ddbItemsView.InitSearchInputDoneCallback(&searchString)

	return rootView
}
