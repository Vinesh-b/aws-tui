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

func populateAlarmHistoryTable(table *tview.Table, data []types.AlarmHistoryItem, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			row.Timestamp.Format(time.DateTime),
			*row.HistorySummary,
		})
	}

	var title = "Alarm History"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
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

type AlarmsDetailsView struct {
	AlarmsTable  *tview.Table
	HistoryTable *tview.Table
	DetailsGrid  *tview.Grid
	SearchInput  *tview.InputField
	RootView     *tview.Flex
	app          *tview.Application
	api          *cloudwatch.CloudWatchAlarmsApi
}

func NewAlarmsDetailsView(
	app *tview.Application,
	api *cloudwatch.CloudWatchAlarmsApi,
	logger *log.Logger,
) *AlarmsDetailsView {
	var alarmsTable = tview.NewTable()
	populateAlarmsTable(alarmsTable, make(map[string]types.MetricAlarm, 0))

	var alarmHistory = tview.NewTable()
	populateAlarmHistoryTable(alarmHistory, make([]types.AlarmHistoryItem, 0), false)

	var alarmDetails = tview.NewGrid()
	populateAlarmDetailsGrid(alarmDetails, nil)

	var inputField = createSearchInput("Alarms")

	const alarmsTableSize = 3500
	const alarmHistorySize = 3000

	var serviceView = NewServiceView(app, logger)
	serviceView.RootView.
		AddItem(alarmDetails, 14, 0, false).
		AddItem(alarmHistory, 0, alarmHistorySize, false).
		AddItem(alarmsTable, 0, alarmsTableSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		alarmHistory, alarmsTable,
		alarmHistorySize, alarmsTableSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			alarmsTable,
			alarmHistory,
			alarmDetails,
		},
	)

	return &AlarmsDetailsView{
		AlarmsTable:  alarmsTable,
		DetailsGrid:  alarmDetails,
		HistoryTable: alarmHistory,
		SearchInput:  inputField,
		RootView:     serviceView.RootView,
		app:          app,
		api:          api,
	}

}

func (inst *AlarmsDetailsView) RefreshAlarms(search string, force bool) {
	var data map[string]types.MetricAlarm
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			data = inst.api.FilterByName(search)
		} else {
			data = inst.api.ListAlarms(force)
		}
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.AlarmsTable.Box, resultChannel, func() {
		populateAlarmsTable(inst.AlarmsTable, data)
	})
}

func (inst *AlarmsDetailsView) RefreshHistory(alarmName string, force bool) {
	var data []types.AlarmHistoryItem
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListAlarmHistory(alarmName, force)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.HistoryTable.Box, resultChannel, func() {
		populateAlarmHistoryTable(inst.HistoryTable, data, !force)
	})
}

func (inst *AlarmsDetailsView) RefreshDetails(alarmName string) {
	var data map[string]types.MetricAlarm
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListAlarms(false)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.DetailsGrid.Box, resultChannel, func() {
		var details *types.MetricAlarm = nil
		var val, ok = data[alarmName]
		if ok {
			details = &val
		}
		populateAlarmDetailsGrid(inst.DetailsGrid, details)
	})
}

func (inst *AlarmsDetailsView) InitInputCapture() {
	var refreshDetails = func(row int) {
		if row < 1 {
			return
		}
		var name = inst.AlarmsTable.GetCell(row, 0).Text
		inst.RefreshDetails(name)
		inst.RefreshHistory(name, true)
	}

	inst.AlarmsTable.SetSelectedFunc(func(row, column int) {
		refreshDetails(row)
	})

	inst.AlarmsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshAlarms(inst.SearchInput.GetText(), true)
			var row, _ = inst.AlarmsTable.GetSelection()
			refreshDetails(row)
		}
		return event
	})

	inst.HistoryTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			var row, _ = inst.AlarmsTable.GetSelection()
			refreshDetails(row)
		case tcell.KeyCtrlN:
			inst.RefreshHistory("", false)
		}
		return event
	})

	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshAlarms(inst.SearchInput.GetText(), false)
			inst.app.SetFocus(inst.AlarmsTable)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})
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
	alarmsDetailsView.InitInputCapture()

	var pages = tview.NewPages().
		AddAndSwitchToPage("Alarms", alarmsDetailsView.RootView, true)

	var orderedPages = []string{
		"Alarms",
	}

	var serviceRootView = NewServiceRootView(
		app, string(CLOUDWATCH_ALARMS), pages, orderedPages).Init()

	return serviceRootView.RootView
}
