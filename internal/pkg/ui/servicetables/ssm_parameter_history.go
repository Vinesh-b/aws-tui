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

type SSMParameterHistoryTable struct {
	*core.SelectableTable[types.ParameterHistory]
	data              []types.ParameterHistory
	filtered          []types.ParameterHistory
	selectedHistory   types.ParameterHistory
	selectedParameter types.Parameter
	serviceCtx        *core.ServiceContext[awsapi.SystemsManagerApi]
}

func NewSSMParameterHistoryTable(
	serviceViewCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *SSMParameterHistoryTable {

	var view = &SSMParameterHistoryTable{
		SelectableTable: core.NewSelectableTable[types.ParameterHistory](
			"SSM Parameter History",
			core.TableRow{
				"Version",
				"Type",
				"Value",
				"LastModified",
			},
			serviceViewCtx.AppContext,
		),
		data:              nil,
		selectedHistory:   types.ParameterHistory{},
		selectedParameter: types.Parameter{},
		serviceCtx:        serviceViewCtx,
	}

	view.populateTable(view.data)
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

func (inst *SSMParameterHistoryTable) populateTable(data []types.ParameterHistory) {
	var tableData []core.TableRow
	var privateData []types.ParameterHistory
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			fmt.Sprintf("%d", row.Version),
			string(row.Type),
			aws.ToString(row.Value),
			aws.ToTime(row.LastModifiedDate).Format(time.DateTime),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetTable().SetSelectable(true, true)
	inst.GetTable().SetFixed(1, 1)
}

func (inst *SSMParameterHistoryTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
			name,
			inst.data,
			func(f types.ParameterHistory) string {
				return aws.ToString(f.Name)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable(inst.filtered)
	})
}

func (inst *SSMParameterHistoryTable) RefreshHistory(reset bool) {
	var paramName = aws.ToString(inst.selectedParameter.Name)
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.GetParameterHistory(paramName, reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
			inst.SetTitleExtra(paramName)
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable(inst.data)
	})
}

func (inst *SSMParameterHistoryTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedHistory = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SSMParameterHistoryTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedHistory = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SSMParameterHistoryTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshHistory(true)
		case core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshHistory(false)
		}
		return capture(event)
	})
}

func (inst *SSMParameterHistoryTable) SetSeletedParameter(param types.Parameter) {
	inst.selectedParameter = param
}

func (inst *SSMParameterHistoryTable) GetSeletedHistory() types.ParameterHistory {
	return inst.selectedHistory
}
