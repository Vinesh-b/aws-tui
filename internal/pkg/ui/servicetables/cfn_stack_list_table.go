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
	data          []types.StackSummary
	filtered      []types.StackSummary
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

	view.populateStacksTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *StackListTable) populateStacksTable(data []types.StackSummary) {
	var tableData []core.TableRow
	for _, row := range data {
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

	inst.SetData(tableData, nil, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *StackListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
			name,
			inst.data,
			func(v types.StackSummary) string {
				return aws.ToString(v.StackName)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStacksTable(inst.filtered)
	})
}

func (inst *StackListTable) RefreshStacks(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.api.ListStacks(reset)
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
		inst.populateStacksTable(inst.data)
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
