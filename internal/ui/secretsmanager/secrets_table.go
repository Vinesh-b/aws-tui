package secretsmanager

import (
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/gdamore/tcell/v2"
)

type SecretsListTable struct {
	*core.SelectableTable[types.SecretListEntry]
	data           []types.SecretListEntry
	selectedSecret types.SecretListEntry
	DeleteView     *core.FloatingConfirmView
	toDelete       secretRef
	serviceCtx     *core.ServiceContext[awsapi.SecretsManagerApi]
}

func NewSecretsListTable(
	serviceViewCtx *core.ServiceContext[awsapi.SecretsManagerApi],
) *SecretsListTable {
	var view = &SecretsListTable{
		SelectableTable: core.NewSelectableTable[types.SecretListEntry](
			"Secrets",
			core.TableRow{
				"Name",
				"Description",
				"LastChanged",
			},
			serviceViewCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0),
		serviceCtx: serviceViewCtx,
	}

	view.DeleteView = core.NewFloatingConfirmView(serviceViewCtx.AppContext, "Delete Secret", "Delete")
	// Removing the secret from the table deletes it, after it is confirmed
	view.AddRuneToggleOverlay(deleteSecretOverlayId, view.DeleteView, core.APP_KEY_BINDINGS.RemoveItem, false)
	view.DeleteView.Input.SetOnShowFunc(view.prepareDeleteSecret)
	view.DeleteView.Input.SetOnConfirmFunc(func() {
		view.ToggleOverlay(deleteSecretOverlayId, true)
		view.deleteSecret(view.toDelete)
	})
	view.DeleteView.Input.SetOnCancelFunc(func() {
		view.ToggleOverlay(deleteSecretOverlayId, true)
	})

	var deleteKeys = core.FormatRune(core.APP_KEY_BINDINGS.RemoveItem)
	if sequence := core.APP_KEY_BINDINGS.RemoveItemSequence; len(sequence) > 0 {
		deleteKeys += ", " + sequence
	}
	view.HelpView.View.
		AddItem(deleteKeys, "Delete the selected secret", nil)

	view.populateTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *SecretsListTable) populateTable(data []types.SecretListEntry) {
	var tableData []core.TableRow
	var privateData []types.SecretListEntry
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			aws.ToString(row.Description),
			core.FormatTime(row.LastChangedDate),
		})
		privateData = append(privateData, row)
	}

	// Nothing is selected in an empty table, such as after the last secret is deleted
	if len(data) == 0 {
		inst.selectedSecret = types.SecretListEntry{}
	}

	inst.SetData(tableData, privateData, 0)
}

func (inst *SecretsListTable) RefreshSecrets(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListSecrets(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable(inst.data)
	})
}

func (inst *SecretsListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSecret = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SecretsListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSecret = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SecretsListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SetLoadMoreFunc(func() { inst.RefreshSecrets(false) })
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshSecrets(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *SecretsListTable) GetSelectedSecret() types.SecretListEntry {
	return inst.selectedSecret
}

const deleteSecretOverlayId = "DELETE_SECRET"

// secretRef is a secret to delete
type secretRef struct {
	id   string
	name string
}

// prepareDeleteSecret notes the selected secret and asks about it. It is kept so
// the table changing underneath can't change what is deleted.
func (inst *SecretsListTable) prepareDeleteSecret() {
	var view = inst.DeleteView.Input
	inst.toDelete = secretRef{}

	var secret = inst.selectedSecret
	if secret.ARN == nil {
		view.SetMessage("No secret selected")
		return
	}

	inst.toDelete = secretRef{id: aws.ToString(secret.ARN), name: aws.ToString(secret.Name)}
	view.SetMessage(fmt.Sprintf(
		"Delete the secret %s?\n\nIt can be restored for %d days, then it is gone for good.",
		inst.toDelete.name, awsapi.SecretRecoveryWindowDays,
	))
}

// deleteSecret schedules the secret for deletion without blocking the UI, then
// reloads the secrets. Failures are shown as errors.
func (inst *SecretsListTable) deleteSecret(secret secretRef) {
	if secret.id == "" {
		inst.ErrorMessageCallback("No secret selected")
		return
	}

	go func() {
		var err = inst.serviceCtx.Api.DeleteSecret(inst.serviceCtx.Ctx, secret.id)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}

			inst.RefreshSecrets(true)
		})
	}()
}
