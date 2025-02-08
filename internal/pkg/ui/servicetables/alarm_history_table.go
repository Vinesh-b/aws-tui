package servicetables

import (
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
)

type AlarmHistoryTable struct {
	*core.SelectableTable[any]
	selectedAlarm string
	data          []types.AlarmHistoryItem
	serviceCtx    *core.ServiceContext[awsapi.CloudWatchAlarmsApi]
}

func NewAlarmHistoryTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchAlarmsApi],
) *AlarmHistoryTable {

	var view = &AlarmHistoryTable{
		SelectableTable: core.NewSelectableTable[any](
			"Alarm History",
			core.TableRow{
				"Timestamp",
				"History",
			},
			serviceContext.AppContext,
		),
		data:          nil,
		selectedAlarm: "",
		serviceCtx:    serviceContext,
	}

	view.populateAlarmHistoryTable(true)
	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			view.RefreshHistory(false)
		}
	})
	return view
}

func (inst *AlarmHistoryTable) populateAlarmHistoryTable(reset bool) {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			row.Timestamp.Format(time.DateTime),
			aws.ToString(row.HistorySummary),
		})
	}

	if !reset {
		inst.ExtendData(tableData, nil)
		return
	}

	inst.SetData(tableData, nil, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.ScrollToBeginning()
}

func (inst *AlarmHistoryTable) RefreshHistory(force bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.data, err = inst.serviceCtx.Api.ListAlarmHistory(inst.selectedAlarm, force)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateAlarmHistoryTable(force)
	})
}

func (inst *AlarmHistoryTable) SetSelectedAlarm(name string) {
	inst.selectedAlarm = name
}
