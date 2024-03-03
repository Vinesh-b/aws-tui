package ui

import (
	"log"
	"time"

	"aws-tui/cloudwatchlogs"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogEventsView struct {
	LogEventsTable       *tview.Table
	ExpandedLogsTextArea *tview.TextArea
	SearchInput          *tview.InputField
	RootView             *tview.Flex
	selectedLogGroup     string
	selectedLogStream    string
	searchPositions      []int
	app                  *tview.Application
	api                  *cloudwatchlogs.CloudWatchLogsApi
}

func populateLogGroupsTable(table *tview.Table, data []types.LogGroup) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.LogGroupName,
		})
	}

	initSelectableTable(table, "LogGroups",
		tableRow{
			"Name",
		},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLogStreamsTable(table *tview.Table, data []types.LogStream, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			aws.ToString(row.LogStreamName),
			time.UnixMilli(aws.ToInt64(row.LastEventTimestamp)).Format(time.DateTime),
		})
	}

	var title = "LogStreams"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
		tableRow{
			"Name",
			"LastEventTimestamp",
		},
		tableData,
		[]int{1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLogEventsTable(table *tview.Table, data []types.OutputLogEvent, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			time.UnixMilli(*row.Timestamp).Format("2006-01-02 15:04:05.000"),
			*row.Message,
		})
	}

	var title = "LogEvents"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
		tableRow{
			"Timestamp",
			"Message",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(1, 0)
}

func NewLogEventsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogEventsView {
	var logEventsTable = tview.NewTable()
	populateLogEventsTable(logEventsTable, make([]types.OutputLogEvent, 0), false)

	var expandedLogsView = createExpandedLogView(app, logEventsTable, 1, DATA_TYPE_STRING)

	var inputField = createSearchInput("Log Events")

	const expandedLogsSize = 7
	const logTableSize = 13

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(expandedLogsView, 0, expandedLogsSize, false).
		AddItem(logEventsTable, 0, logTableSize, true).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		expandedLogsView, logEventsTable,
		expandedLogsSize, logTableSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			logEventsTable,
			expandedLogsView,
		},
	)

	return &LogEventsView{
		LogEventsTable:       logEventsTable,
		ExpandedLogsTextArea: expandedLogsView,
		SearchInput:          inputField,
		RootView:             serviceView.RootView,
		selectedLogGroup:     "",
		selectedLogStream:    "",
		app:                  app,
		api:                  api,
	}
}

func (inst *LogEventsView) RefreshEvents(selectedGroup string, selectedStream string, force bool) {
	inst.selectedLogGroup = selectedGroup
	inst.selectedLogStream = selectedStream
	var data []types.OutputLogEvent
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListLogEvents(
			inst.selectedLogGroup,
			inst.selectedLogStream,
			force,
		)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.LogEventsTable.Box, resultChannel, func() {
		populateLogEventsTable(inst.LogEventsTable, data, !force)
	})
}

func (inst *LogEventsView) InitInputCapture() {
	inst.SearchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			inst.searchPositions = highlightTableSearch(
				inst.app,
				inst.LogEventsTable,
				inst.SearchInput.GetText(),
				[]int{0, 1},
			)
			inst.app.SetFocus(inst.LogEventsTable)
		case tcell.KeyCtrlR:
			inst.SearchInput.SetText("")
			clearSearchHighlights(inst.LogEventsTable)
			inst.searchPositions = nil
		}
		return event
	})

	var nextSearch = 0
	inst.LogEventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshEvents(inst.selectedLogGroup, inst.selectedLogStream, true)
		case tcell.KeyCtrlN:
			inst.RefreshEvents(inst.selectedLogGroup, inst.selectedLogStream, false)
		}

		var searchCount = len(inst.searchPositions)
		if searchCount > 0 {
			switch event.Rune() {
			case rune('n'):
				nextSearch = (nextSearch + 1) % searchCount
				inst.LogEventsTable.Select(inst.searchPositions[nextSearch], 0)
			case rune('N'):
				nextSearch = (nextSearch - 1 + searchCount) % searchCount
				inst.LogEventsTable.Select(inst.searchPositions[nextSearch], 0)
			}
		}
		return event
	})
}

type LogStreamsView struct {
	LogStreamsTable    *tview.Table
	SearchInput        *tview.InputField
	RootView           *tview.Flex
	selectedLogGroup   string
	streamSearchbuffer *string
	app                *tview.Application
	api                *cloudwatchlogs.CloudWatchLogsApi
}

func NewLogStreamsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogStreamsView {
	var logStreamsTable = tview.NewTable()
	populateLogStreamsTable(logStreamsTable, make([]types.LogStream, 0), false)

	var inputField = createSearchInput("Stream Prefix")
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			app.SetFocus(logStreamsTable)
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(logStreamsTable, 0, 1, true).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			logStreamsTable,
		},
	)

	return &LogStreamsView{
		LogStreamsTable:    logStreamsTable,
		SearchInput:        inputField,
		RootView:           serviceView.RootView,
		selectedLogGroup:   "",
		streamSearchbuffer: nil,
		app:                app,
		api:                api,
	}
}

func (inst *LogStreamsView) RefreshStreams(groupName string, force bool) {
	inst.selectedLogGroup = groupName

	var data []types.LogStream
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListLogStreams(
			inst.selectedLogGroup,
			inst.streamSearchbuffer,
			force,
		)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.LogStreamsTable.Box, resultChannel, func() {
		populateLogStreamsTable(inst.LogStreamsTable, data, !force)
	})
}
func (inst *LogStreamsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			*inst.streamSearchbuffer = inst.SearchInput.GetText()
			inst.RefreshStreams(inst.selectedLogGroup, true)
			inst.app.SetFocus(inst.LogStreamsTable)
		}
	})

	inst.LogStreamsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshStreams(inst.selectedLogGroup, true)
		case tcell.KeyCtrlN:
			inst.RefreshStreams(inst.selectedLogGroup, false)
		}
		return event
	})
}

func (inst *LogStreamsView) InitSearchInputBuffer(searchBuffer *string) {
	inst.streamSearchbuffer = searchBuffer
}

type LogGroupsView struct {
	LogGroupsTable   *tview.Table
	SearchInput      *tview.InputField
	RootView         *tview.Flex
	selectedLogGroup string
	app              *tview.Application
	api              *cloudwatchlogs.CloudWatchLogsApi
}

func NewLogGroupsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsView {
	var logGroupsTable = tview.NewTable()
	populateLogGroupsTable(logGroupsTable, make([]types.LogGroup, 0))

	var inputField = createSearchInput("Log Groups")

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(logGroupsTable, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			logGroupsTable,
		},
	)

	return &LogGroupsView{
		LogGroupsTable:   logGroupsTable,
		SearchInput:      inputField,
		RootView:         serviceView.RootView,
		selectedLogGroup: "",
		app:              app,
		api:              api,
	}
}

func (inst *LogGroupsView) RefreshGroups(search string) {
	var data []types.LogGroup
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			data = inst.api.FilterGroupByName(search)
		} else {
			data = inst.api.ListLogGroups(false)
		}
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.LogGroupsTable.Box, resultChannel, func() {
		populateLogGroupsTable(inst.LogGroupsTable, data)
	})
}

func (inst *LogGroupsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshGroups(inst.SearchInput.GetText())
			inst.app.SetFocus(inst.LogGroupsTable)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})
}

func createLogsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer resetGlobalStyle()

	var api = cloudwatchlogs.NewCloudWatchLogsApi(config, logger)
	var logEventsView = NewLogEventsView(app, api, logger)
	var logStreamsView = NewLogStreamsView(app, api, logger)
	var logGroupsView = NewLogGroupsView(app, api, logger)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView.RootView, true, true).
		AddPage("Streams", logStreamsView.RootView, true, true).
		AddAndSwitchToPage("Groups", logGroupsView.RootView, true)

	var orderedPages = []string{
		"Groups",
		"Streams",
		"Events",
	}

	var serviceRootView = NewServiceRootView(
		app, string(CLOUDWATCH_LOGS_GROUPS), pages, orderedPages).Init()

	var selectedGroupName = ""
	logGroupsView.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		selectedGroupName = logGroupsView.LogGroupsTable.GetCell(row, 0).Text
		logStreamsView.RefreshStreams(selectedGroupName, true)
		serviceRootView.ChangePage(1, logStreamsView.LogStreamsTable)
	})

	var streamName = ""
	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		streamName = logStreamsView.LogStreamsTable.GetCell(row, 0).Text
		logEventsView.RefreshEvents(selectedGroupName, streamName, true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	var searchPrefix = ""
	logEventsView.InitInputCapture()
	logStreamsView.InitInputCapture()
	logStreamsView.InitSearchInputBuffer(&searchPrefix)
	logGroupsView.InitInputCapture()

	return serviceRootView.RootView
}
