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

type InsightsQueryInputView struct {
	*tview.Flex
	RunButton  *core.Button
	StopButton *core.Button

	appCtx         *core.AppContext
	queryTextArea  *tview.TextArea
	startDateInput *core.DateTimeInputField
	endDateInput   *core.DateTimeInputField
}

func NewInsightsQueryInputView(appContext *core.AppContext) *InsightsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &InsightsQueryInputView{
		Flex:       flex,
		RunButton:  core.NewButton("Run", appContext.Theme),
		StopButton: core.NewButton("Stop", appContext.Theme),

		appCtx:         appContext,
		queryTextArea:  tview.NewTextArea(),
		startDateInput: core.NewDateTimeInputField(appContext.Theme),
		endDateInput:   core.NewDateTimeInputField(appContext.Theme),
	}

	var separator = tview.NewBox()

	view.
		AddItem(view.queryTextArea, 0, 1, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.startDateInput, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.endDateInput, 0, 1, false),
			1, 0, false,
		).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.RunButton, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.StopButton, 0, 1, false),
			1, 0, false,
		)

	view.queryTextArea.SetText(
		"fields @timestamp, @message, @log\n"+
			"| sort @timestamp desc\n"+
			"| limit 1000\n",
		false,
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

	var timeNow = time.Now()
	view.startDateInput.
		SetText(timeNow.Add(time.Duration(-3 * time.Hour)).Format(time.DateTime)).
		SetLabel("Start Time ")

	view.endDateInput.
		SetText(timeNow.Format(time.DateTime)).
		SetLabel("End Time ")

	return view
}

// NavigationRows are the views to move between in the order they are laid out
func (inst *InsightsQueryInputView) NavigationRows() [][]core.View {
	return [][]core.View{
		{inst.queryTextArea},
		{inst.startDateInput, inst.endDateInput},
		{inst.RunButton, inst.StopButton},
	}
}

// GenerateQuery is the query as it is entered, which is empty with an error if
// it can't be run
func (inst *InsightsQueryInputView) GenerateQuery() (InsightsQuery, error) {
	var empty = InsightsQuery{}
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
