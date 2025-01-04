package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const logNameCol = 0

type SelectedGroupsTable struct {
	*core.SelectableTable[string]
	data          core.StringSet
	selectedGroup string
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudWatchLogsApi
}

func NewSelectedGroupsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *SelectedGroupsTable {

	var view = &SelectedGroupsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Selected Groups",
			core.TableRow{
				"Name",
			},
		),
		data:          core.StringSet{},
		selectedGroup: "",
		logger:        logger,
		app:           app,
		api:           api,
	}

	view.HighlightSearch = true
	view.populateSelectedGroupsTable()
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			view.data = core.StringSet{}
			view.RefreshSelectedGroups()
		}
		switch event.Rune() {
		case rune('u'):
			var groupName = view.GetSelectedLogGroup()
			logger.Printf("Removing: %v", groupName)
			view.RemoveLogGroup(groupName)
			view.RefreshSelectedGroups()
		}
		return event
	})

	return view
}

func (inst *SelectedGroupsTable) populateSelectedGroupsTable() {
	var tableData []core.TableRow
	var privateData []string
	for row := range inst.data {
		tableData = append(tableData, core.TableRow{
			row,
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, logNameCol)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *SelectedGroupsTable) RefreshSelectedGroups() {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateSelectedGroupsTable()
	})

}

func (inst *SelectedGroupsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		var ref = inst.GetCell(row, logNameCol).Reference
		if row < 1 || ref == nil {
			return
		}
		inst.selectedGroup = ref.(string)
		handler(row, column)
	})
}

func (inst *SelectedGroupsTable) AddLogGroup(groupName string) {
	if len(groupName) > 0 {
		inst.data[groupName] = struct{}{}
	}
}

func (inst *SelectedGroupsTable) RemoveLogGroup(groupName string) {
	delete(inst.data, groupName)
}

func (inst *SelectedGroupsTable) GetSelectedLogGroup() string {
	return inst.selectedGroup
}

func (inst *SelectedGroupsTable) GetAllLogGroups() []string {
	var allGroups = []string{}
	for group := range inst.data {
		allGroups = append(allGroups, group)
	}
	return allGroups
}
