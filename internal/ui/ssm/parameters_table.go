package ssm

import (
	"fmt"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/gdamore/tcell/v2"
)

type SSMParametersListTable struct {
	*core.SelectableTable[types.Parameter]
	data              []types.Parameter
	filtered          []types.Parameter
	selectedParameter types.Parameter
	DeleteView        *core.FloatingConfirmView
	toDelete          string
	serviceCtx        *core.ServiceContext[awsapi.SystemsManagerApi]
}

func NewSSMParametersListTable(
	serviceViewCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *SSMParametersListTable {

	var view = &SSMParametersListTable{
		SelectableTable: core.NewSelectableTable[types.Parameter](
			"SSM  Parameters",
			core.TableRow{
				"Name",
				"Type",
				"Version",
				"LastModified",
			},
			serviceViewCtx.AppContext,
		).SetColumnExpansion(0, 1),
		data:              nil,
		selectedParameter: types.Parameter{},
		serviceCtx:        serviceViewCtx,
	}

	view.DeleteView = core.NewFloatingConfirmView(serviceViewCtx.AppContext, "Delete Parameter", "Delete")
	// Removing the parameter from the table deletes it, after it is confirmed
	view.AddRuneToggleOverlay(deleteParameterOverlayId, view.DeleteView, core.APP_KEY_BINDINGS.RemoveItem, false)
	view.DeleteView.Input.SetOnShowFunc(view.prepareDeleteParameter)
	view.DeleteView.Input.SetOnConfirmFunc(func() {
		view.ToggleOverlay(deleteParameterOverlayId, true)
		view.deleteParameter(view.toDelete)
	})
	view.DeleteView.Input.SetOnCancelFunc(func() {
		view.ToggleOverlay(deleteParameterOverlayId, true)
	})

	var deleteKeys = core.FormatRune(core.APP_KEY_BINDINGS.RemoveItem)
	if sequence := core.APP_KEY_BINDINGS.RemoveItemSequence; len(sequence) > 0 {
		deleteKeys += ", " + sequence
	}
	view.HelpView.View.
		AddItem(deleteKeys, "Delete the selected parameter", nil)

	view.populateParametersTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *SSMParametersListTable) populateParametersTable(data []types.Parameter) {
	var tableData []core.TableRow
	var privateData []types.Parameter
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			string(row.Type),
			fmt.Sprintf("%d", row.Version),
			aws.ToTime(row.LastModifiedDate).Format(time.DateTime),
		})
		privateData = append(privateData, row)
	}

	// Nothing is selected in an empty table, such as after the last parameter is deleted
	if len(data) == 0 {
		inst.selectedParameter = types.Parameter{}
	}

	inst.SetData(tableData, privateData, 0)
}

func (inst *SSMParametersListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(
			name,
			inst.data,
			func(f types.Parameter) string {
				return aws.ToString(f.Name)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateParametersTable(inst.filtered)
	})
}

func (inst *SSMParametersListTable) RefreshParameters(path string, reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.GetParametersByPath(inst.serviceCtx.Ctx, path, reset)
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
		inst.populateParametersTable(inst.data)
	})
}

func (inst *SSMParametersListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedParameter = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SSMParametersListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedParameter = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SSMParametersListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SetLoadMoreFunc(func() { inst.RefreshParameters("/", false) })
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshParameters("/", true)
			return nil
		}
		return capture(event)
	})
}

func (inst *SSMParametersListTable) GetSelectedParameter() types.Parameter {
	return inst.selectedParameter
}

const deleteParameterOverlayId = "DELETE_PARAMETER"

// prepareDeleteParameter notes the selected parameter and asks about it. It is
// kept so the table changing underneath can't change what is deleted.
func (inst *SSMParametersListTable) prepareDeleteParameter() {
	var view = inst.DeleteView.Input
	inst.toDelete = ""

	var name = aws.ToString(inst.selectedParameter.Name)
	if name == "" {
		view.SetMessage("No parameter selected")
		return
	}

	inst.toDelete = name
	view.SetMessage(fmt.Sprintf(
		"Delete the parameter %s and its history?\n\nThis can't be undone.", name,
	))
}

// deleteParameter deletes the parameter without blocking the UI, then reloads
// the parameters. Failures are shown as errors.
func (inst *SSMParametersListTable) deleteParameter(name string) {
	if name == "" {
		inst.ErrorMessageCallback("No parameter selected")
		return
	}

	go func() {
		var err = inst.serviceCtx.Api.DeleteParameter(inst.serviceCtx.Ctx, name)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}

			inst.RefreshParameters("/", true)
		})
	}()
}
