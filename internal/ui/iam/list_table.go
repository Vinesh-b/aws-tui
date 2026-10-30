package iam

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

// IamListTable lists one kind of IAM resource, such as roles. Each resource
// has a key the api and links know it by, which is its name for roles and
// users and its ARN for policies.
type IamListTable[T any] struct {
	*core.SelectableTable[T]
	data     []T
	selected T
	// kind names the resource in messages, such as "Role"
	kind       string
	keyOf      func(T) string
	rowOf      func(T) core.TableRow
	load       func(ctx context.Context) ([]T, error)
	serviceCtx *core.ServiceContext[awsapi.IamApi]
}

func NewIamListTable[T any](
	serviceCtx *core.ServiceContext[awsapi.IamApi],
	title string,
	headings core.TableRow,
	kind string,
	keyOf func(T) string,
	rowOf func(T) core.TableRow,
	load func(ctx context.Context) ([]T, error),
) *IamListTable[T] {
	var view = &IamListTable[T]{
		SelectableTable: core.NewSelectableTable[T](title, headings, serviceCtx.AppContext).SetColumnExpansion(0, 1).SetFilterColumns(0),
		kind:            kind,
		keyOf:           keyOf,
		rowOf:           rowOf,
		load:            load,
		serviceCtx:      serviceCtx,
	}

	view.populateTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *IamListTable[T]) populateTable(data []T) {
	var tableData []core.TableRow
	for _, item := range data {
		tableData = append(tableData, inst.rowOf(item))
	}

	inst.SetData(tableData, data, 0)
	inst.Select(1, 0)
}

func (inst *IamListTable[T]) Refresh(reset bool) {
	inst.refresh(func() {})
}

func (inst *IamListTable[T]) refresh(loaded func()) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var items []T

	dataLoader.AsyncLoadData(func() {
		var err error
		items, err = inst.load(inst.serviceCtx.Ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = items
		inst.populateTable(inst.data)
		loaded()
	})
}

// Open selects the resource with the key. The resources are loaded again
// first if it isn't one of those loaded.
func (inst *IamListTable[T]) Open(key string) {
	if inst.selectKey(key) {
		return
	}

	inst.refresh(func() {
		if !inst.selectKey(key) {
			inst.ErrorMessageCallback("%s %s not found", inst.kind, key)
		}
	})
}

// selectKey shows all the resources, without a search, to select the one with
// the key.
func (inst *IamListTable[T]) selectKey(key string) bool {
	var idx = slices.IndexFunc(inst.data, func(item T) bool { return inst.keyOf(item) == key })
	if idx < 0 {
		return false
	}

	inst.populateTable(inst.data)
	inst.Select(idx+1, 0)
	return true
}

func (inst *IamListTable[T]) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selected = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *IamListTable[T]) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selected = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *IamListTable[T]) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.Refresh(true)
			return nil
		}
		return capture(event)
	})
}

// GetSelectedKey is the key of the selected resource, empty while none is
func (inst *IamListTable[T]) GetSelectedKey() string {
	return inst.keyOf(inst.selected)
}
