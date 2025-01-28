package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.Flex
	table *tview.Table
}

func NewHelpView() *HelpView {
	var table = tview.NewTable()
	table.
		SetSelectable(true, false).
		SetSelectedStyle(
			tcell.Style{}.Background(MoreContrastBackgroundColor),
		)

	SetTableHeading(table, "Shortcut", 0)
	SetTableHeading(table, "Description", 1)
	table.GetCell(0, 1).SetExpansion(1)

	var flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true)

	return &HelpView{
		Flex:  flex,
		table: table,
	}
}

func (inst *HelpView) AddItem(shortcut string, descirption string, handler func()) *HelpView {
	var rows = inst.table.GetRowCount()
	inst.table.SetCell(rows, 0, NewTableCell[any](shortcut, nil))
	inst.table.SetCell(rows, 1, NewTableCell[any](descirption, nil))

	return inst
}
