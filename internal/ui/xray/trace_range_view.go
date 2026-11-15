package xray

import (
	"time"

	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// timeInputLayout is how the times are entered, which the inputs show when
// they are empty
const timeInputLayout = "2006-01-02 15:04:05"

// TraceRangeView picks the time range of the traces of the table: a range that
// ends now, which loads them as it is picked, or the start and end times that
// are entered, which load them on Enter. Entering a time moves to Custom.
type TraceRangeView struct {
	*tview.Flex
	// ErrorMessageCallback reports times that can't be used
	ErrorMessageCallback func(text string, a ...any)

	table         *TracesTable
	rangeDropDown *core.DropDown
	startInput    *core.DateTimeInputField
	endInput      *core.DateTimeInputField
	// settingRange is set while the range of the table is shown, which is not
	// a range being picked or times being entered
	settingRange bool
}

func NewTraceRangeView(table *TracesTable, appCtx *core.AppContext) *TraceRangeView {
	var view = &TraceRangeView{
		Flex:                 tview.NewFlex().SetDirection(tview.FlexColumn),
		ErrorMessageCallback: func(text string, a ...any) {},
		table:                table,
		rangeDropDown:        core.NewDropDown(appCtx.Theme),
		startInput:           core.NewDateTimeInputField(appCtx.Theme),
		endInput:             core.NewDateTimeInputField(appCtx.Theme),
	}

	var separator = tview.NewBox()
	view.
		AddItem(view.rangeDropDown, 0, 22, false).
		AddItem(separator, 1, 0, false).
		AddItem(view.startInput, 0, 26, false).
		AddItem(separator, 1, 0, false).
		AddItem(view.endInput, 0, 24, false)

	view.rangeDropDown.SetLabel("Range ")
	for _, option := range traceRanges {
		view.rangeDropDown.AddOption(option.label, nil)
	}
	view.startInput.SetLabel("Start ")
	view.endInput.SetLabel("End ")

	view.rangeDropDown.SetSelectedFunc(func(text string, index int) {
		if !view.settingRange {
			table.SetTimeRange(index)
		}
	})

	var enteredTime = func(string) {
		if !view.settingRange {
			view.settingRange = true
			defer func() { view.settingRange = false }()
			view.rangeDropDown.SetCurrentOption(customTraceRangeIdx)
		}
	}
	var done = func(key tcell.Key) {
		if key == core.APP_KEY_BINDINGS.Done {
			view.ApplyCustomRange()
		}
	}
	for _, input := range []*core.DateTimeInputField{view.startInput, view.endInput} {
		input.SetChangedFunc(enteredTime)
		input.SetDoneFunc(done)
	}

	table.SetRangeChangedFunc(view.showRange)

	return view
}

// showRange shows the range of the table and its times
func (inst *TraceRangeView) showRange(rangeIdx int, start, end time.Time) {
	inst.settingRange = true
	defer func() { inst.settingRange = false }()

	inst.rangeDropDown.SetCurrentOption(rangeIdx)
	inst.startInput.SetTextTime(start)
	inst.endInput.SetTextTime(end)
}

// ApplyCustomRange loads the traces between the times that are entered
func (inst *TraceRangeView) ApplyCustomRange() {
	var start, startErr = inst.startInput.ValidateInput()
	if startErr != nil {
		inst.ErrorMessageCallback("The start time is not a time like %s", timeInputLayout)
		return
	}
	var end, endErr = inst.endInput.ValidateInput()
	if endErr != nil {
		inst.ErrorMessageCallback("The end time is not a time like %s", timeInputLayout)
		return
	}

	if err := inst.table.SetCustomRange(start, end); err != nil {
		inst.ErrorMessageCallback("%s", err.Error())
	}
}

// NavigationRow is the views to move between, in the order they are laid out
func (inst *TraceRangeView) NavigationRow() []core.View {
	return []core.View{inst.rangeDropDown, inst.startInput, inst.endInput}
}
