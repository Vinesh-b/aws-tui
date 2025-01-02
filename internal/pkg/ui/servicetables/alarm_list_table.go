package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AlarmListTable struct {
	*core.SelectableTable[any]
	selectedAlarm string
	data          []types.MetricAlarm
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudWatchAlarmsApi
}

func NewAlarmListTable(
	app *tview.Application,
	api *awsapi.CloudWatchAlarmsApi,
	logger *log.Logger,
) *AlarmListTable {

	var view = &AlarmListTable{
		SelectableTable: core.NewSelectableTable[any](
			"Alarms",
			core.TableRow{
				"Name",
				"State",
			},
		),
		data:          nil,
		selectedAlarm: "",
		logger:        logger,
		app:           app,
		api:           api,
	}

	view.populateAlarmsTable()
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			view.RefreshAlarms(true)
		case tcell.KeyCtrlN:
			view.RefreshAlarms(false)
		}
		return event
	})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			view.RefreshAlarms(false)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.RefreshAlarms(false)
	})

	return view
}

func (inst *AlarmListTable) populateAlarmsTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.AlarmName),
			string(row.StateValue),
		})
	}

	inst.SetData(tableData)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *AlarmListTable) RefreshAlarms(force bool) {
	var search = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterByName(search)
		} else {
			var err error = nil
			inst.data, err = inst.api.ListAlarms(force)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateAlarmsTable()
	})
}

func (inst *AlarmListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedAlarm = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *AlarmListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedAlarm = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *AlarmListTable) GetSelectedAlarm() string {
	return inst.selectedAlarm
}
