package dynamodb

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DdbTabName = string

const (
	DdbTabNameDetails DdbTabName = "Details"
	DdbTabNameTags    DdbTabName = "Tags"
	DdbTabNameMessage DdbTabName = "Message"
	DdbTabNameAddItem DdbTabName = "Add Item"
)

type DynamoDBDetailsPageView struct {
	*core.ServicePageView
	TablesTable  *DynamoDBTablesTable
	DetailsTable *DynamoDBDetailsTable
	TagsTable    *core.TagsTable[types.Tag, awsapi.DynamoDBApi]
	TabView      *core.TabViewHorizontal
	serviceCtx   *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBDetailsPageView(
	tablesTable *DynamoDBTablesTable,
	detailsTable *DynamoDBDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBDetailsPageView {
	var tagsTable = core.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() func() ([]types.Tag, error) {
			var arn = detailsTable.GetSelectedTableArn()
			return func() ([]types.Tag, error) {
				return serviceCtx.Api.ListTags(serviceCtx.Ctx, true, arn)
			}
		})

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(DdbTabNameDetails, detailsTable, 0, 1, true).
		AddTab(DdbTabNameTags, tagsTable, 0, 1, true)

	const tabViewSize = 3000
	const tablesTableSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		tablesTable, tablesTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &DynamoDBDetailsPageView{
		ServicePageView: serviceView,
		TablesTable:     tablesTable,
		DetailsTable:    detailsTable,
		TagsTable:       tagsTable,
		TabView:         tabView,
		serviceCtx:      serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	tablesTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{tablesTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *DynamoDBDetailsPageView) initInputCapture() {
	var loadedTabs = map[int]bool{}
	var lastSelection = ""
	var tabChangeFunc = func(tabName string, index int) {
		var selectedTable = inst.TablesTable.GetSelectedTable()

		if selectedTable != lastSelection {
			lastSelection = selectedTable
			utils.ClearMap(loadedTabs)
		}

		// Only automaticaly load new data on first change
		if loadedTabs[index] {
			return
		}

		switch tabName {
		case DdbTabNameTags:
			inst.TagsTable.RefreshDetails()
		}

		loadedTabs[index] = true
	}

	inst.TablesTable.SetSelectionChangedFunc(func(row, column int) {
		inst.DetailsTable.SetSelectedTable(inst.TablesTable.GetSelectedTable())
		inst.DetailsTable.RefreshDetails()
	})

	inst.TabView.SetOnTabChangeFunc(tabChangeFunc)
}

type DynamoDBItemsPageView struct {
	*core.ServicePageView
	ItemsTable  *DynamoDBGenericTable
	TabView     *core.TabViewHorizontal
	AddItemView *DynamoDBAddItemPageView
	serviceCtx  *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBItemsPageView(
	itemsTable *DynamoDBGenericTable,
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBItemsPageView {
	var expandItemView = core.CreateJsonTableDataView(
		serviceCtx.AppContext, itemsTable, -1,
	)
	expandItemView.SetJsonHighlighting(true)

	var addItemView = NewDynamoDBAddItemPageView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(DdbTabNameMessage, expandItemView, 0, 1, true).
		AddTab(DdbTabNameAddItem, addItemView, 0, 1, true)

	const tabViewSize = 3
	const itemsTableSize = 7

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		itemsTable, itemsTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &DynamoDBItemsPageView{
		ServicePageView: serviceView,
		ItemsTable:      itemsTable,
		TabView:         tabView,
		AddItemView:     addItemView,
		serviceCtx:      serviceCtx,
	}

	addItemView.SetOnItemAdded(view.LoadTableItems)
	addItemView.SetSelectedItemFunc(itemsTable.GetSelectedItem)

	itemsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{itemsTable},
		},
	)

	return view
}

func (inst *DynamoDBItemsPageView) LoadTableItems(tableName string) {
	inst.ItemsTable.SetSelectedTable(tableName)
	inst.AddItemView.SetSelectedTable(tableName)
	inst.ItemsTable.ExecuteSearch(DDBTableScan, expression.Expression{}, true)
}

func NewDynamoDBHomeView(appCtx *core.AppContext, api awsapi.DynamoDBApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x003388))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		ddbDetailsView = NewDynamoDBDetailsPageView(
			NewDynamoDBTablesTable(serviceCtx),
			NewDynamoDBDetailsTable(serviceCtx),
			serviceCtx,
		)
		ddbItemsView = NewDynamoDBItemsPageView(
			NewDynamoDBGenericTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.DYNAMODB), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Tables", ddbDetailsView, true).
		AddPage("Items", ddbItemsView, true, true)

	serviceRootView.InitPageNavigation()

	ddbDetailsView.TablesTable.SetSelectedFunc(func(row, column int) {
		if selectedTable := ddbDetailsView.TablesTable.GetSelectedTable(); len(selectedTable) > 0 {
			ddbItemsView.LoadTableItems(selectedTable)
			serviceRootView.ChangePage(1, nil)
		}
	})

	return serviceRootView
}
