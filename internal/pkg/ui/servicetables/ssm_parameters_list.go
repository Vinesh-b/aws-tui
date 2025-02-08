package servicetables

import (
	"fmt"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/gdamore/tcell/v2"
)

type SSMParametersListTable struct {
	*core.SelectableTable[types.Parameter]
	data              []types.Parameter
	filtered          []types.Parameter
	selectedParameter types.Parameter
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
		),
		data:              nil,
		selectedParameter: types.Parameter{},
		serviceCtx:        serviceViewCtx,
	}

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

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *SSMParametersListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
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
		var data, err = inst.serviceCtx.Api.GetParametersByPath(path, reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
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
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshParameters("/", true)
		case core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshParameters("/", false)
		}
		return capture(event)
	})
}

func (inst *SSMParametersListTable) GetSeletedParameter() types.Parameter {
	return inst.selectedParameter
}
