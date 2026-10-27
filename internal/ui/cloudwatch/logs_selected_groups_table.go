package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

const logNameCol = 0

// SelectedGroupsTable lists the log groups that are marked to be queried, and
// unmarks them
type SelectedGroupsTable struct {
	*core.SelectableTable[string]
	marks         *LogGroupMarks
	selectedGroup string
	serviceCtx    *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewSelectedGroupsTable(
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
	marks *LogGroupMarks,
) *SelectedGroupsTable {

	var view = &SelectedGroupsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Marked Groups",
			core.TableRow{
				"Name",
			},
			serviceViewCtx.AppContext,
		),
		marks:         marks,
		selectedGroup: "",
		serviceCtx:    serviceViewCtx,
	}

	view.HighlightSearch = true
	view.populateSelectedGroupsTable()
	marks.OnChange(view.populateSelectedGroupsTable)
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.marks.Clear()
			return nil
		case core.APP_KEY_BINDINGS.RemoveItem:
			var groupName = view.GetSelectedLogGroup()
			serviceViewCtx.Logger.Printf("Removing: %v", groupName)
			view.marks.Remove(groupName)
			return nil
		}
		return event
	})

	return view
}

func (inst *SelectedGroupsTable) populateSelectedGroupsTable() {
	var tableData []core.TableRow
	var privateData = inst.marks.All()
	for _, group := range privateData {
		tableData = append(tableData, core.TableRow{
			group,
		})
	}

	inst.selectedGroup = ""
	inst.SetData(tableData, privateData, logNameCol)
	inst.GetCell(0, 0).SetExpansion(1)

	if len(privateData) > 0 {
		inst.selectedGroup = privateData[0]
	}
}

func (inst *SelectedGroupsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedGroup = inst.GetPrivateData(row, logNameCol)
		handler(row, column)
	})
}

func (inst *SelectedGroupsTable) GetSelectedLogGroup() string {
	return inst.selectedGroup
}
