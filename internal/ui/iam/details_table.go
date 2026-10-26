package iam

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// IamDetailsTable shows the details of the IAM resource with a key, loaded
// each time the key changes.
type IamDetailsTable[T any] struct {
	*core.DetailsTable
	data *T
	// requested is the key last asked for, so a slower earlier load can't
	// replace it
	requested  string
	onLoaded   []func(*T)
	load       func(ctx context.Context, key string) (*T, error)
	rowsOf     func(*T) []core.TableRow
	nameOf     func(*T) string
	serviceCtx *core.ServiceContext[awsapi.IamApi]
}

func NewIamDetailsTable[T any](
	serviceCtx *core.ServiceContext[awsapi.IamApi],
	title string,
	load func(ctx context.Context, key string) (*T, error),
	rowsOf func(*T) []core.TableRow,
	nameOf func(*T) string,
) *IamDetailsTable[T] {
	var table = &IamDetailsTable[T]{
		DetailsTable: core.NewDetailsTable(title, serviceCtx.AppContext),
		load:         load,
		rowsOf:       rowsOf,
		nameOf:       nameOf,
		serviceCtx:   serviceCtx,
	}

	table.populateDetailsTable()

	return table
}

// AddLoadedFunc adds a function called with the resource each time its details
// are shown, which is nil while there is no resource.
func (inst *IamDetailsTable[T]) AddLoadedFunc(handler func(*T)) {
	inst.onLoaded = append(inst.onLoaded, handler)
}

func (inst *IamDetailsTable[T]) populateDetailsTable() {
	for _, handler := range inst.onLoaded {
		handler(inst.data)
	}

	var tableData []core.TableRow
	if inst.data != nil {
		tableData = inst.rowsOf(inst.data)
		inst.SetTitleExtra(inst.nameOf(inst.data))
		inst.SetRawData(inst.data)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *IamDetailsTable[T]) RefreshDetails(key string) {
	inst.requested = key
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var data *T

	dataLoader.AsyncLoadData(func() {
		var err error
		data, err = inst.load(inst.serviceCtx.Ctx, key)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The loader puts back the title from before it loaded, which a later
		// load may have changed, so what is shown is drawn again
		if inst.requested == key {
			inst.data = data
		}
		inst.populateDetailsTable()
	})
}
