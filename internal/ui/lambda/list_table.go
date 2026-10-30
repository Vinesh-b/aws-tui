package lambda

import (
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/gdamore/tcell/v2"
)

type LambdaListTable struct {
	*core.SelectableTable[types.FunctionConfiguration]
	data           []types.FunctionConfiguration
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
		).SetColumnExpansion(0, 1).SetFilterColumns(0),
		data:           nil,
		selectedLambda: types.FunctionConfiguration{},
		serviceCtx:     serviceCtx,
	}

	view.populateLambdasTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

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
	inst.Select(1, 0)
}

func (inst *LambdaListTable) RefreshLambdas(reset bool) {
	inst.refreshLambdas(reset, func() {})
}

// OpenLambda selects the lambda and then calls opened. The lambdas are loaded
// again first if it isn't one of those loaded.
func (inst *LambdaListTable) OpenLambda(name string, opened func()) {
	if inst.selectLambda(name) {
		opened()
		return
	}

	inst.refreshLambdas(true, func() {
		if inst.selectLambda(name) {
			opened()
			return
		}
		inst.ErrorMessageCallback("Lambda %s not found", name)
	})
}

// selectLambda shows all the lambdas, without a search, to select the one with
// the name.
func (inst *LambdaListTable) selectLambda(name string) bool {
	if !slices.ContainsFunc(inst.data, func(f types.FunctionConfiguration) bool {
		return aws.ToString(f.FunctionName) == name
	}) {
		return false
	}

	inst.ClearFilter()
	inst.populateLambdasTable(inst.data)
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if aws.ToString(inst.GetPrivateData(row, 0).FunctionName) == name {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *LambdaListTable) refreshLambdas(reset bool, loaded func()) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListLambdas(inst.serviceCtx.Ctx, reset)
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
		inst.populateLambdasTable(inst.data)
		loaded()
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
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshLambdas(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *LambdaListTable) GetSelectedLambdaLogGroup() string {
	if inst.selectedLambda.LoggingConfig != nil {
		return aws.ToString(inst.selectedLambda.LoggingConfig.LogGroup)
	}
	return ""
}

func (inst *LambdaListTable) GetSelectedLambda() types.FunctionConfiguration {
	return inst.selectedLambda
}
