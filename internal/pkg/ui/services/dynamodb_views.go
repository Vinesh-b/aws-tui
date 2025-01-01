package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBDetailsPage struct {
	*core.ServicePageView
	TablesTable  *tables.DynamoDBTablesTable
	DetailsTable *tables.DynamoDBDetailsTable
	app          *tview.Application
	api          *awsapi.DynamoDBApi
}

func NewDynamoDBDetailsPage(
	detailsTable *tables.DynamoDBDetailsTable,
	tablesTable *tables.DynamoDBTablesTable,
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
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

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
	ItemsTable       *tables.DynamoDBGenericTable
	app              *tview.Application
	api              *awsapi.DynamoDBApi
	logger           *log.Logger
	tableName        string
	tableDescription *types.TableDescription
	lastTableOp      tables.DDBTableOp
}

func NewDynamoDBTableItemsPage(
	itemsTable *tables.DynamoDBGenericTable,
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTableItemsPage {
	var expandItemView = core.CreateExpandedLogView(app, itemsTable.Table, 0, core.DATA_TYPE_MAP_STRING_ANY)

	const expandItemViewSize = 3
	const itemsTableSize = 7

	var mainPage = core.NewResizableView(
		expandItemView, expandItemViewSize,
		itemsTable, itemsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			itemsTable,
			expandItemView,
		},
	)

	itemsTable.ErrorMessageCallback = func(text string) {
		serviceView.SetAndDisplayError(text)
	}

	return &DynamoDBTableItemsPage{
		ServicePageView:  serviceView,
		ItemsTable:       itemsTable,
		app:              app,
		api:              api,
		logger:           logger,
		tableDescription: nil,
	}
}

func (inst *DynamoDBTableItemsPage) InitInputCapture() {}

func (inst *DynamoDBTableItemsPage) SetTableName(tableName string) *DynamoDBTableItemsPage {
	inst.tableName = tableName
	return inst
}

func NewDynamoDBHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0x003388))
	defer core.ResetGlobalStyle()

	var (
		api = awsapi.NewDynamoDBApi(config, logger)

		ddbDetailsView = NewDynamoDBDetailsPage(
			tables.NewDynamoDBDetailsTable(app, api, logger),
			tables.NewDynamoDBTablesTable(app, api, logger),
			app, api, logger,
		)
		ddbItemsView = NewDynamoDBTableItemsPage(
			tables.NewDynamoDBGenericTable(app, api, logger),
			app, api, logger,
		)
	)

	var serviceRootView = core.NewServiceRootView(app, string(DYNAMODB))

	serviceRootView.
		AddAndSwitchToPage("Tables", ddbDetailsView, true).
		AddPage("Items", ddbItemsView, true, true)

	serviceRootView.InitPageNavigation()

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
			ddbItemsView.ItemsTable.ExecuteSearch(tables.DDBTableScan, expression.Expression{}, true)
			serviceRootView.ChangePage(1, ddbItemsView.ItemsTable.Table)
		}
	})

	ddbDetailsView.InitInputCapture()

	ddbItemsView.
		SetTableName(selectedTableName).
		InitInputCapture()

	return serviceRootView
}
