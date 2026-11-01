package eventbridge

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const defaultScheduleGroup = "default"

// maxFlexibleWindowMinutes is the longest flexible time window the api allows
const maxFlexibleWindowMinutes = 1440

var scheduleNamePattern = regexp.MustCompile(`^[0-9a-zA-Z._-]{1,64}$`)

// scheduleFormValues are the settings of a schedule that can be typed in
type scheduleFormValues struct {
	Name        string
	Group       string
	Expression  string
	Timezone    string
	TargetArn   string
	RoleArn     string
	Input       string
	Window      string
	Description string
	Enabled     bool
}

// validScheduleValues checks the values typed for a schedule and returns them
// trimmed, with the default group filled in. The name and group are only
// checked when creating, as they can't be changed afterwards.
func validScheduleValues(values scheduleFormValues, creating bool) (scheduleFormValues, error) {
	values.Name = strings.TrimSpace(values.Name)
	values.Group = strings.TrimSpace(values.Group)
	values.Expression = strings.TrimSpace(values.Expression)
	values.Timezone = strings.TrimSpace(values.Timezone)
	values.TargetArn = strings.TrimSpace(values.TargetArn)
	values.RoleArn = strings.TrimSpace(values.RoleArn)
	values.Window = strings.TrimSpace(values.Window)
	values.Description = strings.TrimSpace(values.Description)

	if creating {
		if values.Name == "" {
			return values, fmt.Errorf("Name is empty")
		}
		if !scheduleNamePattern.MatchString(values.Name) {
			return values, fmt.Errorf("Names can have up to 64 letters, digits, dots, hyphens and underscores")
		}
		if values.Group == "" {
			values.Group = defaultScheduleGroup
		}
		if !scheduleNamePattern.MatchString(values.Group) {
			return values, fmt.Errorf("Group names can have up to 64 letters, digits, dots, hyphens and underscores")
		}
	}

	if values.Expression == "" {
		return values, fmt.Errorf("Schedule is empty")
	}
	if !isScheduleExpression(values.Expression) {
		return values, fmt.Errorf("Schedules are rate(...), cron(...) or at(...)")
	}
	if values.TargetArn == "" {
		return values, fmt.Errorf("Target ARN is empty")
	}
	if values.RoleArn == "" {
		return values, fmt.Errorf("Role ARN is empty")
	}
	if _, err := flexibleWindow(values.Window); err != nil {
		return values, err
	}
	return values, nil
}

func isScheduleExpression(expression string) bool {
	if !strings.HasSuffix(expression, ")") {
		return false
	}
	for _, prefix := range []string{"rate(", "cron(", "at("} {
		if strings.HasPrefix(expression, prefix) {
			return true
		}
	}
	return false
}

// flexibleWindow is the window for the minutes typed, which is off when they
// are left out or zero
func flexibleWindow(minutes string) (*types.FlexibleTimeWindow, error) {
	if minutes == "" || minutes == "0" {
		return &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff}, nil
	}
	var value, err = strconv.Atoi(minutes)
	if err != nil || value < 1 || value > maxFlexibleWindowMinutes {
		return nil, fmt.Errorf("Flexible window is a number of minutes from 0 to %d", maxFlexibleWindowMinutes)
	}
	return &types.FlexibleTimeWindow{
		Mode:                   types.FlexibleTimeWindowModeFlexible,
		MaximumWindowInMinutes: aws.Int32(int32(value)),
	}, nil
}

func scheduleState(enabled bool) types.ScheduleState {
	if enabled {
		return types.ScheduleStateEnabled
	}
	return types.ScheduleStateDisabled
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return aws.String(value)
}

// newCreateScheduleInput is the request creating a schedule with the values,
// which must have been validated
func newCreateScheduleInput(values scheduleFormValues) scheduler.CreateScheduleInput {
	var window, _ = flexibleWindow(values.Window)
	return scheduler.CreateScheduleInput{
		Name:                       aws.String(values.Name),
		GroupName:                  aws.String(values.Group),
		ScheduleExpression:         aws.String(values.Expression),
		ScheduleExpressionTimezone: optionalString(values.Timezone),
		Description:                optionalString(values.Description),
		FlexibleTimeWindow:         window,
		State:                      scheduleState(values.Enabled),
		Target: &types.Target{
			Arn:     aws.String(values.TargetArn),
			RoleArn: aws.String(values.RoleArn),
			Input:   optionalString(values.Input),
		},
	}
}

// newUpdateScheduleInput is the request that stores the schedule as it is.
// An update replaces every setting, so they are all copied over. The target
// is copied, so changing it leaves the schedule alone.
func newUpdateScheduleInput(schedule scheduler.GetScheduleOutput) scheduler.UpdateScheduleInput {
	var target *types.Target
	if schedule.Target != nil {
		var copied = *schedule.Target
		target = &copied
	}
	return scheduler.UpdateScheduleInput{
		Name:                       schedule.Name,
		GroupName:                  schedule.GroupName,
		ScheduleExpression:         schedule.ScheduleExpression,
		ScheduleExpressionTimezone: schedule.ScheduleExpressionTimezone,
		Description:                schedule.Description,
		FlexibleTimeWindow:         schedule.FlexibleTimeWindow,
		StartDate:                  schedule.StartDate,
		EndDate:                    schedule.EndDate,
		State:                      schedule.State,
		KmsKeyArn:                  schedule.KmsKeyArn,
		ActionAfterCompletion:      schedule.ActionAfterCompletion,
		Target:                     target,
	}
}

// newEditScheduleInput is the request that changes the schedule to the
// values, which must have been validated. Settings the form doesn't have,
// such as the retry policy of the target, are kept.
func newEditScheduleInput(schedule scheduler.GetScheduleOutput, values scheduleFormValues) scheduler.UpdateScheduleInput {
	var input = newUpdateScheduleInput(schedule)
	var window, _ = flexibleWindow(values.Window)

	input.ScheduleExpression = aws.String(values.Expression)
	input.ScheduleExpressionTimezone = optionalString(values.Timezone)
	input.Description = optionalString(values.Description)
	input.FlexibleTimeWindow = window
	input.State = scheduleState(values.Enabled)
	if input.Target == nil {
		input.Target = &types.Target{}
	}
	input.Target.Arn = aws.String(values.TargetArn)
	input.Target.RoleArn = aws.String(values.RoleArn)
	input.Target.Input = optionalString(values.Input)
	return input
}

// scheduleValuesOf are the form values that show the schedule
func scheduleValuesOf(schedule scheduler.GetScheduleOutput) scheduleFormValues {
	var values = scheduleFormValues{
		Name:        aws.ToString(schedule.Name),
		Group:       aws.ToString(schedule.GroupName),
		Expression:  aws.ToString(schedule.ScheduleExpression),
		Timezone:    aws.ToString(schedule.ScheduleExpressionTimezone),
		Description: aws.ToString(schedule.Description),
		Enabled:     schedule.State != types.ScheduleStateDisabled,
	}
	if schedule.Target != nil {
		values.TargetArn = aws.ToString(schedule.Target.Arn)
		values.RoleArn = aws.ToString(schedule.Target.RoleArn)
		values.Input = aws.ToString(schedule.Target.Input)
	}
	if window := schedule.FlexibleTimeWindow; window != nil && window.Mode == types.FlexibleTimeWindowModeFlexible {
		values.Window = strconv.Itoa(int(aws.ToInt32(window.MaximumWindowInMinutes)))
	}
	return values
}

const (
	scheduleStateOptionEnabled  = "Enabled"
	scheduleStateOptionDisabled = "Disabled"
)

// ScheduleFormView is the form for creating a schedule, or for editing one,
// which has no name and group fields as those can't be changed.
type ScheduleFormView struct {
	*tview.Flex
	creating         bool
	nameInput        *core.InputField
	groupInput       *core.InputField
	expressionInput  *core.InputField
	timezoneInput    *core.InputField
	targetInput      *core.InputField
	roleInput        *core.InputField
	payloadInput     *core.InputField
	windowInput      *core.InputField
	stateDropDown    *core.DropDown
	descriptionInput *core.InputField
	message          *tview.TextView
	submitButton     *core.Button
	closeButton      *core.Button
	navigator        *core.ViewNavigation2D
	onShow           func()
}

func NewScheduleFormView(appCtx *core.AppContext, creating bool) *ScheduleFormView {
	var newInput = func(label string, placeholder string) *core.InputField {
		var input = core.NewInputField(appCtx.Theme)
		input.SetLabel(fmt.Sprintf("%-20s", label)).SetPlaceholder(placeholder)
		return input
	}

	var submitLabel = "Save"
	if creating {
		submitLabel = "Create"
	}

	var layout = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &ScheduleFormView{
		Flex:             layout,
		creating:         creating,
		nameInput:        newInput("Name", ""),
		groupInput:       newInput("Group", defaultScheduleGroup),
		expressionInput:  newInput("Schedule", "rate(5 minutes), cron(...) or at(...)"),
		timezoneInput:    newInput("Timezone", "UTC"),
		targetInput:      newInput("Target ARN", ""),
		roleInput:        newInput("Role ARN", "Role the scheduler assumes to invoke the target"),
		payloadInput:     newInput("Input", "Optional text or JSON sent to the target"),
		windowInput:      newInput("Flexible Window", "Minutes, empty for off"),
		stateDropDown:    core.NewDropDown(appCtx.Theme),
		descriptionInput: newInput("Description", ""),
		message:          tview.NewTextView().SetLabel("Status "),
		submitButton:     core.NewButton(submitLabel, appCtx.Theme),
		closeButton:      core.NewButton("Close", appCtx.Theme),
		onShow:           func() {},
	}

	view.stateDropDown.SetLabel(fmt.Sprintf("%-20s", "State"))
	view.stateDropDown.
		AddOption(scheduleStateOptionEnabled, nil).
		AddOption(scheduleStateOptionDisabled, nil).
		SetCurrentOption(0)

	var rows [][]core.View
	if creating {
		layout.
			AddItem(view.nameInput, 1, 0, true).
			AddItem(view.groupInput, 1, 0, false)
		rows = append(rows, []core.View{view.nameInput}, []core.View{view.groupInput})
	}
	layout.
		AddItem(view.expressionInput, 1, 0, !creating).
		AddItem(view.timezoneInput, 1, 0, false).
		AddItem(view.targetInput, 1, 0, false).
		AddItem(view.roleInput, 1, 0, false).
		AddItem(view.payloadInput, 1, 0, false).
		AddItem(view.windowInput, 1, 0, false).
		AddItem(view.stateDropDown, 1, 0, false).
		AddItem(view.descriptionInput, 1, 0, false).
		AddItem(view.message, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.submitButton, 0, 1, true).
				AddItem(view.closeButton, 0, 1, true),
			1, 0, false,
		)
	rows = append(rows,
		[]core.View{view.expressionInput},
		[]core.View{view.timezoneInput},
		[]core.View{view.targetInput},
		[]core.View{view.roleInput},
		[]core.View{view.payloadInput},
		[]core.View{view.windowInput},
		[]core.View{view.stateDropDown},
		[]core.View{view.descriptionInput},
		[]core.View{view.submitButton, view.closeButton},
	)

	view.navigator = core.NewViewNavigation2D(layout, rows, appCtx.App)

	return view
}

// GetValues returns the values as typed
func (inst *ScheduleFormView) GetValues() scheduleFormValues {
	var _, state = inst.stateDropDown.GetCurrentOption()
	return scheduleFormValues{
		Name:        inst.nameInput.GetText(),
		Group:       inst.groupInput.GetText(),
		Expression:  inst.expressionInput.GetText(),
		Timezone:    inst.timezoneInput.GetText(),
		TargetArn:   inst.targetInput.GetText(),
		RoleArn:     inst.roleInput.GetText(),
		Input:       inst.payloadInput.GetText(),
		Window:      inst.windowInput.GetText(),
		Description: inst.descriptionInput.GetText(),
		Enabled:     state != scheduleStateOptionDisabled,
	}
}

// SetValues fills in the form
func (inst *ScheduleFormView) SetValues(values scheduleFormValues) {
	inst.nameInput.SetText(values.Name)
	inst.groupInput.SetText(values.Group)
	inst.expressionInput.SetText(values.Expression)
	inst.timezoneInput.SetText(values.Timezone)
	inst.targetInput.SetText(values.TargetArn)
	inst.roleInput.SetText(values.RoleArn)
	inst.payloadInput.SetText(values.Input)
	inst.windowInput.SetText(values.Window)
	inst.descriptionInput.SetText(values.Description)
	if values.Enabled {
		inst.stateDropDown.SetCurrentOption(0)
	} else {
		inst.stateDropDown.SetCurrentOption(1)
	}
}

func (inst *ScheduleFormView) SetStatusMessage(msg string) {
	inst.message.SetText(msg)
}

func (inst *ScheduleFormView) GetStatusMessage() string {
	return inst.message.GetText(true)
}

// SetOnSubmitFunc sets what happens when Create or Save is pressed. It is
// given the values as typed.
func (inst *ScheduleFormView) SetOnSubmitFunc(handler func(values scheduleFormValues)) {
	inst.submitButton.SetSelectedFunc(func() {
		handler(inst.GetValues())
	})
}

func (inst *ScheduleFormView) SetOnCloseFunc(handler func()) {
	inst.closeButton.SetSelectedFunc(func() {
		inst.message.SetText("")
		handler()
	})
}

// SetOnShowFunc sets what is called each time the form is shown, after it was
// cleared.
func (inst *ScheduleFormView) SetOnShowFunc(handler func()) {
	inst.onShow = handler
}

// OnShow starts from a clean form.
func (inst *ScheduleFormView) OnShow() {
	inst.SetValues(scheduleFormValues{Enabled: true})
	inst.message.SetText("")
	inst.navigator.SetFocusedRow(0)
	inst.onShow()
}

func (inst *ScheduleFormView) GetLastFocusedView() tview.Primitive {
	return inst.navigator.GetLastFocusedView()
}

type FloatingScheduleFormView struct {
	*tview.Flex
	Input *ScheduleFormView
}

func NewFloatingScheduleFormView(appCtx *core.AppContext, title string, creating bool) *FloatingScheduleFormView {
	var input = NewScheduleFormView(appCtx, creating)
	var height = 14
	if creating {
		height += 2
	}

	return &FloatingScheduleFormView{
		Flex:  core.FloatingView(title, input, 76, height),
		Input: input,
	}
}

func (inst *FloatingScheduleFormView) OnShow() {
	inst.Input.OnShow()
}

func (inst *FloatingScheduleFormView) GetLastFocusedView() tview.Primitive {
	return inst.Input.GetLastFocusedView()
}

func (inst *FloatingScheduleFormView) NavigateWithin(key tcell.Key) bool {
	return inst.Input.navigator.NavigateWithin(key)
}

func (inst *FloatingScheduleFormView) FocusFrom(key tcell.Key) {
	inst.Input.navigator.FocusFrom(key)
}
