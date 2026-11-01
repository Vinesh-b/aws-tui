package eventbridge

import (
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/gdamore/tcell/v2"
)

const (
	// createScheduleKey opens the create schedule view on the schedule list.
	createScheduleKey = 'c'
	// editScheduleKey opens the edit view for the selected schedule.
	editScheduleKey = 'e'
	// deleteScheduleKey opens the delete confirmation for the selected schedule.
	deleteScheduleKey = 'x'
	// toggleScheduleKey enables the selected schedule if it is disabled, and
	// disables it otherwise.
	toggleScheduleKey = 'T'
)

const (
	createScheduleOverlayId = "CREATE_SCHEDULE"
	editScheduleOverlayId   = "EDIT_SCHEDULE"
	deleteScheduleOverlayId = "DELETE_SCHEDULE"
)

// scheduleId is how a schedule is found, its name being unique in its group
type scheduleId struct {
	Group string
	Name  string
}

func (id scheduleId) String() string {
	return id.Group + "/" + id.Name
}

func (id scheduleId) isEmpty() bool {
	return id.Name == ""
}

func scheduleIdOf(summary types.ScheduleSummary) scheduleId {
	return scheduleId{Group: aws.ToString(summary.GroupName), Name: aws.ToString(summary.Name)}
}

type ScheduleListTable struct {
	*core.SelectableTable[types.ScheduleSummary]
	CreateView       *FloatingScheduleFormView
	EditView         *FloatingScheduleFormView
	DeleteView       *core.FloatingConfirmView
	data             []types.ScheduleSummary
	selectedSchedule types.ScheduleSummary
	// toDelete is the schedule the delete view was shown for
	toDelete scheduleId
	// editing is the schedule the edit view was shown for, and edited is the
	// schedule as loaded for it, which is nil until it has been loaded.
	// editRequest counts the times the edit view was shown, so a load for an
	// earlier showing is ignored.
	editing           scheduleId
	edited            *scheduler.GetScheduleOutput
	editRequest       int
	onScheduleDeleted func(id scheduleId)
	serviceCtx        *core.ServiceContext[awsapi.SchedulerApi]
}

func NewScheduleListTable(
	serviceCtx *core.ServiceContext[awsapi.SchedulerApi],
) *ScheduleListTable {
	var view = &ScheduleListTable{
		SelectableTable: core.NewSelectableTable[types.ScheduleSummary](
			"Schedules",
			core.TableRow{
				"Name",
				"Group",
				"State",
				"Target",
				"LastModified",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		CreateView:        NewFloatingScheduleFormView(serviceCtx.AppContext, "Create Schedule", true),
		EditView:          NewFloatingScheduleFormView(serviceCtx.AppContext, "Edit Schedule", false),
		DeleteView:        core.NewFloatingConfirmView(serviceCtx.AppContext, "Delete Schedule", "Delete"),
		onScheduleDeleted: func(scheduleId) {},
		serviceCtx:        serviceCtx,
	}

	view.AddRuneToggleOverlay(createScheduleOverlayId, view.CreateView, createScheduleKey, false)
	view.CreateView.Input.SetOnSubmitFunc(view.createSchedule)
	view.CreateView.Input.SetOnCloseFunc(func() {
		view.ToggleOverlay(createScheduleOverlayId, true)
	})
	view.HelpView.View.AddItem(string(createScheduleKey), "Create a schedule", nil)

	view.AddRuneToggleOverlay(editScheduleOverlayId, view.EditView, editScheduleKey, false)
	view.EditView.Input.SetOnShowFunc(view.prepareEdit)
	view.EditView.Input.SetOnSubmitFunc(view.editSchedule)
	view.EditView.Input.SetOnCloseFunc(func() {
		view.ToggleOverlay(editScheduleOverlayId, true)
	})
	view.HelpView.View.AddItem(string(editScheduleKey), "Edit the selected schedule", nil)

	view.AddRuneToggleOverlay(deleteScheduleOverlayId, view.DeleteView, deleteScheduleKey, false)
	view.DeleteView.Input.SetOnShowFunc(view.prepareDelete)
	view.DeleteView.Input.SetOnConfirmFunc(func() {
		view.ToggleOverlay(deleteScheduleOverlayId, true)
		view.deleteSchedule(view.toDelete)
	})
	view.DeleteView.Input.SetOnCancelFunc(func() {
		view.ToggleOverlay(deleteScheduleOverlayId, true)
	})
	view.HelpView.View.AddItem(string(deleteScheduleKey), "Delete the selected schedule", nil)

	view.HelpView.View.AddItem(string(toggleScheduleKey), "Enable or disable the selected schedule", nil)

	view.populateScheduleTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func scheduleTargetArn(summary types.ScheduleSummary) string {
	if summary.Target == nil {
		return ""
	}
	return aws.ToString(summary.Target.Arn)
}

func (inst *ScheduleListTable) populateScheduleTable(data []types.ScheduleSummary) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			aws.ToString(row.GroupName),
			string(row.State),
			scheduleTargetArn(row),
			core.FormatTime(row.LastModificationDate),
		})
	}

	inst.SetData(tableData, data, 0)
}

func (inst *ScheduleListTable) RefreshSchedules() {
	inst.refreshSchedulesSelecting(scheduleId{})
}

// refreshSchedulesSelecting reloads the schedules, and then selects the one
// given if it is listed. Otherwise the first one is selected.
func (inst *ScheduleListTable) refreshSchedulesSelecting(id scheduleId) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var schedules []types.ScheduleSummary

	dataLoader.AsyncLoadData(func() {
		var err error
		schedules, err = inst.serviceCtx.Api.ListSchedules(inst.serviceCtx.Ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = schedules
		inst.populateScheduleTable(inst.data)
		inst.selectSchedule(id)
	})
}

// selectSchedule moves the cursor to the schedule if it is listed
func (inst *ScheduleListTable) selectSchedule(id scheduleId) {
	if id.isEmpty() {
		return
	}
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if scheduleIdOf(inst.GetPrivateData(row, 0)) == id {
			inst.Select(row, 0)
			return
		}
	}
}

func (inst *ScheduleListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSchedule = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *ScheduleListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSchedule = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *ScheduleListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshSchedules()
			return nil
		case toggleScheduleKey:
			inst.toggleSchedule(inst.currentSchedule())
			return nil
		}
		return capture(event)
	})
}

func (inst *ScheduleListTable) GetSelectedSchedule() types.ScheduleSummary {
	return inst.selectedSchedule
}

// SetOnScheduleDeleted sets a function called on the UI goroutine after a
// schedule was deleted, before the schedules are reloaded.
func (inst *ScheduleListTable) SetOnScheduleDeleted(handler func(id scheduleId)) {
	inst.onScheduleDeleted = handler
}

// currentSchedule is the schedule on the selected row, which is empty when no
// row is selected
func (inst *ScheduleListTable) currentSchedule() scheduleId {
	var row, _ = inst.GetTable().GetSelection()
	if row < 1 || row >= inst.GetTable().GetRowCount() {
		return scheduleId{}
	}
	return scheduleIdOf(inst.GetPrivateData(row, 0))
}

// prepareDelete notes the selected schedule and asks about it. It is kept so
// the list changing underneath can't change what is deleted.
func (inst *ScheduleListTable) prepareDelete() {
	inst.toDelete = inst.currentSchedule()

	if inst.toDelete.isEmpty() {
		inst.DeleteView.Input.SetMessage("No schedule selected")
		return
	}

	inst.DeleteView.Input.SetMessage(fmt.Sprintf(
		"Delete schedule %s from group %s?\n\nThis can't be undone.", inst.toDelete.Name, inst.toDelete.Group,
	))
}

// deleteSchedule deletes the schedule without blocking the UI, then reloads
// the schedules. Failures are shown as errors.
func (inst *ScheduleListTable) deleteSchedule(id scheduleId) {
	if id.isEmpty() {
		inst.ErrorMessageCallback("No schedule selected")
		return
	}

	go func() {
		var err = inst.serviceCtx.Api.DeleteSchedule(inst.serviceCtx.Ctx, id.Group, id.Name)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}

			if scheduleIdOf(inst.selectedSchedule) == id {
				inst.selectedSchedule = types.ScheduleSummary{}
			}
			inst.onScheduleDeleted(id)
			inst.RefreshSchedules()
		})
	}()
}

// createSchedule creates the schedule without blocking the UI, then reloads
// the schedules. The result is shown in the create view.
func (inst *ScheduleListTable) createSchedule(typed scheduleFormValues) {
	var view = inst.CreateView.Input

	var values, err = validScheduleValues(typed, true)
	if err != nil {
		view.SetStatusMessage(err.Error())
		return
	}
	view.SetStatusMessage("Creating...")

	go func() {
		var arn, err = inst.serviceCtx.Api.CreateSchedule(inst.serviceCtx.Ctx, newCreateScheduleInput(values))

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				view.SetStatusMessage(err.Error())
				return
			}

			view.SetStatusMessage("Schedule created: " + arn)
			inst.refreshSchedulesSelecting(scheduleId{Group: values.Group, Name: values.Name})
		})
	}()
}

// prepareEdit loads the selected schedule into the edit view. It is loaded
// again, rather than taken from the details, so the edit starts from the
// schedule as it is now.
func (inst *ScheduleListTable) prepareEdit() {
	var view = inst.EditView.Input
	var id = inst.currentSchedule()
	inst.editRequest++
	var request = inst.editRequest
	inst.editing = id
	inst.edited = nil

	if id.isEmpty() {
		view.SetStatusMessage("No schedule selected")
		return
	}
	view.SetStatusMessage("Loading " + id.String() + "...")

	go func() {
		var schedule, err = inst.serviceCtx.Api.GetSchedule(inst.serviceCtx.Ctx, id.Group, id.Name)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			// The view was shown again while this one was loading
			if request != inst.editRequest {
				return
			}
			if err != nil {
				view.SetStatusMessage(err.Error())
				return
			}

			inst.edited = &schedule
			view.SetValues(scheduleValuesOf(schedule))
			view.SetStatusMessage("Editing " + id.String())
		})
	}()
}

// editSchedule stores the changes to the schedule being edited without
// blocking the UI, then reloads the schedules. The result is shown in the
// edit view.
func (inst *ScheduleListTable) editSchedule(typed scheduleFormValues) {
	var view = inst.EditView.Input
	if inst.editing.isEmpty() {
		view.SetStatusMessage("No schedule selected")
		return
	}
	if inst.edited == nil {
		view.SetStatusMessage("The schedule has not been loaded yet")
		return
	}

	var values, err = validScheduleValues(typed, false)
	if err != nil {
		view.SetStatusMessage(err.Error())
		return
	}
	var input = newEditScheduleInput(*inst.edited, values)
	var id = inst.editing
	view.SetStatusMessage("Saving...")

	go func() {
		var _, err = inst.serviceCtx.Api.UpdateSchedule(inst.serviceCtx.Ctx, input)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				view.SetStatusMessage(err.Error())
				return
			}

			view.SetStatusMessage("Schedule saved: " + id.String())
			inst.refreshSchedulesSelecting(id)
		})
	}()
}

// toggleSchedule disables the schedule if it is enabled and enables it
// otherwise, without blocking the UI, then reloads the schedules. It is loaded
// first, as an update replaces every setting. Failures are shown as errors.
func (inst *ScheduleListTable) toggleSchedule(id scheduleId) {
	if id.isEmpty() {
		inst.ErrorMessageCallback("No schedule selected")
		return
	}

	go func() {
		var err = inst.setScheduleState(id)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}
			inst.refreshSchedulesSelecting(id)
		})
	}()
}

func (inst *ScheduleListTable) setScheduleState(id scheduleId) error {
	var schedule, err = inst.serviceCtx.Api.GetSchedule(inst.serviceCtx.Ctx, id.Group, id.Name)
	if err != nil {
		return err
	}

	var input = newUpdateScheduleInput(schedule)
	input.State = scheduleState(schedule.State == types.ScheduleStateDisabled)
	_, err = inst.serviceCtx.Api.UpdateSchedule(inst.serviceCtx.Ctx, input)
	return err
}
