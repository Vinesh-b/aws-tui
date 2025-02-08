package servicetables

import (
	"aws-tui/internal/pkg/ui/core"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
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

	appCtx            *core.AppContext
	viewNavigation    *core.ViewNavigation1D
	statusDropDown    *core.DropDown
	executionArnInput *core.InputField
	startDateInput    *core.DateTimeInputField
	endDateInput      *core.DateTimeInputField
	query             SfnExecutionsQuery
}

func NewSfnExecutionsQueryInputView(appContext *core.AppContext) *SfnExecutionsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &SfnExecutionsQueryInputView{
		Flex:         flex,
		DoneButton:   core.NewButton("Done"),
		CancelButton: core.NewButton("Cancel"),

		appCtx:            appContext,
		viewNavigation:    core.NewViewNavigation1D(flex, nil, appContext.App),
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

type FloatingSfnExecutionsQueryInputView struct {
	*tview.Flex
	Input *SfnExecutionsQueryInputView
}

func NewFloatingSfnExecutionsQueryInputView(
	appContext *core.AppContext,
) *FloatingSfnExecutionsQueryInputView {
	var queryView = NewSfnExecutionsQueryInputView(appContext)
	return &FloatingSfnExecutionsQueryInputView{
		Flex:  core.FloatingView("Query", queryView, 55, 8),
		Input: queryView,
	}
}

func (inst *FloatingSfnExecutionsQueryInputView) GetLastFocusedView() tview.Primitive {
	return inst.Input.viewNavigation.GetLastFocusedView()
}
