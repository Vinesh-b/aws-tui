package cloudwatch

import (
	"errors"
	"strings"
	"time"

	"aws-tui/internal/ui/core"

	"github.com/atotto/clipboard"
	"github.com/rivo/tview"
)

type InsightsQuery struct {
	Query     string
	StartTime time.Time
	EndTime   time.Time
}

// insightsRange is a time range to pick, ending now. Custom has no span, and
// leaves the times as they are entered.
type insightsRange struct {
	label string
	span  time.Duration
}

var insightsRanges = []insightsRange{
	{"Last 15 minutes", 15 * time.Minute},
	{"Last hour", time.Hour},
	{"Last 3 hours", 3 * time.Hour},
	{"Last 24 hours", 24 * time.Hour},
	{"Last 7 days", 7 * 24 * time.Hour},
	{"Custom", 0},
}

const (
	defaultInsightsRange = 2
	customInsightsRange  = 5
)

const defaultInsightsQuery = "fields @timestamp, @message, @log\n" +
	"| sort @timestamp desc\n" +
	"| limit 1000\n"

type InsightsQueryInputView struct {
	*tview.Flex
	RunButton  *core.Button
	StopButton *core.Button

	appCtx         *core.AppContext
	queryTextArea  *tview.TextArea
	rangeDropDown  *core.DropDown
	startDateInput *core.DateTimeInputField
	endDateInput   *core.DateTimeInputField
	// settingRange is set while the times are put in by a range, which is not
	// them being entered
	settingRange bool
}

func NewInsightsQueryInputView(appContext *core.AppContext) *InsightsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &InsightsQueryInputView{
		Flex:       flex,
		RunButton:  core.NewButton("Run", appContext.Theme),
		StopButton: core.NewButton("Stop", appContext.Theme),

		appCtx:         appContext,
		queryTextArea:  tview.NewTextArea(),
		rangeDropDown:  core.NewDropDown(appContext.Theme),
		startDateInput: core.NewDateTimeInputField(appContext.Theme),
		endDateInput:   core.NewDateTimeInputField(appContext.Theme),
	}

	var separator = tview.NewBox()

	view.
		AddItem(view.queryTextArea, 0, 1, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.rangeDropDown, 0, 22, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.startDateInput, 0, 26, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.endDateInput, 0, 24, false),
			1, 0, false,
		).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.RunButton, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.StopButton, 0, 1, false),
			1, 0, false,
		)

	view.queryTextArea.SetClipboard(
		func(s string) { clipboard.WriteAll(s) },
		func() string {
			var res, _ = clipboard.ReadAll()
			return res
		},
	)
	view.queryTextArea.SetSelectedStyle(appContext.Theme.GetFocusFormItemStyle())
	view.queryTextArea.
		SetTitle("Query").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	view.startDateInput.SetLabel("Start ")
	view.endDateInput.SetLabel("End ")
	view.initRanges()
	view.queryTextArea.SetText(defaultInsightsQuery, false)

	return view
}

// initRanges adds the ranges to pick from, which put their times in as they are
// picked and when the query is run. Entering a time moves to Custom.
func (inst *InsightsQueryInputView) initRanges() {
	inst.rangeDropDown.SetLabel("Range ")
	for _, option := range insightsRanges {
		inst.rangeDropDown.AddOption(option.label, nil)
	}

	var enteredTime = func(string) {
		if !inst.settingRange {
			inst.rangeDropDown.SetCurrentOption(customInsightsRange)
		}
	}
	inst.startDateInput.SetChangedFunc(enteredTime)
	inst.endDateInput.SetChangedFunc(enteredTime)

	inst.rangeDropDown.SetSelectedFunc(func(text string, index int) {
		inst.applyRange(index, time.Now())
	})
	inst.rangeDropDown.SetCurrentOption(defaultInsightsRange)
}

// applyRange puts the times of the range ending at now in. Custom leaves the
// times that were entered.
func (inst *InsightsQueryInputView) applyRange(index int, now time.Time) {
	var span = insightsRanges[index].span
	if span == 0 {
		return
	}

	inst.settingRange = true
	defer func() { inst.settingRange = false }()
	inst.startDateInput.SetTextTime(now.Add(-span))
	inst.endDateInput.SetTextTime(now)
}

func (inst *InsightsQueryInputView) rangeIndex() int {
	var index, _ = inst.rangeDropDown.GetCurrentOption()
	return index
}

// NavigationRows are the views to move between in the order they are laid out
func (inst *InsightsQueryInputView) NavigationRows() [][]core.View {
	return [][]core.View{
		{inst.queryTextArea},
		{inst.rangeDropDown, inst.startDateInput, inst.endDateInput},
		{inst.RunButton, inst.StopButton},
	}
}

// GenerateQuery is the query as it is entered, which is empty with an error if
// it can't be run
func (inst *InsightsQueryInputView) GenerateQuery() (InsightsQuery, error) {
	var empty = InsightsQuery{}
	// A range that is picked ends when the query is run, not when it was picked
	inst.applyRange(inst.rangeIndex(), time.Now())
	var query = InsightsQuery{Query: strings.TrimSpace(inst.queryTextArea.GetText())}

	var err error
	if query.StartTime, err = inst.startDateInput.ValidateInput(); err != nil {
		return empty, err
	}
	if query.EndTime, err = inst.endDateInput.ValidateInput(); err != nil {
		return empty, err
	}

	if len(query.Query) == 0 {
		return empty, errors.New("The query is empty")
	}
	if query.EndTime.Before(query.StartTime) {
		return empty, errors.New("The end time is before the start time")
	}

	return query, nil
}
