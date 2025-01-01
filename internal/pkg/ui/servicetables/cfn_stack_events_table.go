package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StackEventsTable struct {
	*core.SelectableTable[string]
	selectedStack string
	data          []types.StackEvent
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudFormationApi
}

func NewStackEventsTable(
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *StackEventsTable {

	var view = &StackEventsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Stacks Events",
			core.TableRow{
				"Timestamp",
				"LogicalId",
				"ResourceType",
				"Status",
				"Reason",
			},
		),
		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	view.HighlightSearch = true
	view.populateStackEventsTable(true)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *StackEventsTable) populateStackEventsTable(reset bool) {
	var tableData []core.TableRow
	var privateData []string
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			row.Timestamp.Format("2006-01-02 15:04:05.000"),
			aws.ToString(row.LogicalResourceId),
			aws.ToString(row.ResourceType),
			string(row.ResourceStatus),
			aws.ToString(row.ResourceStatusReason),
		})
		privateData = append(privateData, aws.ToString(row.ResourceStatusReason))
	}
	if !reset {
		inst.ExtendData(tableData)
		inst.ExtendPrivateData(privateData)
		return
	}

	inst.SetData(tableData)
	inst.SetPrivateData(privateData, 4)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *StackEventsTable) RefreshEvents(reset bool) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(inst.selectedStack) > 0 {
			var err error = nil
			inst.data, err = inst.api.DescribeStackEvents(inst.selectedStack, reset)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		} else {
			inst.data = make([]types.StackEvent, 0)
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		inst.populateStackEventsTable(reset)
	})
}

func (inst *StackEventsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		handler(row, column)
	})
}

func (inst *StackEventsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshEvents(true)
		}
		return capture(event)
	})
}

func (inst *StackEventsTable) SetSelectedStackName(name string) {
	inst.selectedStack = name
}

func (inst *StackEventsTable) GetResourceStatusReason(row int) string {
	var reason = inst.GetCell(row, 4).Reference
	if row < 1 || reason == nil {
		return ""
	}
	return reason.(string)
}
