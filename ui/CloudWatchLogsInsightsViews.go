package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"aws-tui/cloudwatchlogs"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func populateSelectedGroupsTable(table *tview.Table, data []string, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			row,
		})
	}

	var title = "Selected Groups"
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

type LogGroupsSelectionView struct {
	LogGroupsTable     *tview.Table
	SeletedGroupsTable *tview.Table
	SearchInput        *tview.InputField
	RootView           *tview.Flex
	app                *tview.Application
	api                *cloudwatchlogs.CloudWatchLogsApi
}

func NewLogGroupsSelectionView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsSelectionView {
	var selectedGroupsTable = tview.NewTable()
	populateSelectedGroupsTable(selectedGroupsTable, make([]string, 0), false)

	var logGroupsView = NewLogGroupsView(app, api, logger)
	logGroupsView.InitInputCapture()

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(selectedGroupsTable, 0, 1, false).
		AddItem(logGroupsView.LogGroupsTable, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(logGroupsView.SearchInput, 0, 1, true),
			3, 0, true,
		)

	serviceView.InitViewNavigation(
		[]view{
			logGroupsView.SearchInput,
			logGroupsView.LogGroupsTable,
			selectedGroupsTable,
		},
	)

	return &LogGroupsSelectionView{
		SeletedGroupsTable: selectedGroupsTable,
		LogGroupsTable:     logGroupsView.LogGroupsTable,
		SearchInput:        logGroupsView.SearchInput,
		RootView:           serviceView.RootView,
		app:                app,
		api:                api,
	}
}

func (inst *LogGroupsSelectionView) RefreshSelectedGroups(groupName string, force bool) {
	var data []string
	var dataChannel = make(chan []string)
	var resultChannel = make(chan struct{})

	go func() {
		dataChannel <- []string{groupName}
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.SeletedGroupsTable.Box, resultChannel, func() {
		populateSelectedGroupsTable(inst.SeletedGroupsTable, data, !force)
	})
}

func (inst *LogGroupsSelectionView) InitInputCapture() {
	inst.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var groupName = inst.LogGroupsTable.GetCell(row, 0).Reference.(string)
		inst.RefreshSelectedGroups(groupName, false)
	})

}

func populateQueryResultsTable(table *tview.Table, data [][]types.ResultField, extend bool) {
	table.
		Clear().
		SetBorders(false).
		SetFixed(1, 0)
	table.
		SetTitle("Query Results").
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	var headingIdx = 0
	var headingIdxMap = make(map[string]int)
	for rowIdx, rowData := range data {
		for _, resField := range rowData {
			var colIdx, ok = headingIdxMap[*resField.Field]
			if !ok {
				headingIdxMap[*resField.Field] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellData = fmt.Sprintf("%s", aws.ToString(resField.Value))
			var previewText = clampStringLen(&cellData, 100)
			table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(previewText).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}

	for heading, colIdx := range headingIdxMap {
		table.SetCell(0, colIdx, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(secondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(contrastBackgroundColor),
		)
	}

	if len(data) > 0 {
		table.SetSelectable(true, true).SetSelectedStyle(
			tcell.Style{}.Background(moreContrastBackgroundColor),
		)
	}
	table.Select(0, 0)
	table.ScrollToBeginning()
}

type InsightsQueryResultsView struct {
	QueryResultsTable   *tview.Table
	ExpandedResult      *tview.TextArea
	QueryInput          *tview.TextArea
	QueryStartDateInput *tview.InputField
	QueryEndDateInput   *tview.InputField
	RunQueryButton      *tview.Button
	SearchInput         *tview.InputField
	RootView            *tview.Flex
	app                 *tview.Application
	api                 *cloudwatchlogs.CloudWatchLogsApi
	queryId             string
	selectedLogGroups   *[]string
}

func NewInsightsQueryResultsView(
	app *tview.Application,
	api *cloudwatchlogs.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsView {
	var resultsTable = tview.NewTable()
	populateQueryResultsTable(resultsTable, make([][]types.ResultField, 0), false)

	var queryInputView = createTextArea("Query")
	queryInputView.SetText(
		"fields @timestamp, @message, @log\n| sort @timestamp desc\n| limit 1000",
		true,
	)

	var runQueryButton = tview.NewButton("Run Query")
	var startDateInput = tview.NewInputField().SetFieldWidth(20).SetLabel("Start Date ")
	var endDateInput = tview.NewInputField().SetFieldWidth(20).SetLabel("End Date   ")
	var timeNow = time.Now()
	startDateInput.SetText(timeNow.Add(time.Duration(-time.Hour)).Format(time.DateTime))
	endDateInput.SetText(timeNow.Format(time.DateTime))

	var queryRunView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(startDateInput, 1, 0, false).
		AddItem(endDateInput, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(runQueryButton, 1, 0, false)
	queryRunView.SetBorder(true)

	var serviceView = NewServiceView(app)
	serviceView.InitViewTabNavigation(queryRunView, []view{
		startDateInput,
		endDateInput,
		runQueryButton,
	})

	var queryView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(queryInputView, 0, 1, false).
		AddItem(queryRunView, 34, 0, false)

	var expandedResultView = createExpandedLogView(app, resultsTable, -1)

	var inputField = createSearchInput("Search Results")

	const expandedLogsSize = 5
	const resultsTableSize = 10
	const queryViewSize = 9

	serviceView.RootView.
		AddItem(expandedResultView, 0, expandedLogsSize, false).
		AddItem(resultsTable, 0, resultsTableSize, true).
		AddItem(queryView, queryViewSize, 0, true).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		expandedResultView, resultsTable,
		expandedLogsSize, resultsTableSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			queryRunView,
			queryInputView,
			resultsTable,
			expandedResultView,
		},
	)

	return &InsightsQueryResultsView{
		QueryResultsTable:   resultsTable,
		QueryInput:          queryInputView,
		ExpandedResult:      expandedResultView,
		QueryStartDateInput: startDateInput,
		QueryEndDateInput:   endDateInput,
		RunQueryButton:      runQueryButton,
		SearchInput:         inputField,
		RootView:            serviceView.RootView,
		app:                 app,
		api:                 api,
		queryId:             "",
	}
}

func (inst *InsightsQueryResultsView) RefreshResults(queryId string) {
	var data [][]types.ResultField
	var dataChannel = make(chan [][]types.ResultField)
	var resultChannel = make(chan struct{})

	go func() {
		var result, _ = inst.api.GetInightsQueryResults(queryId)
		dataChannel <- result
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.QueryResultsTable.Box, resultChannel, func() {
		populateQueryResultsTable(inst.QueryResultsTable, data, false) // update accoring query status
	})
}

func (inst *InsightsQueryResultsView) InitInputCapture() {
	inst.SearchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.SearchInput.SetText("")
			highlightTableSearch(inst.app, inst.QueryResultsTable, "", []int{})
		}
		return event
	})

	inst.QueryResultsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshResults(inst.queryId)
		}

		return event
	})

	inst.RunQueryButton.SetSelectedFunc(func() {
		var layout = "2006-01-02 15:04:05"
		var startTime, err_1 = time.Parse(layout, inst.QueryStartDateInput.GetText())
		var endTime, err_2 = time.Parse(layout, inst.QueryEndDateInput.GetText())

		if err_1 != nil || err_2 != nil {
			return
		}

		var queryIdChan = make(chan string, 1)
		go func() {
			queryIdChan <- inst.api.StartInightsQuery(
				*inst.selectedLogGroups,
				startTime,
				endTime,
				inst.QueryInput.GetText(),
			)
		}()

		go func() {
			inst.queryId = <-queryIdChan
			inst.RefreshResults(inst.queryId)
		}()

	})
}

func (inst *InsightsQueryResultsView) InitSearchInputBuffer(selectedGroups *[]string) {
	inst.selectedLogGroups = selectedGroups
}

func createLogsInsightsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer resetGlobalStyle()

	var api = cloudwatchlogs.NewCloudWatchLogsApi(config, logger)
	var insightsResultsView = NewInsightsQueryResultsView(app, api, logger)
	var groupSelectionView = NewLogGroupsSelectionView(app, api, logger)
	var logEventsView = NewLogEventsView(app, api, logger)

	var pages = tview.NewPages().
		AddPage("LogEvents", logEventsView.RootView, true, true).
		AddPage("Query", insightsResultsView.RootView, true, true).
		AddAndSwitchToPage("GroupsSelection", groupSelectionView.RootView, true)

	var orderedPages = []string{
		"GroupsSelection",
		"Query",
		"LogEvents",
	}

	var serviceRootView = NewServiceRootView(
		app, string(CLOUDWATCH_LOGS_INSIGHTS), pages, orderedPages).Init()

	var logGroups []string
	groupSelectionView.SeletedGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		for r := range groupSelectionView.SeletedGroupsTable.GetRowCount() {
			var group = groupSelectionView.SeletedGroupsTable.GetCell(r+1, 0).Text
			if len(group) > 0 {
				logGroups = append(logGroups, group)
			}
		}

		logger.Println(logGroups)

		serviceRootView.ChangePage(1, insightsResultsView.QueryInput)
	})

	groupSelectionView.InitInputCapture()
	insightsResultsView.InitInputCapture()
	insightsResultsView.InitSearchInputBuffer(&logGroups)

	var recordPtr = ""
	insightsResultsView.QueryResultsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var lastCol = insightsResultsView.QueryResultsTable.GetColumnCount() - 1
		recordPtr = insightsResultsView.QueryResultsTable.GetCell(row, lastCol).Reference.(string)

		var record = api.GetInsightsLogRecord(recordPtr)

		var logStream = record["@logStream"]
		var _, logGroup, _ = strings.Cut(record["@log"], ":")

		logEventsView.RefreshEvents(logGroup, logStream, false)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	return serviceRootView.RootView
}
