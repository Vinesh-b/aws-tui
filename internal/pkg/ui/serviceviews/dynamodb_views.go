package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBDetailsPage struct {
	*core.ServicePageView
	TablesTable  *DynamoDBTablesTable
	DetailsTable *DynamoDBDetailsTable
	app          *tview.Application
	api          *awsapi.DynamoDBApi
}

func NewDynamoDBDetailsPage(
	detailsTable *DynamoDBDetailsTable,
	tablesTable *DynamoDBTablesTable,
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBDetailsPage {
	const detailsSize = 3000
	const tablesSize = 5000

	var mainPage = core.NewResizableView(
		detailsTable, detailsSize,
		tablesTable, tablesSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			tablesTable,
			detailsTable,
		},
	)

	return &DynamoDBDetailsPage{
		ServicePageView: serviceView,
		TablesTable:     tablesTable,
		DetailsTable:    detailsTable,
		app:             app,
		api:             api,
	}
}

func (inst *DynamoDBDetailsPage) InitInputCapture() {}

type DynamoDBTableItemsPage struct {
	*core.ServicePageView
	ItemsTable       *DynamoDBGenericTable
	app              *tview.Application
	api              *awsapi.DynamoDBApi
	logger           *log.Logger
	tableName        string
	tableDescription *types.TableDescription
	lastTableOp      DDBTableOp
}

func NewDynamoDBTableItemsPage(
	itemsTable *DynamoDBGenericTable,
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTableItemsPage {
	var expandItemView = core.CreateExpandedLogView(app, itemsTable.Table, 0, core.DATA_TYPE_MAP_STRING_ANY)

	const expandItemViewSize = 3
	const itemsTableSize = 7

	var mainPage = core.NewResizableView(
		expandItemView, expandItemViewSize,
		itemsTable.RootView, itemsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			itemsTable.RootView,
			expandItemView,
		},
	)

	return &DynamoDBTableItemsPage{
		ServicePageView:  serviceView,
		ItemsTable:       itemsTable,
		app:              app,
		api:              api,
		logger:           logger,
		tableDescription: nil,
	}
}

func (inst *DynamoDBTableItemsPage) InitInputCapture() *DynamoDBTableItemsPage {
	inst.ItemsTable.QueryDoneButton.SetSelectedFunc(func() {
		inst.ItemsTable.SetPartitionKeyName(inst.ItemsTable.pkName)
		inst.ItemsTable.SetSortKeyName(inst.ItemsTable.skName)

		var expr, err = inst.ItemsTable.GenerateQueryExpression()
		if err != nil {
			inst.logger.Println(err.Error())
			return
		}
		inst.ItemsTable.ExecuteSearch(DDBTableQuery, expr, true)
	})

	inst.ItemsTable.ScanDoneButton.SetSelectedFunc(func() {
		var expr, err = inst.ItemsTable.GenerateScanExpression()
		if err != nil {
			inst.logger.Println(err.Error())
			return
		}
		inst.ItemsTable.ExecuteSearch(DDBTableScan, expr, true)
	})

	return inst
}

func (inst *DynamoDBTableItemsPage) SetTableName(tableName string) *DynamoDBTableItemsPage {
	inst.tableName = tableName
	return inst
}

func NewDynamoDBHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x003388))
	defer core.ResetGlobalStyle()

	var (
		api = awsapi.NewDynamoDBApi(config, logger)

		ddbDetailsView = NewDynamoDBDetailsPage(
			NewDynamoDBDetailsTable(app, api, logger),
			NewDynamoDBTablesTable(app, api, logger),
			app, api, logger,
		)
		ddbItemsView = NewDynamoDBTableItemsPage(
			NewDynamoDBGenericTable(app, api, logger),
			app, api, logger,
		)
	)

	var pages = tview.NewPages()
	pages.
		AddPage("Items", ddbItemsView, true, true).
		AddPage("Tables", ddbDetailsView, true, true)

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
		selectedTableName = ddbDetailsView.TablesTable.GetSelectedTable()
		ddbDetailsView.DetailsTable.SetSelectedTable(selectedTableName)
		ddbDetailsView.DetailsTable.RefreshDetails()
	})

	ddbDetailsView.TablesTable.SetSelectedFunc(func(row, column int) {
		selectedTableName = ddbDetailsView.TablesTable.GetSelectedTable()
		if len(selectedTableName) > 0 {
			ddbItemsView.ItemsTable.SetSelectedTable(selectedTableName)
			ddbItemsView.ItemsTable.ExecuteSearch(DDBTableScan, expression.Expression{}, true)
			serviceRootView.ChangePage(1, ddbItemsView.ItemsTable.Table)
		}
	})

	ddbDetailsView.InitInputCapture()

	ddbItemsView.
		SetTableName(selectedTableName).
		InitInputCapture()

	return serviceRootView.RootView
}
