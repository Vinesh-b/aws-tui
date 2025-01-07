package servicetables

import (
	"log"
	"slices"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogGroupsTable struct {
	*core.SelectableTable[string]
	data             []types.LogGroup
	filtered         []types.LogGroup
	selectedLogGroup string
	logger           *log.Logger
	app              *tview.Application
	api              *awsapi.CloudWatchLogsApi
}

func NewLogGroupsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsTable {

	var view = &LogGroupsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Log Groups",
			core.TableRow{
				"Name",
			},
            app,
		),
		data:             nil,
		selectedLogGroup: "",
		logger:           logger,
		app:              app,
		api:              api,
	}

	view.populateLogGroupsTable(view.data)
	view.SetSelectedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshLogGroups(true)
		}
		return event
	})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var search = view.GetSearchText()
			view.FilterByName(search)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *LogGroupsTable) populateLogGroupsTable(data []types.LogGroup) {
	var tableData []core.TableRow
	var privateData []string

	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.LogGroupName),
		})
		privateData = append(privateData, aws.ToString(row.LogGroupName))
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.ScrollToBeginning()
}

func (inst *LogGroupsTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(name, inst.data, func(v types.LogGroup) string {
			return aws.ToString(v.LogGroupName)
		})
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogGroupsTable(inst.filtered)
	})
}

func (inst *LogGroupsTable) RefreshLogGroups(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.api.ListLogGroups(reset)
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
		inst.populateLogGroupsTable(inst.data)
	})
}

func (inst *LogGroupsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		var ref = inst.GetCell(row, 0).Reference
		if row < 1 || ref == nil {
			return
		}

		inst.selectedLogGroup = ref.(string)
		handler(row, column)
	})
}

func (inst *LogGroupsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		var ref = inst.GetCell(row, 0).Reference
		if row < 1 || ref == nil {
			return
		}

		inst.selectedLogGroup = ref.(string)
		handler(row, column)
	})
}

func (inst *LogGroupsTable) GetSeletedLogGroup() string {
	return inst.selectedLogGroup
}

func (inst *LogGroupsTable) GetLogGroupDetail() types.LogGroup {
	var idx = slices.IndexFunc(inst.data, func(d types.LogGroup) bool {
		return aws.ToString(d.LogGroupName) == inst.selectedLogGroup
	})
	if idx == -1 {
		return types.LogGroup{}
	}

	return inst.data[idx]
}
