package servicetables

import (
	"fmt"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/rivo/tview"
)

type AlarmDetailsTable struct {
	*tview.Grid
	ErrorMessageHandler func(text string)
	selectedAlarm       string
	data                *types.MetricAlarm
	logger              *log.Logger
	app                 *tview.Application
	api                 *awsapi.CloudWatchAlarmsApi
}

func NewAlarmDetailsTable(
	app *tview.Application,
	api *awsapi.CloudWatchAlarmsApi,
	logger *log.Logger,
) *AlarmDetailsTable {
	var view = &AlarmDetailsTable{
		Grid:                tview.NewGrid(),
		ErrorMessageHandler: func(text string) {},
		data:                nil,
		selectedAlarm:       "",
		logger:              logger,
		app:                 app,
		api:                 api,
	}
	view.
		Clear().
		SetRows(1, 2, 1, 3, 1, 1, 1, 1, 1, 1, 0).
		SetColumns(18, 0)
	view.
		SetTitle("Alarm Details").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	view.populateAlarmDetailsGrid()
	return view
}

func (inst *AlarmDetailsTable) populateAlarmDetailsGrid() {
	var tableData []core.TableRow
	var data = inst.data
	if data != nil {
		tableData = []core.TableRow{
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

	inst.SetTitle("Alarm Details")

	for idx, row := range tableData {
		inst.AddItem(
			tview.NewTextView().
				SetWrap(false).
				SetText(row[0]).
				SetTextColor(core.TertiaryTextColor),
			idx, 0, 1, 1, 0, 0, false,
		)
		inst.AddItem(
			tview.NewTextView().
				SetWrap(true).
				SetText(row[1]).
				SetTextColor(core.TertiaryTextColor),
			idx, 1, 1, 1, 0, 0, false,
		)
	}
}

func (inst *AlarmDetailsTable) RefreshDetails() {
	var data map[string]types.MetricAlarm
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(inst.selectedAlarm) > 0 {
			var err error = nil
			data, err = inst.api.ListAlarms(false)
			if err != nil {
				inst.ErrorMessageHandler(err.Error())
			}
			inst.data = &types.MetricAlarm{}
			var val, ok = data[inst.selectedAlarm]
			if ok {
				inst.data = &val
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateAlarmDetailsGrid()
	})
}

func (inst *AlarmDetailsTable) SetSelectedAlarm(name string) {
	inst.selectedAlarm = name
}
