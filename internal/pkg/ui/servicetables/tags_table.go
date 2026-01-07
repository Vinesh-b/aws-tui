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
) *TagsTable[T, AwsApi] {
	var table = &TagsTable[T, AwsApi]{
		DetailsTable: core.NewDetailsTable("Tags", serviceCtx.AppContext),
		serviceCtx:   serviceCtx,
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

func (inst *TagsTable[T, AwsApi]) ClearDetails() *TagsTable[T, AwsApi] {
	inst.data = nil
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTagsTable()
	})
	return inst
}

func (inst *TagsTable[T, AwsApi]) RefreshDetails() *TagsTable[T, AwsApi] {
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

	return inst
}

func (inst *TagsTable[T, AwsApi]) SetExtractKeyValFunc(
	f func(T) (k string, v string),
) *TagsTable[T, AwsApi] {
	inst.extractKeyValueFunc = f
	return inst
}

func (inst *TagsTable[T, AwsApi]) SetGetTagsFunc(
	f func() ([]T, error),
) *TagsTable[T, AwsApi] {
	inst.getTagsFunc = f
	return inst
}
