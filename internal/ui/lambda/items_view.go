package lambda

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

// LambdaItemsView lists what belongs to a lambda, such as its triggers or
// versions, beside the details of the selected one. The items are loaded each
// time another lambda is picked.
type LambdaItemsView[T any] struct {
	*core.ResizableView
	ListTable    *core.SelectableTable[T]
	DetailsTable *core.DetailsTable
	lambdaArn    string
	rowOf        func(T) core.TableRow
	nameOf       func(T) string
	detailsOf    func(T) []core.TableRow
	load         func(ctx context.Context, lambdaArn string) ([]T, error)
	serviceCtx   *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaItemsView[T any](
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
	title string,
	detailsTitle string,
	headings core.TableRow,
	rowOf func(T) core.TableRow,
	nameOf func(T) string,
	detailsOf func(T) []core.TableRow,
	load func(ctx context.Context, lambdaArn string) ([]T, error),
) *LambdaItemsView[T] {
	var listTable = core.NewSelectableTable[T](title, headings, serviceCtx.AppContext)
	var detailsTable = core.NewDetailsTable(detailsTitle, serviceCtx.AppContext)

	const listTableSize = 5000
	const detailsTableSize = 5000

	var view = &LambdaItemsView[T]{
		ResizableView: core.NewResizableView(
			listTable, listTableSize,
			detailsTable, detailsTableSize,
			tview.FlexColumn,
		),
		ListTable:    listTable,
		DetailsTable: detailsTable,
		rowOf:        rowOf,
		nameOf:       nameOf,
		detailsOf:    detailsOf,
		load:         load,
		serviceCtx:   serviceCtx,
	}

	listTable.SetSelectionChangedFunc(func(row, column int) {
		view.showSelectedItem()
	})
	view.populateListTable(nil)

	return view
}

// SetErrorMessageCallback reports what fails in the list or the details
func (inst *LambdaItemsView[T]) SetErrorMessageCallback(callback func(text string, a ...any)) {
	inst.ListTable.ErrorMessageCallback = callback
	inst.DetailsTable.ErrorMessageCallback = callback
}

func (inst *LambdaItemsView[T]) populateListTable(items []T) {
	var tableData []core.TableRow
	for _, item := range items {
		tableData = append(tableData, inst.rowOf(item))
	}

	inst.ListTable.SetData(tableData, items, 0)
	inst.ListTable.GetCell(0, 0).SetExpansion(1)
	inst.ListTable.Select(1, 0)
	inst.showSelectedItem()
}

func (inst *LambdaItemsView[T]) showSelectedItem() {
	var table = inst.ListTable.GetTable()
	var row, _ = table.GetSelection()
	if row < 1 || row >= table.GetRowCount() {
		inst.DetailsTable.SetTitleExtra("")
		inst.DetailsTable.SetRawData(nil)
		inst.DetailsTable.SetData(nil)
		return
	}

	var item = inst.ListTable.GetPrivateData(row, 0)
	inst.DetailsTable.SetTitleExtra(inst.nameOf(item))
	inst.DetailsTable.SetRawData(item)
	inst.DetailsTable.SetData(inst.detailsOf(item))
	inst.DetailsTable.Select(0, 0)
}

// Refresh lists the items of the lambda, which is empty to clear them
func (inst *LambdaItemsView[T]) Refresh(lambdaArn string) {
	inst.lambdaArn = lambdaArn
	inst.ListTable.SetTitleExtra(lambdaName(lambdaArn))
	if len(lambdaArn) == 0 {
		inst.populateListTable(nil)
		return
	}

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var items []T

	dataLoader.AsyncLoadData(func() {
		var err error
		items, err = inst.load(inst.serviceCtx.Ctx, lambdaArn)
		if err != nil {
			inst.ListTable.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.ListTable.Box, func() {
		// Another lambda was picked while these loaded. The loader puts back the
		// title from before it loaded, so it is set again.
		if inst.lambdaArn != lambdaArn {
			inst.ListTable.RefreshTitle(0)
			return
		}
		inst.populateListTable(items)
	})
}
