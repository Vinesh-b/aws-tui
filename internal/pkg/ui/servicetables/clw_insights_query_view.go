package servicetables

import (
	"aws-tui/internal/pkg/ui/core"
	"log"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InsightsQuery struct {
	query     string
	startTime time.Time
	endTime   time.Time
}

type InsightsQueryInputView struct {
	*tview.Flex
	DoneButton   *tview.Button
	CancelButton *tview.Button

	logger         *log.Logger
	app            *tview.Application
	viewNavigation *core.ViewNavigation
	queryTextArea  *tview.TextArea
	startDateInput *tview.InputField
	endDateInput   *tview.InputField
	query          InsightsQuery
}

func NewInsightsQueryInputView(app *tview.Application, logger *log.Logger) *InsightsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &InsightsQueryInputView{
		Flex:         flex,
		DoneButton:   tview.NewButton("Done"),
		CancelButton: tview.NewButton("Cancel"),

		logger:         logger,
		app:            app,
		viewNavigation: core.NewViewNavigation(flex, nil, app),
		queryTextArea:  tview.NewTextArea(),
		startDateInput: tview.NewInputField(),
		endDateInput:   tview.NewInputField(),
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
			view.CancelButton,
			view.DoneButton,
			view.endDateInput,
			view.startDateInput,
			view.queryTextArea,
		}, 0,
	)

	view.queryTextArea.SetText(
		"fields @timestamp, @message, @log\n"+
			"| sort @timestamp desc\n"+
			"| limit 1000\n",
		false,
	)

	var timeNow = time.Now()
	var dateTimelayout = "2006-01-02 15:04:05"
	view.startDateInput.
		SetPlaceholder(dateTimelayout).
		SetPlaceholderTextColor(core.PlaceHolderTextColor).
		SetText(timeNow.Add(time.Duration(-3 * time.Hour)).Format(time.DateTime)).
		SetLabel("Start Time ")

	view.endDateInput.
		SetPlaceholder(dateTimelayout).
		SetPlaceholderTextColor(core.PlaceHolderTextColor).
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

	var layout = "2006-01-02 15:04:05"
	if inst.query.startTime, err = time.Parse(layout, inst.startDateInput.GetText()); err != nil {
		return empty, err
	}
	if inst.query.endTime, err = time.Parse(layout, inst.endDateInput.GetText()); err != nil {
		return empty, err
	}

	var queryText = strings.TrimSpace(inst.queryTextArea.GetText())
	if inst.query.query = queryText; len(queryText) > 0 {

		// Todo return error
	}

	return inst.query, err
}

type InsightsQuerySearchView struct {
	*tview.Pages
	MainPage tview.Primitive

	queryView       *InsightsQueryInputView
	app             *tview.Application
	logger          *log.Logger
	queryViewHidden bool
}

func NewInsightsQuerySearchView(
	mainPage tview.Primitive,
	app *tview.Application,
	logger *log.Logger,
) *InsightsQuerySearchView {
	var queryView = NewInsightsQueryInputView(app, logger)
	var floatingQuery = core.FloatingView("Query", queryView, 70, 12)

	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage("QUERY", floatingQuery, true, false)

	var view = &InsightsQuerySearchView{
		Pages:    pages,
		MainPage: mainPage,

		queryView:       queryView,
		queryViewHidden: true,
	}

	view.queryView.CancelButton.SetSelectedFunc(func() {
		pages.HidePage(QUERY_PAGE_NAME)
		view.queryViewHidden = true
	})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Find:
			if view.queryViewHidden {
				view.ShowPage("QUERY")
			} else {
				view.HidePage("QUERY")
			}
			view.queryViewHidden = !view.queryViewHidden
			return nil
		}
		return event
	})

	return view
}
