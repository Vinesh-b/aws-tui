package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaListTable struct {
	*core.SelectableTable[types.FunctionConfiguration]
	data           []types.FunctionConfiguration
	filtered       []types.FunctionConfiguration
	selectedLambda types.FunctionConfiguration
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdasListTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaListTable {

	var view = &LambdaListTable{
		SelectableTable: core.NewSelectableTable[types.FunctionConfiguration](
			"Lambdas",
			core.TableRow{
				"Name",
				"LastModified",
			},
		),
		data:           nil,
		selectedLambda: types.FunctionConfiguration{},
		logger:         logger,
		app:            app,
		api:            api,
	}

	view.populateLambdasTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
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
}

func (inst *LambdaListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
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
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.api.ListLambdas(reset)
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
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshLambdas(true)
		}
		return capture(event)
	})
}

func (inst *LambdaListTable) GetSeletedLambda() types.FunctionConfiguration {
	return inst.selectedLambda
}
