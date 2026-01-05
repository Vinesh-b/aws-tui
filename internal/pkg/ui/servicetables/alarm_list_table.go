package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"aws-tui/internal/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
)

type AlarmListTable struct {
	*core.SelectableTable[types.MetricAlarm]
	selectedAlarm types.MetricAlarm
	data          []types.MetricAlarm
	filtered      []types.MetricAlarm
	serviceCtx    *core.ServiceContext[awsapi.CloudWatchAlarmsApi]
}

func NewAlarmListTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchAlarmsApi],
) *AlarmListTable {

	var view = &AlarmListTable{
		SelectableTable: core.NewSelectableTable[types.MetricAlarm](
			"Alarms",
			core.TableRow{
				"Name",
				"State",
			},
			serviceContext.AppContext,
		),
		data:          nil,
		selectedAlarm: types.MetricAlarm{},
		serviceCtx:    serviceContext,
	}

	view.populateAlarmsTable(view.data)
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshAlarms(true)
			return nil
		case core.APP_KEY_BINDINGS.LoadMoreData:
			view.RefreshAlarms(false)
			return nil
		}
		return event
	})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var search = view.GetSearchText()
			view.FilterbyName(search)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterbyName(text)
	})

	return view
}

func (inst *AlarmListTable) populateAlarmsTable(data []types.MetricAlarm) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.AlarmName),
			string(row.StateValue),
		})
	}

	inst.SetData(tableData, data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.ScrollToBeginning()
}

func (inst *AlarmListTable) FilterbyName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(
			name,
			inst.data,
			func(a types.MetricAlarm) string {
				return aws.ToString(a.AlarmName)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateAlarmsTable(inst.filtered)
	})
}

func (inst *AlarmListTable) RefreshAlarms(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListAlarms(reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateAlarmsTable(inst.data)
	})
}

func (inst *AlarmListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedAlarm = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *AlarmListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedAlarm = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *AlarmListTable) GetSelectedAlarm() types.MetricAlarm {
	return inst.selectedAlarm
}

func (inst *AlarmListTable) GetSelectedAlarmName() string {
	return aws.ToString(inst.selectedAlarm.AlarmName)
}
