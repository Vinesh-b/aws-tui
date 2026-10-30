package secretsmanager

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/secretsmanager/secretsmanagerfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type deleteSecretFixture struct {
	table *SecretsListTable
	api   *secretsmanagerfake.Api
}

func newDeleteSecretFixture(t *testing.T, api *secretsmanagerfake.Api) *deleteSecretFixture {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SecretsManagerApi](api)
	var table = NewSecretsListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(table, true)
		serviceCtx.App.SetFocus(table)
	})
	table.RefreshSecrets(true)
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return table.GetTable().GetRowCount() == len(api.Secrets())+1
	})
	return &deleteSecretFixture{table: table, api: api}
}

func newSecrets() []types.SecretListEntry {
	return []types.SecretListEntry{newSecret("db/password", "Database"), newSecret("api/key", "")}
}

func (f *deleteSecretFixture) app() *tview.Application { return f.table.serviceCtx.App }

func (f *deleteSecretFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app(), func() {})
}

func (f *deleteSecretFixture) openDelete(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyRune, core.APP_KEY_BINDINGS.RemoveItem)
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

// confirmDelete moves focus from Cancel to Delete and presses it
func (f *deleteSecretFixture) confirmDelete(t *testing.T) {
	t.Helper()
	f.focusNext(t)
	f.press(t, tcell.KeyEnter, 0)
}

func (f *deleteSecretFixture) deleteMessage(t *testing.T) string {
	t.Helper()
	var message string
	uitest.OnUI(t, f.app(), func() { message = f.table.DeleteView.Input.GetMessage() })
	return message
}

func TestDeleteSecret__ConfirmingDeletesTheSelectedSecretAndReloads(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithSecrets(newSecrets()))
	var f = newDeleteSecretFixture(t, api)
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.openDelete(t)
	if got := f.deleteMessage(t); !strings.Contains(got, "Delete the secret api/key?") {
		t.Errorf("message = %q", got)
	}
	api.SetSecrets(newSecrets()[:1])
	f.confirmDelete(t)

	uitest.Eventually(t, f.app(), func() bool {
		return f.table.GetTable().GetRowCount() == 2 && f.table.GetCellText(1, 0) == "db/password"
	})
	if deleted := api.Deleted(); len(deleted) != 1 || deleted[0] != "arn:api/key" {
		t.Errorf("Unexpected deletes %v", deleted)
	}
	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
}

func TestDeleteSecret__CancellingDeletesNothing(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithSecrets(newSecrets()))
	var f = newDeleteSecretFixture(t, api)

	f.openDelete(t)
	// Focus starts on Cancel, so Enter alone must not delete
	f.press(t, tcell.KeyEnter, 0)

	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
	if deleted := api.Deleted(); len(deleted) != 0 {
		t.Errorf("Unexpected deletes %v", deleted)
	}
}

func TestDeleteSecret__ReportsApiError(t *testing.T) {
	var api = secretsmanagerfake.New(
		secretsmanagerfake.WithSecrets(newSecrets()),
		secretsmanagerfake.WithDeleteErr(errors.New("AccessDenied")),
	)
	var f = newDeleteSecretFixture(t, api)
	var callback, received = uitest.CaptureErrors()
	f.table.ErrorMessageCallback = callback

	f.openDelete(t)
	f.confirmDelete(t)

	if got := uitest.AwaitError(t, received); got != "AccessDenied" {
		t.Errorf("error = %q", got)
	}
}

func TestDeleteSecret__NothingSelectedReportsAnError(t *testing.T) {
	var api = secretsmanagerfake.New()
	var f = newDeleteSecretFixture(t, api)
	var callback, received = uitest.CaptureErrors()
	f.table.ErrorMessageCallback = callback

	f.openDelete(t)
	if got := f.deleteMessage(t); got != "No secret selected" {
		t.Errorf("message = %q", got)
	}
	f.confirmDelete(t)

	if got := uitest.AwaitError(t, received); got != "No secret selected" {
		t.Errorf("error = %q", got)
	}
	if deleted := api.Deleted(); len(deleted) != 0 {
		t.Errorf("Unexpected deletes %v", deleted)
	}
}

// focusNext moves focus to the next field of the open overlay
func (f *deleteSecretFixture) focusNext(t *testing.T) {
	t.Helper()
	uitest.OnUI(t, f.app(), func() { f.table.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusDown) })
}
