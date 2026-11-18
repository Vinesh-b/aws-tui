package xray

import (
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

// traceFilterPlaceholder shows how a filter expression is written
const traceFilterPlaceholder = `X-Ray filter expression, e.g. service("orders") { fault } AND responsetime > 2`

// TraceFilterView is the X-Ray filter expression the traces of the table are
// found with, which loads them on Enter. An empty filter finds all of them.
type TraceFilterView struct {
	*core.InputField
	table *TracesTable
}

func NewTraceFilterView(table *TracesTable, appCtx *core.AppContext) *TraceFilterView {
	var view = &TraceFilterView{
		InputField: core.NewInputField(appCtx.Theme),
		table:      table,
	}
	view.SetLabel("Filter ")
	view.SetPlaceholder(traceFilterPlaceholder)
	view.SetDoneFunc(func(key tcell.Key) {
		if key == core.APP_KEY_BINDINGS.Done {
			view.ApplyFilter()
		}
	})
	return view
}

// ApplyFilter loads the traces that match the filter that is entered
func (inst *TraceFilterView) ApplyFilter() {
	inst.table.SetFilterExpression(inst.GetText())
}
