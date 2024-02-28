package ui

import (
	"log"

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
