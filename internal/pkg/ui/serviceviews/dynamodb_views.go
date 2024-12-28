package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBDetailsPage struct {
	TablesTable  *DynamoDBTablesTable
	DetailsTable *DynamoDBDetailsTable
	RootView     *tview.Flex
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

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(detailsTable.RootView, 0, detailsSize, false).
		AddItem(tablesTable.RootView, 0, tablesSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		detailsTable.RootView, tablesTable.RootView,
		detailsSize, tablesSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			tablesTable.RootView,
			detailsTable.RootView,
		},
	)

	return &DynamoDBDetailsPage{
		TablesTable:  tablesTable,
		DetailsTable: detailsTable,
		RootView:     serviceView.RootView,
		app:          app,
		api:          api,
	}
}

func (inst *DynamoDBDetailsPage) InitInputCapture() {}

type DynamoDBTableItemsPage struct {
	ItemsTable       *DynamoDBGenericTable
	RootView         *tview.Flex
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

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandItemView, 0, expandItemViewSize, false).
		AddItem(itemsTable.RootView, 0, itemsTableSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		expandItemView, itemsTable.RootView,
		expandItemViewSize, itemsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			itemsTable.RootView,
			expandItemView,
		},
	)

	return &DynamoDBTableItemsPage{
		ItemsTable:       itemsTable,
		RootView:         serviceView.RootView,
		app:              app,
		api:              api,
		logger:           logger,
		tableDescription: nil,
	}
}

func (inst *DynamoDBTableItemsPage) InitInputCapture() *DynamoDBTableItemsPage {
	inst.ItemsTable.DoneButton.SetSelectedFunc(func() {
		inst.ItemsTable.SetPartitionKeyName(inst.ItemsTable.pkName)
		inst.ItemsTable.SetSortKeyName(inst.ItemsTable.skName)

		var expr = inst.ItemsTable.GenerateQueryExpression()
		inst.ItemsTable.RefreshQuery(expr, true)
	})

	return inst
}

func (inst *DynamoDBTableItemsPage) SetTableName(tableName string,
) *DynamoDBTableItemsPage {
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
		selectedTableName = ddbDetailsView.TablesTable.GetSelectedTable()
		ddbDetailsView.DetailsTable.SetSelectedTable(selectedTableName)
		ddbDetailsView.DetailsTable.RefreshDetails()
	})

	ddbDetailsView.TablesTable.SetSelectedFunc(func(row, column int) {
		selectedTableName = ddbDetailsView.TablesTable.GetSelectedTable()
		if len(selectedTableName) > 0 {
			ddbItemsView.ItemsTable.SetSelectedTable(selectedTableName)
			ddbItemsView.ItemsTable.RefreshScan(true)
			serviceRootView.ChangePage(1, ddbItemsView.ItemsTable.Table)
		}
	})

	ddbDetailsView.InitInputCapture()

	ddbItemsView.
		SetTableName(selectedTableName).
		InitInputCapture()

	return serviceRootView.RootView
}
