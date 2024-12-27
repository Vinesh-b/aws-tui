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
	tableName        string
	tableDescription *types.TableDescription
	queryPkInput     *tview.InputField
	querySkInput     *tview.InputField
	runQueryBtn      *tview.Button
	lastTableOp      DDBTableOp
}

func NewDynamoDBTableItemsPage(
	itemsTable *DynamoDBGenericTable,
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTableItemsPage {
	var expandItemView = core.CreateExpandedLogView(app, itemsTable.Table, 0, core.DATA_TYPE_MAP_STRING_ANY)

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
		AddItem(itemsTable.RootView, 0, itemsTableSize, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(queryView, 0, 1, false).
			AddItem(scanView, 0, 1, false),
			5, 0, false,
		)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		expandItemView, itemsTable.RootView,
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
			itemsTable.RootView,
			expandItemView,
		},
	)

	return &DynamoDBTableItemsPage{
		ItemsTable:       itemsTable,
		RootView:         serviceView.RootView,
		app:              app,
		api:              api,
		tableDescription: nil,
		queryPkInput:     pkQueryValInput,
		querySkInput:     skQueryValInput,
		runQueryBtn:      runQueryBtn,
	}
}

func (inst *DynamoDBTableItemsPage) InitInputCapture() *DynamoDBTableItemsPage {
	inst.runQueryBtn.SetSelectedFunc(func() {
		inst.ItemsTable.SetSelectedTable(inst.tableName)
		inst.ItemsTable.SetQuery(
			inst.queryPkInput.GetText(),
			inst.querySkInput.GetText(),
			"",
		)
		inst.ItemsTable.RefreshQuery(true)
	})

	inst.ItemsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			const forceRefresh = true
			inst.ItemsTable.SetSelectedTable(inst.tableName)
			switch inst.lastTableOp {
			case DDB_TABLE_QUERY:
				inst.ItemsTable.SetQuery(
					inst.queryPkInput.GetText(),
					inst.querySkInput.GetText(),
					"",
				)
				inst.ItemsTable.RefreshQuery(forceRefresh)
			case DDB_TABLE_SCAN:
				inst.ItemsTable.RefreshScan(forceRefresh)
			}
		case tcell.KeyCtrlN:
			const forceRefresh = false
			switch inst.lastTableOp {
			case DDB_TABLE_QUERY:
				inst.ItemsTable.SetQuery(
					inst.queryPkInput.GetText(),
					inst.querySkInput.GetText(),
					"",
				)
				inst.ItemsTable.RefreshQuery(forceRefresh)
			case DDB_TABLE_SCAN:
				inst.ItemsTable.RefreshScan(forceRefresh)
			}
		}

		return event
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
		ddbItemsView.ItemsTable.SetSelectedTable(selectedTableName)
		ddbItemsView.ItemsTable.RefreshScan(true)
		serviceRootView.ChangePage(1, ddbItemsView.ItemsTable.Table)
	})

	ddbDetailsView.InitInputCapture()

	ddbItemsView.
		SetTableName(selectedTableName).
		InitInputCapture()

	return serviceRootView.RootView
}
