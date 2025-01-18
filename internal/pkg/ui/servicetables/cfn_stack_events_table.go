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
	*core.SelectableTable[types.StackEvent]
	selectedStack     types.StackEvent
	selectedStackName string
	data              []types.StackEvent
	logger            *log.Logger
	app               *tview.Application
	api               *awsapi.CloudFormationApi
}

func NewStackEventsTable(
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *StackEventsTable {

	var view = &StackEventsTable{
		SelectableTable: core.NewSelectableTable[types.StackEvent](
			"Stacks Events",
			core.TableRow{
				"Timestamp",
				"LogicalId",
				"ResourceType",
				"Status",
				"Reason",
			},
			app,
		),
		data:              nil,
		selectedStack:     types.StackEvent{},
		selectedStackName: "",
		logger:            logger,
		app:               app,
		api:               api,
	}

	view.HighlightSearch = true
	view.populateStackEventsTable(true)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *StackEventsTable) populateStackEventsTable(reset bool) {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			row.Timestamp.Format("2006-01-02 15:04:05.000"),
			aws.ToString(row.LogicalResourceId),
			aws.ToString(row.ResourceType),
			string(row.ResourceStatus),
			aws.ToString(row.ResourceStatusReason),
		})
	}
	if !reset {
		inst.ExtendData(tableData, inst.data)
		return
	}

	inst.SetData(tableData, inst.data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *StackEventsTable) RefreshEvents(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(inst.selectedStackName) > 0 {
			var err error = nil
			inst.data, err = inst.api.DescribeStackEvents(inst.selectedStackName, reset)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		} else {
			inst.data = make([]types.StackEvent, 0)
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
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
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshEvents(true)
		}
		return capture(event)
	})
}

func (inst *StackEventsTable) SetSelectedStackName(name string) {
	inst.selectedStackName = name
}

func (inst *StackEventsTable) GetResourceStatusReason(row int) string {
	return aws.ToString(inst.GetPrivateData(row, 0).ResourceStatusReason)
}
