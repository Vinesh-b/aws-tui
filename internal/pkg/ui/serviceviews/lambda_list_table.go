package serviceviews

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
	*core.SelectableTable[any]
	selectedLambda string
	data           map[string]types.FunctionConfiguration
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
		SelectableTable: core.NewSelectableTable[any](
			"Lambdas",
			core.TableRow{
				"Name",
				"LastModified",
			},
		),
		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	view.populateLambdasTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *LambdaListTable) populateLambdasTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.FunctionName),
			aws.ToString(row.LastModified),
		})
	}

	inst.SetData(tableData)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *LambdaListTable) RefreshLambdas(force bool) {
	var resultChannel = make(chan struct{})
	var searchText = inst.GetSearchText()

	go func() {
		if len(searchText) > 0 {
			inst.data = inst.api.FilterByName(searchText)
		} else {
			inst.data = inst.api.ListLambdas(force)
		}

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		inst.populateLambdasTable()
	})
}

func (inst *LambdaListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedLambda = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *LambdaListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedLambda = inst.GetCell(row, 0).Text
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

func (inst *LambdaListTable) GetSeletedLambda() string {
	return inst.selectedLambda
}
