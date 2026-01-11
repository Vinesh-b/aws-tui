package servicetables

import (
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"aws-tui/internal/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/gdamore/tcell/v2"
)

type BucketListTable struct {
	*core.SelectableTable[any]
	selectedBucket string
	data           []types.Bucket
	allBuckets     []types.Bucket
	serviceCtx     *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewBucketListTable(
	serviceViewCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *BucketListTable {

	var view = &BucketListTable{
		SelectableTable: core.NewSelectableTable[any](
			"Buckets",
			core.TableRow{
				"Name",
				"CreationDate",
			},
			serviceViewCtx.AppContext,
		),
		data:       nil,
		serviceCtx: serviceViewCtx,
	}

	view.populateS3BucketsTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetSelectedFunc(func(row, column int) {})

	return view
}

func (inst *BucketListTable) populateS3BucketsTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			*row.Name,
			row.CreationDate.Format(time.DateTime),
		})
	}

	inst.SetData(tableData, nil, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *BucketListTable) RefreshBuckets(force bool) {
	var search = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		if len(search) > 0 {
			inst.data = utils.FuzzySearch(search, inst.allBuckets, func(b types.Bucket) string {
				return aws.ToString(b.Name)
			})
		} else {
			var err error = nil
			inst.allBuckets, err = inst.serviceCtx.Api.ListBuckets(force)
			inst.data = inst.allBuckets
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateS3BucketsTable()
	})
}

func (inst *BucketListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedBucket = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *BucketListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedBucket = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *BucketListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshBuckets(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *BucketListTable) GetSeletedBucket() string {
	return inst.selectedBucket
}
