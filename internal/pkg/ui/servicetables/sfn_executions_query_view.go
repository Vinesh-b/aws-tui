package servicetables

import (
	"aws-tui/internal/pkg/ui/core"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SfnExecutionsQuery struct {
	executionArn string
	status       string
	startTime    time.Time
	endTime      time.Time
}

type SfnExecutionsQueryInputView struct {
	*tview.Flex
	DoneButton   *core.Button
	CancelButton *core.Button

	logger            *log.Logger
	app               *tview.Application
	viewNavigation    *core.ViewNavigation1D
	statusDropDown    *core.DropDown
	executionArnInput *core.InputField
	startDateInput    *core.DateTimeInputField
	endDateInput      *core.DateTimeInputField
	query             SfnExecutionsQuery
}

func NewSfnExecutionsQueryInputView(app *tview.Application, logger *log.Logger) *SfnExecutionsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &SfnExecutionsQueryInputView{
		Flex:         flex,
		DoneButton:   core.NewButton("Done"),
		CancelButton: core.NewButton("Cancel"),

		logger:            logger,
		app:               app,
		viewNavigation:    core.NewViewNavigation1D(flex, nil, app),
		statusDropDown:    core.NewDropDown(),
		executionArnInput: core.NewInputField(),
		startDateInput:    core.NewDateTimeInputField(),
		endDateInput:      core.NewDateTimeInputField(),
	}

	var separator = tview.NewBox()

	view.
		AddItem(view.statusDropDown, 0, 1, true).
		AddItem(view.executionArnInput, 0, 1, false).
		AddItem(view.startDateInput, 0, 1, false).
		AddItem(view.endDateInput, 0, 1, false).
		AddItem(separator, 1, 0, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.DoneButton, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.CancelButton, 0, 1, false),
			1, 0, false,
		)

	view.viewNavigation.UpdateOrderedViews(
		[]core.View{
			view.statusDropDown,
			view.executionArnInput,
			view.startDateInput,
			view.endDateInput,
			view.DoneButton,
			view.CancelButton,
		}, 0,
	)

	view.executionArnInput.
		SetLabel("Execution Id ")

	var timeNow = time.Now()
	view.startDateInput.
		SetText(timeNow.Add(time.Duration(-3 * time.Hour)).Format(time.DateTime)).
		SetLabel("Start Time   ")

	view.endDateInput.
		SetText(timeNow.Format(time.DateTime)).
		SetLabel("End Time     ")

	view.statusDropDown.
		SetLabel("Status       ").
		AddOption("ALL", func() { view.query.status = "ALL" }).
		SetCurrentOption(0)

	for _, v := range types.ExecutionStatus.Values(types.ExecutionStatusFailed) {
		var opt = string(v)
		view.statusDropDown.AddOption(opt, func() { view.query.status = opt })
	}

	return view
}

func (inst *SfnExecutionsQueryInputView) SetDefaultTimes(startTime time.Time, endTime time.Time) {
	inst.startDateInput.SetTextTime(startTime)
	inst.endDateInput.SetTextTime(endTime)
}

func (inst *SfnExecutionsQueryInputView) GenerateQuery() (SfnExecutionsQuery, error) {
	var err error = nil
	var empty = SfnExecutionsQuery{}

	if inst.query.startTime, err = inst.startDateInput.ValidateInput(); err != nil {
		return empty, err
	}
	if inst.query.endTime, err = inst.endDateInput.ValidateInput(); err != nil {
		return empty, err
	}

	inst.query.executionArn = strings.TrimSpace(inst.executionArnInput.GetText())

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
	var floatingQuery = core.FloatingView("Query", queryView, 55, 8)

	var queryPageId = "QUERY"

	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(queryPageId, floatingQuery, true, false)

	var view = &SfnExecutionsQuerySearchView{
		Pages:    pages,
		MainPage: mainPage,

		queryView:       queryView,
		queryViewHidden: true,
		app:             app,
	}

	view.queryView.CancelButton.SetSelectedFunc(func() {
		pages.HidePage(QUERY_PAGE_NAME)
		view.queryViewHidden = true
	})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Escape:
			if view.queryViewHidden == false {
				view.HidePage(queryPageId)
				view.queryViewHidden = true
				return nil
			}
		case core.APP_KEY_BINDINGS.Find:
			if view.queryViewHidden {
				view.ShowPage(queryPageId)
				var last = view.queryView.viewNavigation.GetLastFocusedView()
				view.app.SetFocus(last)
			} else {
				view.HidePage(queryPageId)
			}
			view.queryViewHidden = !view.queryViewHidden
			return nil
		}
		return event
	})

	return view
}
