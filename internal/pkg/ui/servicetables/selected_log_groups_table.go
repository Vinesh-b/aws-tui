package servicetables

import (
	"sort"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/gdamore/tcell/v2"
)

const logNameCol = 0

type SelectedGroupsTable struct {
	*core.SelectableTable[string]
	data          core.StringSet
	selectedGroup string
	serviceCtx    *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewSelectedGroupsTable(
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *SelectedGroupsTable {

	var view = &SelectedGroupsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Selected Groups",
			core.TableRow{
				"Name",
			},
			serviceViewCtx.AppContext,
		),
		data:          core.StringSet{},
		selectedGroup: "",
		serviceCtx:    serviceViewCtx,
	}

	view.HighlightSearch = true
	view.populateSelectedGroupsTable()
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.data = core.StringSet{}
			view.RefreshSelectedGroups()
			return nil
		case rune('u'):
			var groupName = view.GetSelectedLogGroup()
			serviceViewCtx.Logger.Printf("Removing: %v", groupName)
			view.RemoveLogGroup(groupName)
			view.RefreshSelectedGroups()
			return nil
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

	sort.Slice(privateData, func(i, j int) bool {
		return privateData[i] < privateData[j]
	})

	sort.Slice(tableData, func(i, j int) bool {
		return tableData[i][0] < tableData[j][0]
	})

	inst.SetData(tableData, privateData, logNameCol)
	inst.GetCell(0, 0).SetExpansion(1)

	if len(privateData) > 0 {
		inst.selectedGroup = privateData[0]
	}
}

func (inst *SelectedGroupsTable) RefreshSelectedGroups() {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateSelectedGroupsTable()
	})

}

func (inst *SelectedGroupsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedGroup = inst.GetPrivateData(row, logNameCol)
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
