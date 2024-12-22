package serviceviews

import (
	"fmt"
	"log"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func populateSelectedGroupsTable(table *tview.Table, data map[string]struct{}) {
	var tableData []core.TableRow
	for row := range data {
		tableData = append(tableData, core.TableRow{
			row,
		})
	}

	var title = "Selected Groups"

	core.InitSelectableTable(table, title,
		core.TableRow{
			"Name",
		},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(1, 0)
}

type LogGroupsSelectionView struct {
	LogGroupsTable     *tview.Table
	SeletedGroupsTable *tview.Table
	SearchInput        *tview.InputField
	RootView           *tview.Flex
	selectedGroups     map[string]struct{}
	app                *tview.Application
	api                *awsapi.CloudWatchLogsApi
}

func NewLogGroupsSelectionView(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsSelectionView {
	var selectedGroupsTable = tview.NewTable()
	populateSelectedGroupsTable(selectedGroupsTable, map[string]struct{}{})

	var logGroupsView = NewLogGroupsView(app, api, logger)
	logGroupsView.InitInputCapture()

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(selectedGroupsTable, 0, 1, false).
		AddItem(logGroupsView.LogGroupsTable, 0, 1, true)

	var searchabelView = core.NewSearchableView(app, logger, mainPage)
	var serviceView = core.NewServiceView(app, logger)
	serviceView.RootView = searchabelView.RootView

	serviceView.InitViewNavigation(
		[]core.View{
			logGroupsView.LogGroupsTable,
			selectedGroupsTable,
		},
	)

	return &LogGroupsSelectionView{
		SeletedGroupsTable: selectedGroupsTable,
		LogGroupsTable:     logGroupsView.LogGroupsTable,
		RootView:           serviceView.RootView,
		selectedGroups:     map[string]struct{}{},
		app:                app,
		api:                api,
	}
}

func (inst *LogGroupsSelectionView) RefreshSelectedGroups(groupName string, force bool) {
	if force {
		inst.selectedGroups = map[string]struct{}{}
	}

	var resultChannel = make(chan struct{})

	go func() {
		if len(groupName) > 0 {
			inst.selectedGroups[groupName] = struct{}{}
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.SeletedGroupsTable.Box, resultChannel, func() {
		populateSelectedGroupsTable(inst.SeletedGroupsTable, inst.selectedGroups)
	})
}

func (inst *LogGroupsSelectionView) InitInputCapture() {
	inst.LogGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var ref = inst.LogGroupsTable.GetCell(row, 0).Reference
		if ref != nil {
			var groupName = ref.(string)
			inst.RefreshSelectedGroups(groupName, false)
		}
	})

	inst.SeletedGroupsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var row, _ = inst.SeletedGroupsTable.GetSelection()
		if row == 0 || len(inst.selectedGroups) == 0 {
			return event
		}

		switch event.Rune() {
		case rune('u'):
			var groupName = inst.SeletedGroupsTable.GetCell(row, 0).Text
			delete(inst.selectedGroups, groupName)
			inst.RefreshSelectedGroups("", false)
		}
		return event
	})
}

func populateQueryResultsTable(table *tview.Table, data [][]types.ResultField, extend bool) {
	table.
		Clear().
		SetBorders(false).
		SetFixed(1, 0)
	table.
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	var tableTitle = fmt.Sprintf("Query Results (%d)", len(data))
	table.SetTitle(tableTitle)

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
			var previewText = core.ClampStringLen(&cellData, 100)
			table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(previewText).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}

	for heading, colIdx := range headingIdxMap {
		table.SetCell(0, colIdx, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(core.SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(core.ContrastBackgroundColor),
		)
	}

	if len(data) > 0 {
		table.SetSelectable(true, true).SetSelectedStyle(
			tcell.Style{}.Background(core.MoreContrastBackgroundColor),
		)
	}
	table.Select(1, 0)
	table.ScrollToBeginning()
}

type InsightsQueryResultsView struct {
	QueryResultsTable   *tview.Table
	ExpandedResult      *tview.TextArea
	QueryInput          *tview.TextArea
	QueryStartDateInput *tview.InputField
	QueryEndDateInput   *tview.InputField
	RunQueryButton      *tview.Button
	RootView            *tview.Flex
	app                 *tview.Application
	api                 *awsapi.CloudWatchLogsApi
	queryId             string
	selectedLogGroups   *[]string
	searchableView      *core.SearchableView
}

func NewInsightsQueryResultsView(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsView {
	var resultsTable = tview.NewTable()
	populateQueryResultsTable(resultsTable, make([][]types.ResultField, 0), false)

	var queryInputView = core.CreateTextArea("Query")
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

	var serviceView = core.NewServiceView(app, logger)
	serviceView.InitViewTabNavigation(queryRunView, []core.View{
		startDateInput,
		endDateInput,
		runQueryButton,
	})

	var queryView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(queryInputView, 0, 1, false).
		AddItem(queryRunView, 34, 0, false)

	var expandedResultView = core.CreateExpandedLogView(app, resultsTable, -1, core.DATA_TYPE_STRING)

	const expandedLogsSize = 5
	const resultsTableSize = 10
	const queryViewSize = 9

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandedResultView, 0, expandedLogsSize, false).
		AddItem(resultsTable, 0, resultsTableSize, true).
		AddItem(queryView, queryViewSize, 0, true)

	var searchabelView = core.NewSearchableView(app, logger, mainPage)
	serviceView.RootView = searchabelView.RootView

	serviceView.SetResizableViews(
		expandedResultView, resultsTable,
		expandedLogsSize, resultsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
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
		RootView:            serviceView.RootView,
		searchableView:      searchabelView,
		app:                 app,
		api:                 api,
		queryId:             "",
	}
}

func (inst *InsightsQueryResultsView) RefreshResults(queryId string) {
	var data [][]types.ResultField
	var resultChannel = make(chan struct{})

	go func() {
		var results [][]types.ResultField
		var status types.QueryStatus
		for range 10 {
			results, status = inst.api.GetInightsQueryResults(queryId)
			if status == types.QueryStatusRunning || status == types.QueryStatusScheduled {
				time.Sleep(2 * time.Second)
			} else {
				break
			}
		}

		data = results
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.QueryResultsTable.Box, resultChannel, func() {
		populateQueryResultsTable(inst.QueryResultsTable, data, false) // update accoring query status
	})
}

func (inst *InsightsQueryResultsView) InitInputCapture() {
	inst.searchableView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.searchableView.SetText("")
			core.HighlightTableSearch(inst.QueryResultsTable, "", []int{})
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

func CreateLogsInsightsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0xBB00DD))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchLogsApi(config, logger)
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

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDWATCH_LOGS_INSIGHTS), pages, orderedPages).Init()

	var logGroups []string
	groupSelectionView.SeletedGroupsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		logGroups = nil
		for r := range groupSelectionView.SeletedGroupsTable.GetRowCount() {
			var group = groupSelectionView.SeletedGroupsTable.GetCell(r+1, 0).Text
			if len(group) > 0 {
				logGroups = append(logGroups, group)
			}
		}

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

		logEventsView.RefreshEvents(logGroup, logStream, true)
		serviceRootView.ChangePage(2, logEventsView.LogEventsTable)
	})

	return serviceRootView.RootView
}
