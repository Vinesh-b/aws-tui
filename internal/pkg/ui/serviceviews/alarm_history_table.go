package serviceviews

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AlarmHistoryTable struct {
	*core.SelectableTable[any]
	selectedAlarm string
	data          []types.AlarmHistoryItem
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudWatchAlarmsApi
}

func NewAlarmHistoryTable(
	app *tview.Application,
	api *awsapi.CloudWatchAlarmsApi,
	logger *log.Logger,
) *AlarmHistoryTable {

	var view = &AlarmHistoryTable{
		SelectableTable: core.NewSelectableTable[any](
			"Alarm History",
			core.TableRow{
				"Timestamp",
				"History",
			},
		),
		data:          nil,
		selectedAlarm: "",
		logger:        logger,
		app:           app,
		api:           api,
	}

	view.populateAlarmHistoryTable(true)
	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			view.RefreshHistory(false)
			view.app.SetFocus(view)
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
		inst.ExtendData(tableData)
		return
	}

	inst.SetData(tableData)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *AlarmHistoryTable) RefreshHistory(force bool) {
	var resultChannel = make(chan struct{})

	go func() {
		inst.data = inst.api.ListAlarmHistory(inst.selectedAlarm, force)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		inst.populateAlarmHistoryTable(force)
	})
}

func (inst *AlarmHistoryTable) SetSelectedAlarm(name string) {
	inst.selectedAlarm = name
}
