package ui

import (
	"log"

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
) *tview.Pages {
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
	initPageNavigation(app, pages, &pagesNavIdx, orderedPages)

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

	return pages
}
