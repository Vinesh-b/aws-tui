package ui

import (
	"encoding/json"
	"log"

	"aws-tui/cloudwatch"
	"aws-tui/cloudwatchlogs"

	"github.com/aws/aws-sdk-go-v2/aws"
	cw_types "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	cwl_types "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

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

func createLogItemsTable(
	params tableCreationParams,
	api *cloudwatchlogs.CloudWatchLogsApi,
) (*tview.Table, func(groupName string, streamName string, extend bool)) {
	var table = tview.NewTable()
	populateLogEventsTable(table, make([]cwl_types.OutputLogEvent, 0), false)

	var refreshViewsFunc = func(groupName string, streamName string, extend bool) {
		var data []cwl_types.OutputLogEvent
		var dataChannel = make(chan []cwl_types.OutputLogEvent)
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

	var expandedLogsView = tview.NewTextArea()
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
	populateLogStreamsTable(table, make([]cwl_types.LogStream, 0), false)

	var refreshViewsFunc = func(groupName string, searchPrefix *string, extend bool) {
		var data []cwl_types.LogStream
		var dataChannel = make(chan []cwl_types.LogStream)
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
	populateLogGroupsTable(table, make([]cwl_types.LogGroup, 0))

	var refreshViewsFunc = func(search string) {
		table.Clear()
		var data []cwl_types.LogGroup
		var dataChannel = make(chan []cwl_types.LogGroup)
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
) *tview.Pages {
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

	initPageNavigation(app, pages, &pagesNavIdx, orderedPages)

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

	return pages
}

// ---- Alarms view ------------------------------------------------------------

type AlarmsDetailsView struct {
	AlarmsTable    *tview.Table
	HistoryTable   *tview.Table
	DetailsGrid    *tview.Grid
	SearchInput    *tview.InputField
	RefreshAlarms  func(search string)
	RefreshHistory func(alarmName string)
	RefreshDetails func(alarmName string)
	RootView       *tview.Flex
}

func createAlarmsTable(
	params tableCreationParams,
	api *cloudwatch.CloudWatchAlarmsApi,
) (*tview.Table, func(search string)) {
	var table = tview.NewTable()
	populateAlarmsTable(table, make(map[string]cw_types.MetricAlarm, 0))

	var refreshViewsFunc = func(search string) {
		var data map[string]cw_types.MetricAlarm
		var dataChannel = make(chan map[string]cw_types.MetricAlarm)
		var resultChannel = make(chan struct{})

		go func() {
			if len(search) > 0 {
				dataChannel <- api.FilterByName(search)
			} else {
				dataChannel <- api.ListAlarms(false)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateAlarmsTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func createAlarmHistoryTable(
	params tableCreationParams,
	api *cloudwatch.CloudWatchAlarmsApi,
) (*tview.Table, func(name string)) {
	var table = tview.NewTable()
	populateAlarmHistoryTable(table, make([]cw_types.AlarmHistoryItem, 0))

	var refreshViewsFunc = func(name string) {
		var data []cw_types.AlarmHistoryItem
		var dataChannel = make(chan []cw_types.AlarmHistoryItem)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListAlarmHistory(name)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateAlarmHistoryTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func createAlarmDetailsGrid(
	params tableCreationParams,
	api *cloudwatch.CloudWatchAlarmsApi,
) (*tview.Grid, func(alarmName string)) {
	var table = tview.NewGrid()
	populateAlarmDetailsGrid(table, nil)

	var refreshViewsFunc = func(alarmName string) {
		var data map[string]cw_types.MetricAlarm
		var dataChannel = make(chan map[string]cw_types.MetricAlarm)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListAlarms(false)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			var details *cw_types.MetricAlarm = nil
			var val, ok = data[alarmName]
			if ok {
				details = &val
			}
			populateAlarmDetailsGrid(table, details)
		})
	}

	return table, refreshViewsFunc
}

func NewAlarmsDetailsView(
	app *tview.Application,
	api *cloudwatch.CloudWatchAlarmsApi,
	logger *log.Logger,
) *AlarmsDetailsView {
	var (
		params = tableCreationParams{app, logger}

		alarmsTable, refreshAlarmsTable   = createAlarmsTable(params, api)
		alarmDetails, refreshAlarmDetails = createAlarmDetailsGrid(params, api)
		alarmHistory, refreshAlarmHistory = createAlarmHistoryTable(params, api)
	)

	alarmsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var name = alarmsTable.GetCell(row, 0).Text
		go refreshAlarmDetails(name)
		go refreshAlarmHistory(name)
	})

	var inputField = tview.NewInputField().
		SetLabel(" Search Alarms: ").
		SetFieldWidth(64)
	inputField.SetBorder(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			go refreshAlarmsTable(inputField.GetText())
			app.SetFocus(alarmsTable)
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var flexHomeView = tview.NewFlex().SetDirection(tview.FlexRow)
	flexHomeView.AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(alarmDetails, 0, 3000, false).
		AddItem(alarmHistory, 0, 2500, false).
		AddItem(alarmsTable, 0, 4500, false),
		0, 4000, false,
	)

	// Keep at bottom
	flexHomeView.AddItem(tview.NewFlex().
		AddItem(inputField, 0, 1, true),
		3, 0, true,
	)

	var startIdx = 0
	initViewNavigation(app, flexHomeView, &startIdx,
		[]view{
			inputField,
			alarmsTable,
			alarmHistory,
			alarmDetails,
		},
	)

	return &AlarmsDetailsView{
		AlarmsTable:    alarmsTable,
		DetailsGrid:    alarmDetails,
		HistoryTable:   alarmHistory,
		SearchInput:    inputField,
		RefreshAlarms:  refreshAlarmsTable,
		RefreshDetails: refreshAlarmDetails,
		RefreshHistory: refreshAlarmHistory,
		RootView:       flexHomeView,
	}

}

func createAlarmsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) *tview.Pages {
	var api = cloudwatch.NewCloudWatchAlarmsApi(config, logger)
	var alarmsDetailsView = NewAlarmsDetailsView(app, api, logger)

	var pages = tview.NewPages().
		AddAndSwitchToPage("Alarms", alarmsDetailsView.RootView, true)

	var pagesNavIdx = 0
	initPageNavigation(app, pages, &pagesNavIdx,
		[]string{
			"Alarms",
		},
	)

	return pages
}
