package servicetables

import (
	"aws-tui/internal/pkg/ui/core"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const dateTimelayout = "2006-01-02 15:04:05"

type SfnExecutionsQuery struct {
	status    string
	startTime time.Time
	endTime   time.Time
}

type SfnExecutionsQueryInputView struct {
	*tview.Flex
	DoneButton   *tview.Button
	CancelButton *tview.Button

	logger         *log.Logger
	app            *tview.Application
	viewNavigation *core.ViewNavigation
	statusDropDown *tview.DropDown
	startDateInput *tview.InputField
	endDateInput   *tview.InputField
	query          SfnExecutionsQuery
}

func NewSfnExecutionsQueryInputView(app *tview.Application, logger *log.Logger) *SfnExecutionsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &SfnExecutionsQueryInputView{
		Flex:         flex,
		DoneButton:   tview.NewButton("Done"),
		CancelButton: tview.NewButton("Cancel"),

		logger:         logger,
		app:            app,
		viewNavigation: core.NewViewNavigation(flex, nil, app),
		statusDropDown: tview.NewDropDown(),
		startDateInput: tview.NewInputField(),
		endDateInput:   tview.NewInputField(),
	}

	var separator = tview.NewBox()

	view.
		AddItem(view.statusDropDown, 0, 1, true).
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
			view.statusDropDown,
		}, 0,
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

	view.statusDropDown.SetListStyles(
		tcell.Style{}.
			Foreground(core.TextColour).
			Background(core.ContrastBackgroundColor),
		tcell.Style{}.
			Foreground(core.MoreContrastBackgroundColor).
			Background(core.TextColour),
	)

	view.statusDropDown.AddOption("ALL", func() { view.query.status = "ALL" })
	view.statusDropDown.SetCurrentOption(0)

	for _, v := range types.ExecutionStatus.Values(types.ExecutionStatusFailed) {
		var opt = string(v)
		view.statusDropDown.AddOption(opt, func() { view.query.status = opt })
	}

	return view
}

func (inst *SfnExecutionsQueryInputView) SetDefaultTimes(startTime time.Time, endTime time.Time) {
	inst.startDateInput.SetText(startTime.Format(dateTimelayout))
	inst.endDateInput.SetText(endTime.Format(dateTimelayout))
}

func (inst *SfnExecutionsQueryInputView) GenerateQuery() (SfnExecutionsQuery, error) {
	var err error = nil
	var empty = SfnExecutionsQuery{}

	var layout = "2006-01-02 15:04:05"
	if inst.query.startTime, err = time.Parse(layout, inst.startDateInput.GetText()); err != nil {
		return empty, err
	}
	if inst.query.endTime, err = time.Parse(layout, inst.endDateInput.GetText()); err != nil {
		return empty, err
	}

	return inst.query, err
}

type SfnExecutionsQuerySearchView struct {
	*tview.Pages
	MainPage tview.Primitive

	queryView       *SfnExecutionsQueryInputView
	app             *tview.Application
	logger          *log.Logger
	queryViewHidden bool
}

func NewSfnExecutionsQuerySearchView(
	mainPage tview.Primitive,
	app *tview.Application,
	logger *log.Logger,
) *SfnExecutionsQuerySearchView {
	var queryView = NewSfnExecutionsQueryInputView(app, logger)
	var floatingQuery = core.FloatingView("Query", queryView, 65, 5)

	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage("QUERY", floatingQuery, true, false)

	var view = &SfnExecutionsQuerySearchView{
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
