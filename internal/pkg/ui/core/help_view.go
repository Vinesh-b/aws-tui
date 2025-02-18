package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.Flex
	table *tview.Table
}

func NewHelpView(appCtx *AppContext) *HelpView {
	var table = tview.NewTable()
	table.
		SetSelectable(true, false).
		SetSelectedStyle(
			tcell.Style{}.Background(appCtx.Theme.MoreContrastBackgroundColor),
		)

	SetTableHeading(table, appCtx.Theme, "Shortcut", 0)
	SetTableHeading(table, appCtx.Theme, "Description", 1)
	table.GetCell(0, 1).SetExpansion(1)

	table.SetSelectedFunc(func(row, column int) {
		if handlerPtr := GetPrivateData[*func()](table.GetCell(row, 0)); handlerPtr != nil {
			(*handlerPtr)()
		}
	})

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
	if handler == nil {
		handler = func() {}
	}
	inst.table.SetCell(rows, 0, NewTableCell(shortcut, &handler))
	inst.table.SetCell(rows, 1, NewTableCell[*func()](descirption, nil))

	return inst
}

type FloatingHelpView struct {
	*tview.Flex
	View *HelpView
}

func NewFloatingHelpView(appCtx *AppContext) *FloatingHelpView {
	var helpView = NewHelpView(appCtx)
	return &FloatingHelpView{
		Flex: FloatingViewRelative("Available actions", helpView, 50, 70),
		View: helpView,
	}
}

func (inst *FloatingHelpView) GetLastFocusedView() tview.Primitive {
	return inst
}
