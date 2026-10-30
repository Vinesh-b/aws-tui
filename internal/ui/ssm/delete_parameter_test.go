package ssm

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/ssm/ssmfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type deleteParameterFixture struct {
	table *SSMParametersListTable
	api   *ssmfake.Api
}

func newParameters() []types.Parameter {
	return []types.Parameter{newParameter("/app/db/host", 3), newParameter("/app/db/port", 1)}
}

func newDeleteParameterFixture(t *testing.T, api *ssmfake.Api) *deleteParameterFixture {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var table = NewSSMParametersListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(table, true)
		serviceCtx.App.SetFocus(table)
	})
	table.RefreshParameters("/", true)
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return table.GetTable().GetRowCount() == len(api.Parameters())+1
	})
	return &deleteParameterFixture{table: table, api: api}
}

func (f *deleteParameterFixture) app() *tview.Application { return f.table.serviceCtx.App }

func (f *deleteParameterFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app(), func() {})
}

func (f *deleteParameterFixture) openDelete(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyRune, core.APP_KEY_BINDINGS.RemoveItem)
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

// confirmDelete moves focus from Cancel to Delete and presses it
func (f *deleteParameterFixture) confirmDelete(t *testing.T) {
	t.Helper()
	f.focusNext(t)
	f.press(t, tcell.KeyEnter, 0)
}

func (f *deleteParameterFixture) deleteMessage(t *testing.T) string {
	t.Helper()
	var message string
	uitest.OnUI(t, f.app(), func() { message = f.table.DeleteView.Input.GetMessage() })
	return message
}

func TestDeleteParameter__ConfirmingDeletesTheSelectedParameterAndReloads(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithParameters(newParameters()))
	var f = newDeleteParameterFixture(t, api)
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.openDelete(t)
	if got := f.deleteMessage(t); !strings.Contains(got, "Delete the parameter /app/db/port and its history?") {
		t.Errorf("message = %q", got)
	}
	api.SetParameters(newParameters()[:1])
	f.confirmDelete(t)

	uitest.Eventually(t, f.app(), func() bool {
		return f.table.GetTable().GetRowCount() == 2 && f.table.GetCellText(1, 0) == "/app/db/host"
	})
	if deleted := api.Deleted(); len(deleted) != 1 || deleted[0] != "/app/db/port" {
		t.Errorf("Unexpected deletes %v", deleted)
	}
	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
}

func TestDeleteParameter__CancellingDeletesNothing(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithParameters(newParameters()))
	var f = newDeleteParameterFixture(t, api)

	f.openDelete(t)
	// Focus starts on Cancel, so Enter alone must not delete
	f.press(t, tcell.KeyEnter, 0)

	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
	if deleted := api.Deleted(); len(deleted) != 0 {
		t.Errorf("Unexpected deletes %v", deleted)
	}
}

func TestDeleteParameter__ReportsApiError(t *testing.T) {
	var api = ssmfake.New(
		ssmfake.WithParameters(newParameters()),
		ssmfake.WithDeleteErr(errors.New("AccessDenied")),
	)
	var f = newDeleteParameterFixture(t, api)
	var callback, received = uitest.CaptureErrors()
	f.table.ErrorMessageCallback = callback

	f.openDelete(t)
	f.confirmDelete(t)

	if got := uitest.AwaitError(t, received); got != "AccessDenied" {
		t.Errorf("error = %q", got)
	}
}

func TestDeleteParameter__NothingSelectedReportsAnError(t *testing.T) {
	var api = ssmfake.New()
	var f = newDeleteParameterFixture(t, api)
	var callback, received = uitest.CaptureErrors()
	f.table.ErrorMessageCallback = callback

	f.openDelete(t)
	if got := f.deleteMessage(t); got != "No parameter selected" {
		t.Errorf("message = %q", got)
	}
	f.confirmDelete(t)

	if got := uitest.AwaitError(t, received); got != "No parameter selected" {
		t.Errorf("error = %q", got)
	}
	if deleted := api.Deleted(); len(deleted) != 0 {
		t.Errorf("Unexpected deletes %v", deleted)
	}
}

// focusNext moves focus from Cancel to the confirm button of the open overlay
func (f *deleteParameterFixture) focusNext(t *testing.T) {
	t.Helper()
	uitest.OnUI(t, f.app(), func() { f.table.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusRight) })
}
