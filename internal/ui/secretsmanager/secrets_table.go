package secretsmanager

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/gdamore/tcell/v2"
)

type SecretsListTable struct {
	*core.SelectableTable[types.SecretListEntry]
	data           []types.SecretListEntry
	filtered       []types.SecretListEntry
	selectedSecret types.SecretListEntry
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
		),
		serviceCtx: serviceViewCtx,
	}

	view.populateTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			view.FilterByName(view.GetSearchText())
		}
	})
	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

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

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *SecretsListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(
			name,
			inst.data,
			func(f types.SecretListEntry) string {
				return aws.ToString(f.Name)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable(inst.filtered)
	})
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
