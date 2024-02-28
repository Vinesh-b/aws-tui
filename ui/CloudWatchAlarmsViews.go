package ui

import (
	"fmt"
	"log"
	"time"

	"aws-tui/cloudwatch"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

func populateAlarmsTable(table *tview.Table, data map[string]types.MetricAlarm) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.AlarmName,
			string(row.StateValue),
		})
	}

	initSelectableTable(table, "Alarms",
		tableRow{
			"Name",
			"State",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateAlarmDetailsGrid(grid *tview.Grid, data *types.MetricAlarm) {
	grid.
		Clear().
		SetRows(1, 2, 1, 3, 1, 1, 1, 1, 1, 1, 0).
		SetColumns(18, 0)
	grid.
		SetTitle("Alarm Details").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	var tableData []tableRow
	if data != nil {
		tableData = []tableRow{
			{"Name", aws.ToString(data.AlarmName)},
			{"Description", aws.ToString(data.AlarmDescription)},
			{"State", string(data.StateValue)},
			{"StateReason", aws.ToString(data.StateReason)},
			{"MetricName", aws.ToString(data.MetricName)},
			{"MetricNamespace", aws.ToString(data.Namespace)},
			{"Period", fmt.Sprintf("%d", aws.ToInt32(data.Period))},
			{"Threshold", fmt.Sprintf("%.2f", aws.ToFloat64(data.Threshold))},
			{"DataPoints", fmt.Sprintf("%d", aws.ToInt32(data.DatapointsToAlarm))},
		}
	}

	for idx, row := range tableData {
		grid.AddItem(
			tview.NewTextView().
				SetWrap(false).
				SetText(row[0]).
				SetTextColor(tertiaryTextColor),
			idx, 0, 1, 1, 0, 0, false,
		)
		grid.AddItem(
			tview.NewTextView().
				SetWrap(true).
				SetText(row[1]).
				SetTextColor(tertiaryTextColor),
			idx, 1, 1, 1, 0, 0, false,
		)
	}
}

func populateAlarmHistoryTable(table *tview.Table, data []types.AlarmHistoryItem) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			row.Timestamp.Format(time.DateTime),
			*row.HistorySummary,
		})
	}

	initSelectableTable(table, "Alarm History",
		tableRow{
			"Timestamp",
			"History",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func createAlarmsTable(
	params tableCreationParams,
	api *cloudwatch.CloudWatchAlarmsApi,
) (*tview.Table, func(search string)) {
	var table = tview.NewTable()
	populateAlarmsTable(table, make(map[string]types.MetricAlarm, 0))

	var refreshViewsFunc = func(search string) {
		var data map[string]types.MetricAlarm
		var dataChannel = make(chan map[string]types.MetricAlarm)
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
	populateAlarmHistoryTable(table, make([]types.AlarmHistoryItem, 0))

	var refreshViewsFunc = func(name string) {
		var data []types.AlarmHistoryItem
		var dataChannel = make(chan []types.AlarmHistoryItem)
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
		var data map[string]types.MetricAlarm
		var dataChannel = make(chan map[string]types.MetricAlarm)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListAlarms(false)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			var details *types.MetricAlarm = nil
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

	var inputField = createSearchInput("Alarms")
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
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0x660000))
	defer resetGlobalStyle()

	var api = cloudwatch.NewCloudWatchAlarmsApi(config, logger)
	var alarmsDetailsView = NewAlarmsDetailsView(app, api, logger)

	var pages = tview.NewPages().
		AddAndSwitchToPage("Alarms", alarmsDetailsView.RootView, true)

	var pagesNavIdx = 0
	var orderedPages = []string{
		"Alarms",
	}

	var paginationView = createPaginatorView()
	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(paginationView.RootView, 1, 0, false)

	initPageNavigation(app, pages, &pagesNavIdx, orderedPages, paginationView.PageCounterView)

	return rootView
}
