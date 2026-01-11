package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"aws-tui/internal/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/gdamore/tcell/v2"
)

type LambdaListTable struct {
	*core.SelectableTable[types.FunctionConfiguration]
	data           []types.FunctionConfiguration
	filtered       []types.FunctionConfiguration
	selectedLambda types.FunctionConfiguration
	serviceCtx     *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdasListTable(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaListTable {

	var view = &LambdaListTable{
		SelectableTable: core.NewSelectableTable[types.FunctionConfiguration](
			"Lambdas",
			core.TableRow{
				"Name",
				"LastModified",
			},
			serviceCtx.AppContext,
		),
		data:           nil,
		selectedLambda: types.FunctionConfiguration{},
		serviceCtx:     serviceCtx,
	}

	view.populateLambdasTable(view.data)
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

func (inst *LambdaListTable) populateLambdasTable(data []types.FunctionConfiguration) {
	var tableData []core.TableRow
	var privateData []types.FunctionConfiguration
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.FunctionName),
			aws.ToString(row.LastModified),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *LambdaListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(
			name,
			inst.data,
			func(f types.FunctionConfiguration) string {
				return aws.ToString(f.FunctionName)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdasTable(inst.filtered)
	})
}

func (inst *LambdaListTable) RefreshLambdas(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListLambdas(reset)
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
		inst.populateLambdasTable(inst.data)
	})
}

func (inst *LambdaListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedLambda = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *LambdaListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedLambda = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *LambdaListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshLambdas(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *LambdaListTable) GetSeletedLambdaLogGroup() string {
	if inst.selectedLambda.LoggingConfig != nil {
		return aws.ToString(inst.selectedLambda.LoggingConfig.LogGroup)
	}
	return ""
}

func (inst *LambdaListTable) GetSeletedLambda() types.FunctionConfiguration {
	return inst.selectedLambda
}
