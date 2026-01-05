package servicetables

import (
	"sort"

	"aws-tui/internal/pkg/ui/core"
)

type TagsTable[T any, AwsApi any] struct {
	*core.DetailsTable
	data                []T
	serviceCtx          *core.ServiceContext[AwsApi]
	extractKeyValueFunc func(T) (string, string)
	getTagsFunc         func() ([]T, error)
}

func NewTagsTable[T any, AwsApi any](
	serviceCtx *core.ServiceContext[AwsApi],
	extractKeyValueFunc func(T) (string, string),
	getTagsFunc func() ([]T, error),
) *TagsTable[T, AwsApi] {
	var table = &TagsTable[T, AwsApi]{
		DetailsTable:        core.NewDetailsTable("Tags", serviceCtx.AppContext),
		serviceCtx:          serviceCtx,
		extractKeyValueFunc: extractKeyValueFunc,
		getTagsFunc:         getTagsFunc,
	}

	table.populateTagsTable()

	return table
}

func (inst *TagsTable[T, AwsApi]) populateTagsTable() {
	var tableData []core.TableRow
	for _, t := range inst.data {
		var k, v = inst.extractKeyValueFunc(t)
		tableData = append(tableData, core.TableRow{k, v})
	}

	sort.Slice(tableData, func(i int, j int) bool {
		return tableData[i][0] < tableData[j][0]
	})

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *TagsTable[T, AwsApi]) ClearDetails() {
	inst.data = nil
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTagsTable()
	})
}

func (inst *TagsTable[T, AwsApi]) RefreshDetails() {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.data, err = inst.getTagsFunc()
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTagsTable()
	})
}
