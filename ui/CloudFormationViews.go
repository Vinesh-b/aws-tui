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

type CloudFormationDetailsView struct {
	StacksTable         *tview.Table
	DetailsTable        *tview.Table
	SearchInput         *tview.InputField
	RefreshStacks       func(search string, reset bool)
	RefreshStackDetails func(stackName string)
	RootView            *tview.Flex
}

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

func createStacksTable(params tableCreationParams, api *cloudformation.CloudFormationApi) (
	*tview.Table, func(search string, reset bool),
) {
	var table = tview.NewTable()
	populateStacksTable(table, make(map[string]types.StackSummary, 0))

	var refreshViewsFunc = func(search string, reset bool) {
		var data map[string]types.StackSummary
		var dataChannel = make(chan map[string]types.StackSummary)
		var resultChannel = make(chan struct{})

		go func() {
			if len(search) > 0 {
				dataChannel <- api.FilterByName(search)
			} else {
				dataChannel <- api.ListStacks(reset)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateStacksTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func createStackDetailsTable(params tableCreationParams, api *cloudformation.CloudFormationApi) (
	*tview.Table, func(stackName string),
) {
	var table = tview.NewTable()
	populateStackDetailsTable(table, nil)

	var refreshViewsFunc = func(stackName string) {
		var data map[string]types.StackSummary
		var dataChannel = make(chan map[string]types.StackSummary)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListStacks(false)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			var details *types.StackSummary = nil
			var val, ok = data[stackName]
			if ok {
				details = &val
			}
			populateStackDetailsTable(table, details)
		})
	}

	return table, refreshViewsFunc
}

func NewStacksDetailsView(
	app *tview.Application,
	api *cloudformation.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationDetailsView {
	var (
		params = tableCreationParams{app, logger}

		stacksTable, refreshStacksTable     = createStacksTable(params, api)
		stacksDetails, refreshStacksDetails = createStackDetailsTable(params, api)
	)

	var onTableSelction = func(row int) {
		if row < 1 {
			return
		}
		refreshStacksDetails(stacksTable.GetCell(row, 0).Text)
	}

	stacksTable.SetSelectionChangedFunc(func(row, column int) {
		onTableSelction(row)
	})

	stacksTable.SetSelectedFunc(func(row, column int) {
		onTableSelction(row)
		app.SetFocus(stacksDetails)
	})

	var inputField = createSearchInput("Stacks")
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			refreshStacksTable(inputField.GetText(), false)
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var stacksView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(stacksDetails, 0, 5000, false).
		AddItem(stacksTable, 0, 3000, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var startIdx = 0
	initViewNavigation(app, stacksView, &startIdx,
		[]view{
			inputField,
			stacksTable,
			stacksDetails,
		},
	)
	return &CloudFormationDetailsView{
		StacksTable:         stacksTable,
		DetailsTable:        stacksDetails,
		SearchInput:         inputField,
		RefreshStacks:       refreshStacksTable,
		RefreshStackDetails: refreshStacksDetails,
		RootView:            stacksView,
	}
}

type CloudFormationStackEventsView struct {
	EventsTable   *tview.Table
	SearchInput   *tview.InputField
	RefreshEvents func(stackName string)
	RootView      *tview.Flex
	stackName     *string
	app           *tview.Application
}

func populateStackEventsTable(table *tview.Table, data []types.StackEvent) {
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

	initSelectableTable(table, "StackEvents",
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

func createStackEventsTable(params tableCreationParams, api *cloudformation.CloudFormationApi) (
	*tview.Table, func(stackName string),
) {
	var table = tview.NewTable()
	populateStackEventsTable(table, make([]types.StackEvent, 0))

	var refreshViewsFunc = func(stackName string) {
		var data []types.StackEvent
		var dataChannel = make(chan []types.StackEvent)
		var resultChannel = make(chan struct{})

		go func() {
			if len(stackName) > 0 {
				dataChannel <- api.DescribeStackEvents(stackName)
			} else {
				dataChannel <- make([]types.StackEvent, 0)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateStackEventsTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func NewStackEventsView(
	app *tview.Application,
	api *cloudformation.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationStackEventsView {
	var (
		params = tableCreationParams{app, logger}

		stackEventsTable, refreshStackEventsTable = createStackEventsTable(params, api)
	)

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
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			refreshStackEventsTable(inputField.GetText())
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var stackEventsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandedMsgView, 0, 5, false).
		AddItem(stackEventsTable, 0, 15, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var startIdx = 0
	initViewNavigation(app, stackEventsView, &startIdx,
		[]view{
			inputField,
			stackEventsTable,
		},
	)
	return &CloudFormationStackEventsView{
		EventsTable:   stackEventsTable,
		RefreshEvents: refreshStackEventsTable,
		SearchInput:   inputField,
		RootView:      stackEventsView,
		app:           app,
	}
}

func (inst *CloudFormationStackEventsView) InitSearchInputDoneCallback(search *string) {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			*search = inst.SearchInput.GetText()
			highlightTableSearch(inst.app, inst.EventsTable, *search, []int{})
			inst.app.SetFocus(inst.EventsTable)
		}
	})
}

func (inst *CloudFormationStackEventsView) InitInputCapture(stackName *string) {
	inst.EventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshEvents(*stackName)
		case tcell.KeyCtrlM:
			inst.RefreshEvents(*stackName)
		}
		return event
	})
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

	var selectedStackName = ""
	stackEventsView.RootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF5:
			stackEventsView.RefreshEvents(selectedStackName)
		}
		return event

	})
	stacksDetailsView.DetailsTable.SetSelectedFunc(func(row, column int) {
		selectedStackName = stacksDetailsView.DetailsTable.GetCell(0, 1).Text
		stackEventsView.RefreshEvents(selectedStackName)
		serviceRootView.ChangePage(1, stackEventsView.EventsTable)
	})

	var searchEvent = ""
	stackEventsView.InitInputCapture(&selectedStackName)
	stackEventsView.InitSearchInputDoneCallback(&searchEvent)

	return serviceRootView.RootView
}
