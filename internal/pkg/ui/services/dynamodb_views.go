package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"
	"aws-tui/internal/pkg/utils"

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
)

type DynamoDBDetailsPageView struct {
	*core.ServicePageView
	TablesTable  *tables.DynamoDBTablesTable
	DetailsTable *tables.DynamoDBDetailsTable
	TagsTable    *tables.TagsTable[types.Tag, awsapi.DynamoDBApi]
	TabView      *core.TabViewHorizontal
	serviceCtx   *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBDetailsPageView(
	tablesTable *tables.DynamoDBTablesTable,
	detailsTable *tables.DynamoDBDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBDetailsPageView {
	var tagsTable = tables.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() ([]types.Tag, error) {
			return serviceCtx.Api.ListTags(
				true, detailsTable.GetSelectedTableArn(),
			)
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
			{tabView.GetTabDisplayView()},
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
	ItemsTable *tables.DynamoDBGenericTable
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBItemsPageView(
	itemsTable *tables.DynamoDBGenericTable,
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBItemsPageView {
	var expandItemView = core.CreateJsonTableDataView(
		serviceCtx.AppContext, itemsTable, -1,
	)

	const expandItemViewSize = 3
	const itemsTableSize = 7

	var mainPage = core.NewResizableView(
		expandItemView, expandItemViewSize,
		itemsTable, itemsTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &DynamoDBItemsPageView{
		ServicePageView: serviceView,
		ItemsTable:      itemsTable,
		serviceCtx:      serviceCtx,
	}

	itemsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	view.InitViewNavigation(
		[][]core.View{
			{expandItemView},
			{itemsTable},
		},
	)

	return view
}

func (inst *DynamoDBItemsPageView) LoadTableItems(tableName string) {
	inst.ItemsTable.SetSelectedTable(tableName)
	inst.ItemsTable.ExecuteSearch(tables.DDBTableScan, expression.Expression{}, true)
}

func NewDynamoDBHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x003388))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewDynamoDBApi(appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		ddbDetailsView = NewDynamoDBDetailsPageView(
			tables.NewDynamoDBTablesTable(serviceCtx),
			tables.NewDynamoDBDetailsTable(serviceCtx),
			serviceCtx,
		)
		ddbItemsView = NewDynamoDBItemsPageView(
			tables.NewDynamoDBGenericTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(DYNAMODB), appCtx)

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
