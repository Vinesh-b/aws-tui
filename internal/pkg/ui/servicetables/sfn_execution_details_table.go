package servicetables

import (
	"log"
	"slices"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachineExecutionDetailsTable struct {
	*core.SelectableTable[StateDetails]
	ExecutionHistory     *sfn.GetExecutionHistoryOutput
	selectedExecutionArn string
	selectedState        StateDetails
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.StateMachineApi
}

func NewStateMachineExecutionDetailsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionDetailsTable {

	var view = &StateMachineExecutionDetailsTable{
		SelectableTable: core.NewSelectableTable[StateDetails](
			"Execution Details",
			core.TableRow{
				"Name",
				"Type",
				"Status",
				"Duration",
			},
			app,
		),
		ExecutionHistory:     nil,
		selectedExecutionArn: "",
		selectedState:        StateDetails{},

		logger: logger,
		app:    app,
		api:    api,
	}

	view.populateTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshExecutionDetails(view.selectedExecutionArn, true)
		}
		return event
	})

	return view
}

type StateDetails struct {
	Id        int64
	Name      string
	Type      string
	Status    string
	Input     string
	Output    string
	StartTime time.Time
	EndTime   time.Time
}

func (inst *StateMachineExecutionDetailsTable) populateTable() {
	var tableData []core.TableRow
	var enteredEventTypes = []types.HistoryEventType{
		types.HistoryEventTypeTaskStateEntered,
		types.HistoryEventTypePassStateEntered,
		types.HistoryEventTypeParallelStateEntered,
		types.HistoryEventTypeFailStateEntered,
		types.HistoryEventTypeSucceedStateEntered,
		types.HistoryEventTypeMapStateEntered,
		types.HistoryEventTypeChoiceStateEntered,
	}

	var exitedEventTypes = []types.HistoryEventType{
		types.HistoryEventTypeTaskStateExited,
		types.HistoryEventTypePassStateExited,
		types.HistoryEventTypeParallelStateExited,
		types.HistoryEventTypeMapStateExited,
		types.HistoryEventTypeChoiceStateExited,
		types.HistoryEventTypeSucceedStateExited,
	}

	var results = []StateDetails{}

	if inst.ExecutionHistory != nil {
		for _, row := range inst.ExecutionHistory.Events {
			if slices.Contains(enteredEventTypes, row.Type) {
				results = append(results, StateDetails{
					Id:        row.Id,
					Name:      aws.ToString(row.StateEnteredEventDetails.Name),
					Type:      strings.Replace(string(row.Type), "Entered", "", 1),
					Status:    "Entered",
					Input:     aws.ToString(row.StateEnteredEventDetails.Input),
					Output:    "",
					StartTime: aws.ToTime(row.Timestamp),
					EndTime:   aws.ToTime(row.Timestamp),
				})
			}

			if slices.Contains(exitedEventTypes, row.Type) {
				var idx = slices.IndexFunc(results, func(d StateDetails) bool {
					return d.Name == aws.ToString(row.StateExitedEventDetails.Name)
				})

				if idx > -1 {
					results[idx].Status = "Succeeded"
					results[idx].Output = aws.ToString(row.StateExitedEventDetails.Output)
					results[idx].EndTime = aws.ToTime(row.Timestamp)
				}
			}
		}
	}

	for _, row := range results {
		tableData = append(tableData, core.TableRow{
			row.Name,
			row.Type,
			row.Status,
			row.EndTime.Sub(row.StartTime).String(),
		})
	}

	inst.SetData(tableData, results, 0)
}

func (inst *StateMachineExecutionDetailsTable) RefreshExecutionDetails(executionArn string, force bool) {
	inst.selectedExecutionArn = executionArn
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.ExecutionHistory, err = inst.api.GetExecutionHistory(executionArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable()
	})
}

func (inst *StateMachineExecutionDetailsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedState = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *StateMachineExecutionDetailsTable) GetSelectedStepInput() string {
	return inst.selectedState.Input
}

func (inst *StateMachineExecutionDetailsTable) GetSelectedStepOutput() string {
	return inst.selectedState.Output
}
