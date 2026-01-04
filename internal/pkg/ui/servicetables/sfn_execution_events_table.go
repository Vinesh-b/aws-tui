package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/gdamore/tcell/v2"
)

type SfnExecutionStateEventsTable struct {
	*core.SelectableTable[EventDetails]
	ExecutionHistory *sfn.GetExecutionHistoryOutput
	State            StateDetails
	selectedEvent    EventDetails
	appCtx           *core.AppContext
	api              *awsapi.StateMachineApi
	cwlApi           *awsapi.CloudWatchLogsApi
}

type StateDetails struct {
	Id       int64
	Name     string
	Type     SfnStateType
	Duration time.Duration
	Events   []EventDetails
}

func NewSfnExecutionStatesTable(
	appCtx *core.AppContext,
	api *awsapi.StateMachineApi,
) *SfnExecutionStateEventsTable {

	var view = &SfnExecutionStateEventsTable{
		SelectableTable: core.NewSelectableTable[EventDetails](
			"Execution State Events",
			core.TableRow{
				"StateName",
				"Type",
				"Resource",
				"Action",
				"StartTime",
				"Duration",
				"Errors",
			},
			appCtx,
		),
		ExecutionHistory: nil,
		selectedEvent:    EventDetails{},

		appCtx: appCtx,
		api:    api,
	}

	view.populateTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshExecutionState(view.State)
			return nil
		}
		return event
	})

	return view
}

func (inst *SfnExecutionStateEventsTable) populateTable() {
	var tableData []core.TableRow

	for _, row := range inst.State.Events {
		tableData = append(tableData, core.TableRow{
			row.Name,
			row.Type,
			row.ResourceType,
			row.Resource,
			row.StartTime.Format(time.DateTime),
			row.EndTime.Sub(row.StartTime).String(),
			row.Errors,
		})
	}

	inst.SetData(tableData, inst.State.Events, 0)
	inst.Select(1, 0)
}

func (inst *SfnExecutionStateEventsTable) RefreshExecutionState(state StateDetails) {
	inst.State = state
	inst.populateTable()
}

func (inst *SfnExecutionStateEventsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedEvent = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SfnExecutionStateEventsTable) GetSelectedStepInput() string {
	return inst.selectedEvent.Input
}

func (inst *SfnExecutionStateEventsTable) GetSelectedStepOutput() string {
	return inst.selectedEvent.Output
}

func (inst *SfnExecutionStateEventsTable) GetSelectedStepErrorCause() string {
	return inst.selectedEvent.Casue
}
