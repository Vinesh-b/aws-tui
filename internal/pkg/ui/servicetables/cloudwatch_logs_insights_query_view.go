package servicetables

import (
	"aws-tui/internal/pkg/ui/core"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/rivo/tview"
)

type InsightsQuery struct {
	query     string
	startTime time.Time
	endTime   time.Time
}

type InsightsQueryInputView struct {
	*tview.Flex
	DoneButton   *core.Button
	CancelButton *core.Button

	appCtx         *core.AppContext
	viewNavigation *core.ViewNavigation1D
	queryTextArea  *tview.TextArea
	startDateInput *core.DateTimeInputField
	endDateInput   *core.DateTimeInputField
	query          InsightsQuery
}

func NewInsightsQueryInputView(appContext *core.AppContext) *InsightsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &InsightsQueryInputView{
		Flex:         flex,
		DoneButton:   core.NewButton("Done", appContext.Theme),
		CancelButton: core.NewButton("Cancel", appContext.Theme),

		appCtx:         appContext,
		viewNavigation: core.NewViewNavigation1D(flex, nil, appContext.App),
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
				AddItem(view.DoneButton, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.CancelButton, 0, 1, false),
			1, 0, false,
		)

	view.viewNavigation.UpdateOrderedViews(
		[]core.View{
			view.queryTextArea,
			view.startDateInput,
			view.endDateInput,
			view.DoneButton,
			view.CancelButton,
		}, 0,
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

	var timeNow = time.Now()
	view.startDateInput.
		SetText(timeNow.Add(time.Duration(-3 * time.Hour)).Format(time.DateTime)).
		SetLabel("Start Time ")

	view.endDateInput.
		SetText(timeNow.Format(time.DateTime)).
		SetLabel("End Time ")

	return view
}

func (inst *InsightsQueryInputView) validateInput() error {

	return nil
}

func (inst *InsightsQueryInputView) GenerateQuery() (InsightsQuery, error) {
	var err error = nil
	var empty = InsightsQuery{}

	if inst.query.startTime, err = inst.startDateInput.ValidateInput(); err != nil {
		return empty, err
	}
	if inst.query.endTime, err = inst.endDateInput.ValidateInput(); err != nil {
		return empty, err
	}

	var queryText = strings.TrimSpace(inst.queryTextArea.GetText())
	if inst.query.query = queryText; len(queryText) > 0 {

		// Todo return error
	}

	return inst.query, err
}

type FloatingInsightsQueryInputView struct {
	*tview.Flex
	Input *InsightsQueryInputView
}

func NewFloatingInsightsQueryInputView(
	appContext *core.AppContext,
) *FloatingInsightsQueryInputView {
	var input = NewInsightsQueryInputView(appContext)

	return &FloatingInsightsQueryInputView{
		Flex:  core.FloatingView("Query", input, 0, 14),
		Input: input,
	}
}

func (inst *FloatingInsightsQueryInputView) GetLastFocusedView() tview.Primitive {
	return inst.Input.viewNavigation.GetLastFocusedView()
}
