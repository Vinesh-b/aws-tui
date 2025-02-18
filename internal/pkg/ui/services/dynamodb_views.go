package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBDetailsPage struct {
	*core.ServicePageView
	TablesTable  *tables.DynamoDBTablesTable
	DetailsTable *tables.DynamoDBDetailsTable
	serviceCtx   *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBDetailsPage(
	detailsTable *tables.DynamoDBDetailsTable,
	tablesTable *tables.DynamoDBTablesTable,
	serviceContext *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBDetailsPage {
	const detailsSize = 3000
	const tablesSize = 5000

	var mainPage = core.NewResizableView(
		detailsTable, detailsSize,
		tablesTable, tablesSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceContext.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{detailsTable},
			{tablesTable},
		},
	)

	return &DynamoDBDetailsPage{
		ServicePageView: serviceView,
		TablesTable:     tablesTable,
		DetailsTable:    detailsTable,
		serviceCtx:      serviceContext,
	}
}

func (inst *DynamoDBDetailsPage) InitInputCapture() {}

type DynamoDBTableItemsPage struct {
	*core.ServicePageView
	ItemsTable       *tables.DynamoDBGenericTable
	serviceCtx       *core.ServiceContext[awsapi.DynamoDBApi]
	tableName        string
	tableDescription *types.TableDescription
	lastTableOp      tables.DDBTableOp
}

func NewDynamoDBTableItemsPage(
	itemsTable *tables.DynamoDBGenericTable,
	serviceContext *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBTableItemsPage {
	var expandItemView = core.CreateJsonTableDataView(
		serviceContext.AppContext, itemsTable, -1,
	)

	const expandItemViewSize = 3
	const itemsTableSize = 7

	var mainPage = core.NewResizableView(
		expandItemView, expandItemViewSize,
		itemsTable, itemsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceContext.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{expandItemView},
			{itemsTable},
		},
	)

	itemsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	return &DynamoDBTableItemsPage{
		ServicePageView:  serviceView,
		ItemsTable:       itemsTable,
		serviceCtx:       serviceContext,
		tableDescription: nil,
	}
}

func (inst *DynamoDBTableItemsPage) InitInputCapture() {}

func (inst *DynamoDBTableItemsPage) SetTableName(tableName string) *DynamoDBTableItemsPage {
	inst.tableName = tableName
	return inst
}

func NewDynamoDBHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x003388))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewDynamoDBApi(*appCtx.Config, appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		ddbDetailsView = NewDynamoDBDetailsPage(
			tables.NewDynamoDBDetailsTable(serviceCtx),
			tables.NewDynamoDBTablesTable(serviceCtx),
			serviceCtx,
		)
		ddbItemsView = NewDynamoDBTableItemsPage(
			tables.NewDynamoDBGenericTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(DYNAMODB), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Tables", ddbDetailsView, true).
		AddPage("Items", ddbItemsView, true, true)

	serviceRootView.InitPageNavigation()

	var selectedTableName = ""
	ddbDetailsView.TablesTable.SetSelectionChangedFunc(func(row, column int) {
		selectedTableName = ddbDetailsView.TablesTable.GetSelectedTable()
		ddbDetailsView.DetailsTable.SetSelectedTable(selectedTableName)
		ddbDetailsView.DetailsTable.RefreshDetails()
	})

	ddbDetailsView.TablesTable.SetSelectedFunc(func(row, column int) {
		selectedTableName = ddbDetailsView.TablesTable.GetSelectedTable()
		if len(selectedTableName) > 0 {
			ddbItemsView.ItemsTable.SetSelectedTable(selectedTableName)
			ddbItemsView.ItemsTable.ExecuteSearch(tables.DDBTableScan, expression.Expression{}, true)
			serviceRootView.ChangePage(1, nil)
		}
	})

	ddbDetailsView.InitInputCapture()

	ddbItemsView.
		SetTableName(selectedTableName).
		InitInputCapture()

	return serviceRootView
}
