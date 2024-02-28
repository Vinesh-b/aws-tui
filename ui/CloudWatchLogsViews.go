package ui

import (
	"encoding/json"
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
	RefreshEvents        func(groupName string, streamName string, extend bool)
	RootView             *tview.Flex

	app *tview.Application
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
			*row.LogStreamName,
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
		},
		tableData,
		[]int{0},
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
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func createLogItemsTable(
	params tableCreationParams,
	api *cloudwatchlogs.CloudWatchLogsApi,
) (*tview.Table, func(groupName string, streamName string, extend bool)) {
	var table = tview.NewTable()
	populateLogEventsTable(table, make([]types.OutputLogEvent, 0), false)

	var refreshViewsFunc = func(groupName string, streamName string, extend bool) {
		var data []types.OutputLogEvent
		var dataChannel = make(chan []types.OutputLogEvent)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListLogEvents(groupName, streamName, !extend)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateLogEventsTable(table, data, extend)
		})
	}

	return table, refreshViewsFunc
}

func NewLogEventsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogEventsView {
	var (
		params                                = tableCreationParams{app, logger}
		logEventsTable, refreshLogEventsTable = createLogItemsTable(params, api)
	)

	var expandedLogsView = tview.NewTextArea().SetSelectedStyle(
		tcell.Style{}.Background(tview.Styles.MoreContrastBackgroundColor),
	)
	expandedLogsView.
		SetBorder(true).
		SetTitle("Message").
		SetTitleAlign(tview.AlignLeft)

	logEventsTable.SetSelectionChangedFunc(func(row, column int) {
		var privateData = logEventsTable.GetCell(row, 1).Reference
		if row < 1 || privateData == nil {
			return
		}
		var logText = privateData.(string)
		var anyJson map[string]interface{}

		var err = json.Unmarshal([]byte(logText), &anyJson)
		if err == nil {
			var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
			logText = string(jsonBytes)
		}
		expandedLogsView.SetText(logText, false)
	})

	logEventsTable.SetSelectedFunc(func(row, column int) {
		app.SetFocus(expandedLogsView)
	})

	var inputField = tview.NewInputField().
		SetLabel(" Search Log Events: ").
		SetFieldWidth(64)
	inputField.SetBorder(true)
	inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inputField.SetText("")
			highlightTableSearch(app, logEventsTable, "", []int{})
		}
		return event
	})

	var eventsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandedLogsView, 0, 7, false).
		AddItem(logEventsTable, 0, 13, true).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var eventsViewNavIdx = 0
	initViewNavigation(app, eventsView, &eventsViewNavIdx,
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
		RefreshEvents:        refreshLogEventsTable,
		RootView:             eventsView,
		app:                  app,
	}
}

func (inst *LogEventsView) InitSearchInputDoneCallback(search *string) {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			*search = inst.SearchInput.GetText()
			highlightTableSearch(inst.app, inst.LogEventsTable, *search, []int{0, 1})
			inst.app.SetFocus(inst.LogEventsTable)
		}
	})
}

func (inst *LogEventsView) InitInputCapture(selectedGroupName *string, streamName *string) {
	inst.LogEventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshEvents(*selectedGroupName, *streamName, false)
		case tcell.KeyCtrlM:
			inst.RefreshEvents(*selectedGroupName, *streamName, true)
		}
		return event
	})
}

type LogStreamsView struct {
	LogStreamsTable *tview.Table
	SearchInput     *tview.InputField
	RefreshStreams  func(groupName string, searchPrefix *string, extend bool)
	RootView        *tview.Flex
	app             *tview.Application
}

func createLogStreamsTable(
	params tableCreationParams,
	api *cloudwatchlogs.CloudWatchLogsApi,
) (*tview.Table, func(groupName string, searchPrefix *string, extend bool)) {
	var table = tview.NewTable()
	populateLogStreamsTable(table, make([]types.LogStream, 0), false)

	var refreshViewsFunc = func(groupName string, searchPrefix *string, extend bool) {
		var data []types.LogStream
		var dataChannel = make(chan []types.LogStream)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListLogStreams(groupName, searchPrefix, !extend)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateLogStreamsTable(table, data, extend)
		})
	}

	return table, refreshViewsFunc
}

func NewLogStreamsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogStreamsView {
	var (
		params                                 = tableCreationParams{app, logger}
		logStreamsTable, refreshLogStreamTable = createLogStreamsTable(params, api)
	)

	var inputField = tview.NewInputField().
		SetLabel(" Search Stream Prefix: ").
		SetFieldWidth(64)
	inputField.SetBorder(true)

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

	var streamsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logStreamsTable, 0, 1, true).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var streamsViewNavIdx = 0
	initViewNavigation(app, streamsView, &streamsViewNavIdx,
		[]view{
			inputField,
			logStreamsTable,
		},
	)

	return &LogStreamsView{
		LogStreamsTable: logStreamsTable,
		SearchInput:     inputField,
		RefreshStreams:  refreshLogStreamTable,
		RootView:        streamsView,
		app:             app,
	}
}

func (inst *LogStreamsView) InitInputCapture(selectedGroupName *string, searchPrefix *string) {
	inst.LogStreamsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshStreams(*selectedGroupName, searchPrefix, false)
		case tcell.KeyCtrlM:
			inst.RefreshStreams(*selectedGroupName, searchPrefix, true)
		}
		return event
	})
}

func (inst *LogStreamsView) InitSearchInputDoneCallback(selectedGroupName *string, searchPrefix *string) {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			*searchPrefix = inst.SearchInput.GetText()
			inst.RefreshStreams(*selectedGroupName, searchPrefix, false)
			inst.app.SetFocus(inst.LogStreamsTable)
		}
	})
}

type LogGroupsView struct {
	LogGroupsTable *tview.Table
	SearchInput    *tview.InputField
	RefreshGroups  func(search string)
	RootView       *tview.Flex
}

func createLogGroupsTable(
	params tableCreationParams,
	api *cloudwatchlogs.CloudWatchLogsApi,
) (*tview.Table, func(search string)) {
	var table = tview.NewTable()
	populateLogGroupsTable(table, make([]types.LogGroup, 0))

	var refreshViewsFunc = func(search string) {
		table.Clear()
		var data []types.LogGroup
		var dataChannel = make(chan []types.LogGroup)
		var resultChannel = make(chan struct{})

		go func() {
			if len(search) > 0 {
				dataChannel <- api.FilterGroupByName(search)
			} else {
				dataChannel <- api.ListLogGroups(false)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateLogGroupsTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func NewLogGroupsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsView {
	var params = tableCreationParams{app, logger}
	var logGroupsTable, refreshLogGroupsTable = createLogGroupsTable(params, api)

	var inputField = tview.NewInputField().
		SetLabel(" Search Log Groups: ").
		SetFieldWidth(64)
	inputField.SetBorder(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			refreshLogGroupsTable(inputField.GetText())
			app.SetFocus(logGroupsTable)
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var groupsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logGroupsTable, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var groupsViewNavIdx = 0
	initViewNavigation(app, groupsView, &groupsViewNavIdx,
		[]view{
			inputField,
			logGroupsTable,
		},
	)

	return &LogGroupsView{
		LogGroupsTable: logGroupsTable,
		SearchInput:    inputField,
		RefreshGroups:  refreshLogGroupsTable,
		RootView:       groupsView,
	}
}

func createLogsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	var api = cloudwatchlogs.NewCloudWatchLogsApi(config, logger)
	var logEventsView = NewLogEventsView(app, api, logger)
	var logStreamsView = NewLogStreamsView(app, api, logger)
	var logGroupsView = NewLogGroupsView(app, api, logger)

	var pages = tview.NewPages().
		AddPage("Events", logEventsView.RootView, true, true).
		AddPage("Streams", logStreamsView.RootView, true, true).
		AddAndSwitchToPage("Groups", logGroupsView.RootView, true)

	var pagesNavIdx = 0
	var orderedPages = []string{
		"Groups",
		"Streams",
		"Events",
	}

	var paginationView = createPaginatorView()
	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(paginationView.RootView, 1, 0, false)

	initPageNavigation(app, pages, &pagesNavIdx, orderedPages, paginationView.PageCounterView)

	var switchAndFocus = func(pageIdx int, view tview.Primitive) {
		pagesNavIdx = pageIdx
		pages.SwitchToPage(orderedPages[pageIdx])
		app.SetFocus(view)
	}

	var selectedGroupName = ""
	logGroupsView.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		selectedGroupName = logGroupsView.LogGroupsTable.GetCell(row, 0).Text
		logStreamsView.RefreshStreams(selectedGroupName, nil, false)
		switchAndFocus(1, logStreamsView.LogStreamsTable)
	})

	var streamName = ""
	logStreamsView.LogStreamsTable.SetSelectedFunc(func(row, column int) {
		streamName = logStreamsView.LogStreamsTable.GetCell(row, 0).Text
		logEventsView.RefreshEvents(selectedGroupName, streamName, false)
		switchAndFocus(2, logEventsView.LogEventsTable)
	})

	var searchPrefix = ""
	var searchEvent = ""
	logEventsView.InitInputCapture(&selectedGroupName, &streamName)
	logEventsView.InitSearchInputDoneCallback(&searchEvent)
	logStreamsView.InitInputCapture(&selectedGroupName, &searchPrefix)
	logStreamsView.InitSearchInputDoneCallback(&selectedGroupName, &searchPrefix)

	return rootView
}
