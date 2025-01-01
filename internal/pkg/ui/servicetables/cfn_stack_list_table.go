package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StackListTable struct {
	*core.SelectableTable[any]
	selectedStack string
	data          map[string]types.StackSummary
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudFormationApi
}

func NewStackListTable(
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *StackListTable {

	var view = &StackListTable{
		SelectableTable: core.NewSelectableTable[any](
			"Stacks",
			core.TableRow{
				"StackName",
				"Status",
				"LastUpdated",
			},
		),
		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	view.populateStacksTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *StackListTable) populateStacksTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		var lastUpdated = "-"
		if row.LastUpdatedTime != nil {
			lastUpdated = row.LastUpdatedTime.Format(time.DateTime)
		}
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.StackName),
			string(row.StackStatus),
			lastUpdated,
		})
	}

	inst.SetData(tableData)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *StackListTable) RefreshStacks(reset bool) {
	var searchText = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(searchText) > 0 {
			inst.data = inst.api.FilterByName(searchText)
		} else {
			var err error = nil
			inst.data, err = inst.api.ListStacks(reset)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStacksTable()
	})
}

func (inst *StackListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedStack = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *StackListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshStacks(true)
		}
		return capture(event)
	})
}

func (inst *StackListTable) GetSelectedStackName() string {
	return inst.selectedStack
}
