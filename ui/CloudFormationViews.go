package ui

import (
	"log"
	"time"

	"aws-tui/cloudformation"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func populateStacksTable(table *tview.Table, data map[string]types.StackSummary) {
	var tableData []tableRow
	for _, row := range data {
		var lastUpdated = "-"
		if row.LastUpdatedTime != nil {
			lastUpdated = row.LastUpdatedTime.Format(time.DateTime)
		}
		tableData = append(tableData, tableRow{
			*row.StackName,
			string(row.StackStatus),
			lastUpdated,
		})
	}

	initSelectableTable(table, "Stacks",
		tableRow{
			"StackName",
			"Status",
			"LastUpdated",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateStackDetailsTable(table *tview.Table, data *types.StackSummary) {
	var tableData []tableRow
	if data != nil {
		var lastUpdated = "-"
		if data.LastUpdatedTime != nil {
			lastUpdated = data.LastUpdatedTime.Format(time.DateTime)
		}
		tableData = []tableRow{
			{"Name", aws.ToString(data.StackName)},
			{"StackId", aws.ToString(data.StackId)},
			{"Description", aws.ToString(data.TemplateDescription)},
			{"Status", string(data.StackStatus)},
			{"StatusReason", aws.ToString(data.StackStatusReason)},
			{"CreationTime", data.CreationTime.Format(time.DateTime)},
			{"LastUpdated", lastUpdated},
		}
	}

	initBasicTable(table, "Stack Details", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

type CloudFormationDetailsView struct {
	StacksTable  *tview.Table
	DetailsTable *tview.Table
	SearchInput  *tview.InputField
	RootView     *tview.Flex
	app          *tview.Application
	api          *cloudformation.CloudFormationApi
}

func NewStacksDetailsView(
	app *tview.Application,
	api *cloudformation.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationDetailsView {
	var stacksTable = tview.NewTable()
	populateStacksTable(stacksTable, make(map[string]types.StackSummary, 0))

	var stacksDetails = tview.NewTable()
	populateStackDetailsTable(stacksDetails, nil)

	var inputField = createSearchInput("Stacks")

	const stackDetailsSize = 5000
	const stackTablesSize = 3000

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(stacksDetails, 0, stackDetailsSize, false).
		AddItem(stacksTable, 0, stackTablesSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		stacksDetails, stacksTable,
		stackDetailsSize, stackTablesSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			stacksTable,
			stacksDetails,
		},
	)
	return &CloudFormationDetailsView{
		StacksTable:  stacksTable,
		DetailsTable: stacksDetails,
		SearchInput:  inputField,
		RootView:     serviceView.RootView,
		app:          app,
		api:          api,
	}
}

func (inst *CloudFormationDetailsView) RefreshStacks(search string, reset bool) {
	var data map[string]types.StackSummary
	var dataChannel = make(chan map[string]types.StackSummary)
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			dataChannel <- inst.api.FilterByName(search)
		} else {
			dataChannel <- inst.api.ListStacks(reset)
		}
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.StacksTable.Box, resultChannel, func() {
		populateStacksTable(inst.StacksTable, data)
	})
}

func (inst *CloudFormationDetailsView) RefreshDetails(stackName string, force bool) {
	var data map[string]types.StackSummary
	var dataChannel = make(chan map[string]types.StackSummary)
	var resultChannel = make(chan struct{})

	go func() {
		dataChannel <- inst.api.ListStacks(force)
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.DetailsTable.Box, resultChannel, func() {
		var details *types.StackSummary = nil
		var val, ok = data[stackName]
		if ok {
			details = &val
		}
		populateStackDetailsTable(inst.DetailsTable, details)
	})
}

func (inst *CloudFormationDetailsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshStacks(inst.SearchInput.GetText(), false)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})

	var refreshDetails = func(row int, force bool) {
		if row < 1 {
			return
		}
		inst.RefreshDetails(inst.StacksTable.GetCell(row, 0).Text, force)
	}

	inst.StacksTable.SetSelectionChangedFunc(func(row, column int) {
		refreshDetails(row, false)
	})

	inst.StacksTable.SetSelectedFunc(func(row, column int) {
		refreshDetails(row, false)
		inst.app.SetFocus(inst.DetailsTable)
	})

	inst.StacksTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshStacks("", true)
		}
		return event
	})

	inst.DetailsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var selctedRow, _ = inst.StacksTable.GetSelection()
		switch event.Key() {
		case tcell.KeyCtrlR:
			refreshDetails(selctedRow, true)
		}
		return event
	})
}

func populateStackEventsTable(table *tview.Table, data []types.StackEvent, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			row.Timestamp.Format("2006-01-02 15:04:05.000"),
			aws.ToString(row.LogicalResourceId),
			aws.ToString(row.ResourceType),
			string(row.ResourceStatus),
			aws.ToString(row.ResourceStatusReason),
		})
	}

	var title = "StackEvents"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
		tableRow{
			"Timestamp",
			"LogicalId",
			"ResourceType",
			"Status",
			"Reason",
		},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

type CloudFormationStackEventsView struct {
	EventsTable        *tview.Table
	SearchInput        *tview.InputField
	RootView           *tview.Flex
	selectedStack      string
	searchEventsBuffer *string
	app                *tview.Application
	api                *cloudformation.CloudFormationApi
}

func NewStackEventsView(
	app *tview.Application,
	api *cloudformation.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationStackEventsView {
	var stackEventsTable = tview.NewTable()
	populateStackEventsTable(stackEventsTable, make([]types.StackEvent, 0), false)

	var expandedMsgView = tview.NewTextArea()
	expandedMsgView.
		SetBorder(true).
		SetTitle("Message").
		SetTitleAlign(tview.AlignLeft)

	stackEventsTable.SetSelectionChangedFunc(func(row, column int) {
		var privateData = stackEventsTable.GetCell(row, 4).Reference
		if row < 1 || privateData == nil {
			return
		}
		var logText = privateData.(string)
		expandedMsgView.SetText(logText, false)
	})

	var inputField = createSearchInput("Events")

	const expandedMsgSize = 5
	const stackEventsSize = 15

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(expandedMsgView, 0, expandedMsgSize, false).
		AddItem(stackEventsTable, 0, stackEventsSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		expandedMsgView, stackEventsTable,
		expandedMsgSize, stackEventsSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			stackEventsTable,
		},
	)
	return &CloudFormationStackEventsView{
		EventsTable:        stackEventsTable,
		SearchInput:        inputField,
		RootView:           serviceView.RootView,
		selectedStack:      "",
		searchEventsBuffer: nil,
		app:                app,
		api:                api,
	}
}

func (inst *CloudFormationStackEventsView) RefreshEvents(stackName string, force bool) {
	inst.selectedStack = stackName

	var data []types.StackEvent
	var dataChannel = make(chan []types.StackEvent)
	var resultChannel = make(chan struct{})

	go func() {
		if len(stackName) > 0 {
			dataChannel <- inst.api.DescribeStackEvents(inst.selectedStack, force)
		} else {
			dataChannel <- make([]types.StackEvent, 0)
		}
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.EventsTable.Box, resultChannel, func() {
		populateStackEventsTable(inst.EventsTable, data, !force)
	})
}

func (inst *CloudFormationStackEventsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			*inst.searchEventsBuffer = inst.SearchInput.GetText()
			highlightTableSearch(inst.app, inst.EventsTable, *inst.searchEventsBuffer, []int{})
			inst.app.SetFocus(inst.EventsTable)
		}
	})

	inst.EventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshEvents(inst.selectedStack, true)
		case tcell.KeyCtrlN:
			inst.RefreshEvents(inst.selectedStack, false)
		}
		return event
	})
}

func (inst *CloudFormationStackEventsView) InitSearchInputBuffer(searchStringBuffer *string) {
	inst.searchEventsBuffer = searchStringBuffer
}

func createStacksHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0x660033))
	defer resetGlobalStyle()

	var (
		api               = cloudformation.NewCloudFormationApi(config, logger)
		stacksDetailsView = NewStacksDetailsView(app, api, logger)
		stackEventsView   = NewStackEventsView(app, api, logger)
	)

	var pages = tview.NewPages().
		AddPage("Events", stackEventsView.RootView, true, true).
		AddAndSwitchToPage("Stacks", stacksDetailsView.RootView, true)

	var orderedPages = []string{
		"Stacks",
		"Events",
	}

	var serviceRootView = NewServiceRootView(
		app, string(CLOUDFORMATION), pages, orderedPages).Init()

	stacksDetailsView.DetailsTable.SetSelectedFunc(func(row, column int) {
		var selectedStackName = stacksDetailsView.DetailsTable.GetCell(0, 1).Text
		stackEventsView.RefreshEvents(selectedStackName, true)
		serviceRootView.ChangePage(1, stackEventsView.EventsTable)
	})

	var searchEvent = ""
	stackEventsView.InitSearchInputBuffer(&searchEvent)
	stackEventsView.InitInputCapture()
	stacksDetailsView.InitInputCapture()

	return serviceRootView.RootView
}
