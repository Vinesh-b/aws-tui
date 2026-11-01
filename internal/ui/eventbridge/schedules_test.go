package eventbridge

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/eventbridge/schedulerfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	lambdaArn = "arn:aws:lambda:us-east-1:123456789012:function:report"
	roleArn   = "arn:aws:iam::123456789012:role/scheduler"
	queueArn  = "arn:aws:sqs:us-east-1:123456789012:dlq"
	kmsArn    = "arn:aws:kms:us-east-1:123456789012:key/k"
)

var schedulesHeadings = core.TableRow{"Name", "Group", "State", "Target", "LastModified"}

var scheduleModified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

func newSchedule(group string, name string) scheduler.GetScheduleOutput {
	return scheduler.GetScheduleOutput{
		Arn:                        aws.String(schedulerfake.ScheduleArn(group, name)),
		Name:                       aws.String(name),
		GroupName:                  aws.String(group),
		Description:                aws.String("Nightly report"),
		ScheduleExpression:         aws.String("rate(1 day)"),
		ScheduleExpressionTimezone: aws.String("Europe/London"),
		State:                      types.ScheduleStateEnabled,
		ActionAfterCompletion:      types.ActionAfterCompletionNone,
		KmsKeyArn:                  aws.String(kmsArn),
		LastModificationDate:       &scheduleModified,
		FlexibleTimeWindow: &types.FlexibleTimeWindow{
			Mode:                   types.FlexibleTimeWindowModeFlexible,
			MaximumWindowInMinutes: aws.Int32(5),
		},
		Target: &types.Target{
			Arn:              aws.String(lambdaArn),
			RoleArn:          aws.String(roleArn),
			Input:            aws.String(`{"job":"report"}`),
			RetryPolicy:      &types.RetryPolicy{MaximumRetryAttempts: aws.Int32(3)},
			DeadLetterConfig: &types.DeadLetterConfig{Arn: aws.String(queueArn)},
		},
	}
}

func scheduleRow(group string, name string, state types.ScheduleState) core.TableRow {
	return core.TableRow{name, group, string(state), lambdaArn, core.FormatTime(&scheduleModified)}
}

type schedulesFixture struct {
	page   *SchedulesPageView
	table  *ScheduleListTable
	api    *schedulerfake.Api
	errors <-chan string
}

func newSchedulesFixture(t *testing.T, api *schedulerfake.Api) *schedulesFixture {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SchedulerApi](api)
	var page = NewSchedulesPageView(NewScheduleListTable(serviceCtx), NewScheduleDetailsTable(serviceCtx), serviceCtx)
	var callback, received = uitest.CaptureErrors()
	page.ScheduleListTable.ErrorMessageCallback = callback
	page.ScheduleDetailsTable.ErrorMessageCallback = callback
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(page.ScheduleListTable, true)
		serviceCtx.App.SetFocus(page.ScheduleListTable)
	})
	page.ScheduleListTable.RefreshSchedules()
	return &schedulesFixture{page: page, table: page.ScheduleListTable, api: api, errors: received}
}

func (f *schedulesFixture) app() *tview.Application { return f.table.serviceCtx.App }

func (f *schedulesFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app(), func() {})
}

func (f *schedulesFixture) awaitRowCount(t *testing.T, count int) {
	t.Helper()
	uitest.Eventually(t, f.app(), func() bool { return f.table.GetTable().GetRowCount() == count },
		"the schedules to be listed")
}

func (f *schedulesFixture) open(t *testing.T, key rune) {
	t.Helper()
	f.press(t, tcell.KeyRune, key)
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() }, "the overlay to open")
}

// submit moves focus down the open form to its Create or Save button and
// presses it
func (f *schedulesFixture) submit(t *testing.T, form *ScheduleFormView) {
	t.Helper()
	uitest.OnUI(t, f.app(), func() {
		for range 20 {
			if f.app().GetFocus() == form.submitButton {
				return
			}
			f.table.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusDown)
		}
		t.Error("the submit button was never focused")
	})
	f.press(t, tcell.KeyEnter, 0)
}

func (f *schedulesFixture) awaitStatus(t *testing.T, form *ScheduleFormView, want string) {
	t.Helper()
	uitest.Eventually(t, f.app(), func() bool { return form.GetStatusMessage() == want }, "status "+want)
}

func TestScheduleListTable__RefreshListsTheSchedulesOfEveryGroup(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(
		newSchedule("ops", "cleanup"), newSchedule("default", "nightly"),
	)))

	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{
		schedulesHeadings,
		scheduleRow("default", "nightly", types.ScheduleStateEnabled),
		scheduleRow("ops", "cleanup", types.ScheduleStateEnabled),
	})
}

func TestScheduleListTable__RefreshReportsApiError(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithErr(errors.New("access denied"))))

	if got := uitest.AwaitError(t, f.errors); got != "access denied" {
		t.Errorf("error = %q", got)
	}
}

func TestSchedulesPageView__SelectingAScheduleShowsItsDetailsAndTarget(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	var details = f.page.ScheduleDetailsTable

	uitest.Eventually(t, f.app(), func() bool {
		return details.GetCellText(uitest.DetailsRowOf(details.DetailsTable, "Schedule"), 1) == "rate(1 day)"
	}, "the details to be shown")

	uitest.OnUI(t, f.app(), func() {
		for label, want := range map[string]string{
			"Name":            "nightly",
			"Group":           "default",
			"State":           "ENABLED",
			"Timezone":        "Europe/London",
			"Flexible Window": "5 minutes",
			"Role ARN":        roleArn,
			"Retry Attempts":  "3",
		} {
			if got := details.GetCellText(uitest.DetailsRowOf(details.DetailsTable, label), 1); got != want {
				t.Errorf("%s = %q, want %q", label, got, want)
			}
		}
		if uitest.DetailsRowOf(details.DetailsTable, "Target") < 0 {
			t.Error("expected a Target section")
		}
		if got := f.page.TargetView.TextView.GetText(); !strings.Contains(got, lambdaArn) {
			t.Errorf("target view = %q", got)
		}
	})
}

func TestCreateSchedule__CreatesTheScheduleAndReloads(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New())
	var form = f.table.CreateView.Input
	f.awaitRowCount(t, 1)

	f.open(t, createScheduleKey)
	uitest.OnUI(t, f.app(), func() {
		form.nameInput.SetText(" hourly ")
		form.expressionInput.SetText("rate(1 hour)")
		form.targetInput.SetText(lambdaArn)
		form.roleInput.SetText(roleArn)
		form.payloadInput.SetText(`{"a":1}`)
		form.windowInput.SetText("15")
		form.stateDropDown.SetCurrentOption(1)
	})
	f.submit(t, form)

	f.awaitStatus(t, form, "Schedule created: "+schedulerfake.ScheduleArn("default", "hourly"))
	f.awaitRowCount(t, 2)
	var creates = f.api.Creates()
	if len(creates) != 1 {
		t.Fatalf("creates = %v", creates)
	}
	var input = creates[0]
	if aws.ToString(input.Name) != "hourly" || aws.ToString(input.GroupName) != "default" ||
		input.State != types.ScheduleStateDisabled || aws.ToString(input.Target.Input) != `{"a":1}` ||
		aws.ToInt32(input.FlexibleTimeWindow.MaximumWindowInMinutes) != 15 {
		t.Errorf("input = %+v, target = %+v", input, input.Target)
	}
}

func TestCreateSchedule__InvalidValuesAreReportedWithoutCallingTheApi(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New())
	var form = f.table.CreateView.Input
	f.awaitRowCount(t, 1)

	f.open(t, createScheduleKey)
	uitest.OnUI(t, f.app(), func() { form.nameInput.SetText("hourly") })
	f.submit(t, form)

	f.awaitStatus(t, form, "Schedule is empty")
	if calls := f.api.CallsWithPrefix("create "); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

func TestCreateSchedule__ReportsApiError(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New())
	var form = f.table.CreateView.Input
	f.awaitRowCount(t, 1)
	f.api.SetErr(errors.New("access denied"))

	f.open(t, createScheduleKey)
	uitest.OnUI(t, f.app(), func() {
		form.SetValues(validFormValues())
	})
	f.submit(t, form)

	f.awaitStatus(t, form, "access denied")
}

func TestCreateSchedule__StartsCleanEachTimeItIsShown(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New())
	var form = f.table.CreateView.Input
	f.awaitRowCount(t, 1)
	uitest.OnUI(t, f.app(), func() {
		var values = validFormValues()
		values.Enabled = false
		form.SetValues(values)
		form.SetStatusMessage("old")
	})

	f.open(t, createScheduleKey)

	uitest.OnUI(t, f.app(), func() {
		if got := form.GetValues(); got != (scheduleFormValues{Enabled: true}) {
			t.Errorf("values = %+v, want the form reset", got)
		}
		if got := form.GetStatusMessage(); got != "" {
			t.Errorf("status = %q, want it cleared", got)
		}
	})
}

func TestEditSchedule__ShowsTheScheduleAndSavesTheChanges(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(
		newSchedule("default", "a-first"), newSchedule("ops", "nightly"),
	)))
	var form = f.table.EditView.Input
	f.awaitRowCount(t, 3)
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.open(t, editScheduleKey)
	f.awaitStatus(t, form, "Editing ops/nightly")
	uitest.OnUI(t, f.app(), func() {
		if got := form.GetValues(); got.Expression != "rate(1 day)" || got.TargetArn != lambdaArn ||
			got.Window != "5" || got.Input != `{"job":"report"}` || !got.Enabled {
			t.Errorf("form values = %+v", got)
		}
		form.expressionInput.SetText("rate(2 days)")
		form.descriptionInput.SetText("")
	})
	f.submit(t, form)

	f.awaitStatus(t, form, "Schedule saved: ops/nightly")
	var updates = f.api.Updates()
	if len(updates) != 1 {
		t.Fatalf("updates = %v", updates)
	}
	var input = updates[0]
	if aws.ToString(input.Name) != "nightly" || aws.ToString(input.GroupName) != "ops" ||
		aws.ToString(input.ScheduleExpression) != "rate(2 days)" || input.Description != nil {
		t.Errorf("input = %+v", input)
	}
	if aws.ToInt32(input.Target.RetryPolicy.MaximumRetryAttempts) != 3 || aws.ToString(input.KmsKeyArn) != kmsArn {
		t.Errorf("settings the form doesn't have were not kept: %+v", input)
	}
	// The details are reloaded with the list
	var details = f.page.ScheduleDetailsTable
	uitest.Eventually(t, f.app(), func() bool {
		return details.GetCellText(uitest.DetailsRowOf(details.DetailsTable, "Schedule"), 1) == "rate(2 days)"
	}, "the details to show the change")
}

func TestEditSchedule__SavingBeforeTheScheduleIsLoadedStoresNothing(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	var form = f.table.EditView.Input
	f.awaitRowCount(t, 2)
	f.api.SetGetErr(errors.New("throttled"))

	f.open(t, editScheduleKey)
	f.awaitStatus(t, form, "throttled")
	uitest.OnUI(t, f.app(), func() { form.SetValues(validFormValues()) })
	f.submit(t, form)

	f.awaitStatus(t, form, "The schedule has not been loaded yet")
	if calls := f.api.CallsWithPrefix("update "); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

func TestEditSchedule__NothingSelectedSaysSo(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New())
	var form = f.table.EditView.Input
	f.awaitRowCount(t, 1)

	f.open(t, editScheduleKey)

	f.awaitStatus(t, form, "No schedule selected")
}

// awaitState waits for the first schedule listed to be in the state
func (f *schedulesFixture) awaitState(t *testing.T, state types.ScheduleState) {
	t.Helper()
	uitest.Eventually(t, f.app(), func() bool {
		return f.table.GetTable().GetCell(1, 2).Text == string(state)
	}, "the schedule to be "+string(state))
}

func TestToggleSchedule__DisablesAnEnabledScheduleAndEnablesItAgain(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	f.awaitRowCount(t, 2)

	f.press(t, tcell.KeyRune, toggleScheduleKey)
	f.awaitState(t, types.ScheduleStateDisabled)

	f.press(t, tcell.KeyRune, toggleScheduleKey)
	f.awaitState(t, types.ScheduleStateEnabled)

	var updates = f.api.Updates()
	if len(updates) != 2 || aws.ToString(updates[0].ScheduleExpression) != "rate(1 day)" ||
		aws.ToString(updates[0].Target.Input) != `{"job":"report"}` {
		t.Errorf("the other settings were not kept: %+v", updates)
	}
}

func TestToggleSchedule__ReportsApiError(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	f.awaitRowCount(t, 2)
	f.api.SetErr(errors.New("access denied"))

	f.press(t, tcell.KeyRune, toggleScheduleKey)

	if got := uitest.AwaitError(t, f.errors); got != "access denied" {
		t.Errorf("error = %q", got)
	}
	if calls := f.api.CallsWithPrefix("update "); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

// confirm moves focus from Cancel to the confirm button and presses it
func (f *schedulesFixture) confirm(t *testing.T) {
	t.Helper()
	uitest.OnUI(t, f.app(), func() { f.table.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusRight) })
	f.press(t, tcell.KeyEnter, 0)
}

func TestDeleteSchedule__ConfirmingDeletesTheSelectedScheduleAndReloads(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(
		newSchedule("default", "nightly"), newSchedule("ops", "cleanup"),
	)))
	f.awaitRowCount(t, 3)
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.open(t, deleteScheduleKey)
	uitest.OnUI(t, f.app(), func() {
		if got := f.table.DeleteView.Input.GetMessage(); !strings.Contains(got, "Delete schedule cleanup from group ops?") {
			t.Errorf("message = %q", got)
		}
	})
	f.confirm(t)

	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{
		schedulesHeadings, scheduleRow("default", "nightly", types.ScheduleStateEnabled),
	})
	if want := []string{"delete ops/cleanup"}; !slices.Equal(f.api.CallsWithPrefix("delete "), want) {
		t.Errorf("Unexpected calls %v", f.api.CallsWithPrefix("delete "))
	}
}

func TestDeleteSchedule__CancellingDeletesNothing(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	f.awaitRowCount(t, 2)

	f.open(t, deleteScheduleKey)
	// Focus starts on Cancel, so Enter alone must not delete
	f.press(t, tcell.KeyEnter, 0)

	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() }, "the overlay to close")
	if calls := f.api.CallsWithPrefix("delete "); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

func TestDeleteSchedule__DeletingTheOnlyScheduleEmptiesTheDetails(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	var details = f.page.ScheduleDetailsTable
	uitest.Eventually(t, f.app(), func() bool { return details.GetCellText(0, 1) == "nightly" }, "the details")

	f.open(t, deleteScheduleKey)
	f.confirm(t)

	uitest.Eventually(t, f.app(), func() bool {
		return details.GetCellText(0, 1) == "" && f.page.TargetView.TextView.GetText() == "" &&
			f.table.GetTable().GetRowCount() == 1
	}, "the details to be emptied")
}

func TestDeleteSchedule__ReportsApiError(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "nightly"))))
	f.awaitRowCount(t, 2)
	f.api.SetErr(errors.New("access denied"))

	f.open(t, deleteScheduleKey)
	f.confirm(t)

	if got := uitest.AwaitError(t, f.errors); got != "access denied" {
		t.Errorf("error = %q", got)
	}
}

func TestToggleSchedule__KeepsTheToggledScheduleSelected(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(
		newSchedule("default", "a-first"), newSchedule("ops", "nightly"),
	)))
	var details = f.page.ScheduleDetailsTable
	f.awaitRowCount(t, 3)
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.press(t, tcell.KeyRune, toggleScheduleKey)

	uitest.Eventually(t, f.app(), func() bool {
		var row, _ = f.table.GetTable().GetSelection()
		return f.table.GetTable().GetCell(2, 2).Text == string(types.ScheduleStateDisabled) && row == 2 &&
			details.GetCellText(uitest.DetailsRowOf(details.DetailsTable, "State"), 1) == "DISABLED"
	}, "the toggled schedule to stay selected and show its new state")
}

func TestCreateSchedule__SelectsTheNewSchedule(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(newSchedule("default", "a-first"))))
	var form = f.table.CreateView.Input
	f.awaitRowCount(t, 2)

	f.open(t, createScheduleKey)
	uitest.OnUI(t, f.app(), func() { form.SetValues(validFormValues()) })
	f.submit(t, form)

	uitest.Eventually(t, f.app(), func() bool {
		return f.table.GetTable().GetRowCount() == 3 && f.table.GetSelectedSchedule().Name != nil &&
			*f.table.GetSelectedSchedule().Name == "nightly"
	}, "the new schedule to be selected")
}
