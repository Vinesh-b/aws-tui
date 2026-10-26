package iam

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// IamChildTable lists what belongs to the IAM resource with a key, such as the
// policies of a role, loaded each time the key changes.
type IamChildTable[T any] struct {
	*core.SelectableTable[T]
	key        string
	rowOf      func(T) core.TableRow
	load       func(ctx context.Context, key string) ([]T, error)
	serviceCtx *core.ServiceContext[awsapi.IamApi]
}

func NewIamChildTable[T any](
	serviceCtx *core.ServiceContext[awsapi.IamApi],
	title string,
	headings core.TableRow,
	rowOf func(T) core.TableRow,
	load func(ctx context.Context, key string) ([]T, error),
) *IamChildTable[T] {
	var table = &IamChildTable[T]{
		SelectableTable: core.NewSelectableTable[T](title, headings, serviceCtx.AppContext),
		rowOf:           rowOf,
		load:            load,
		serviceCtx:      serviceCtx,
	}

	table.populateTable(nil)

	return table
}

func (inst *IamChildTable[T]) populateTable(items []T) {
	var tableData []core.TableRow
	for _, item := range items {
		tableData = append(tableData, inst.rowOf(item))
	}

	inst.SetTitleExtra(inst.key)
	inst.SetData(tableData, items, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

// Refresh lists what belongs to the resource, which is empty to clear it
func (inst *IamChildTable[T]) Refresh(key string) {
	inst.key = key
	if len(key) == 0 {
		inst.populateTable(nil)
		return
	}

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var items []T

	dataLoader.AsyncLoadData(func() {
		var err error
		items, err = inst.load(inst.serviceCtx.Ctx, key)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The selection moved on to another resource while these loaded. The
		// loader puts back the title from before it loaded, so it is set again.
		if inst.key != key {
			inst.RefreshTitle(0)
			return
		}
		inst.populateTable(items)
	})
}

// GetKey is the key of the resource the items belong to
func (inst *IamChildTable[T]) GetKey() string {
	return inst.key
}

// GetSelectedItem is the selected item, if there is one
func (inst *IamChildTable[T]) GetSelectedItem() (T, bool) {
	var row, _ = inst.GetTable().GetSelection()
	if row < 1 || row >= inst.GetTable().GetRowCount() {
		return *new(T), false
	}
	return inst.GetPrivateData(row, 0), true
}
