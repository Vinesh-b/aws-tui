package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BucketListTable struct {
	*core.SelectableTable[any]
	selectedBucket string
	data           map[string]types.Bucket
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.S3BucketsApi
}

func NewBucketListTable(
	app *tview.Application,
	api *awsapi.S3BucketsApi,
	logger *log.Logger,
) *BucketListTable {

	var view = &BucketListTable{
		SelectableTable: core.NewSelectableTable[any](
			"Buckets",
			core.TableRow{
				"Name",
				"CreationDate",
			},
		),
		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
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

	inst.SetData(tableData)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(0, 0)
}

func (inst *BucketListTable) RefreshBuckets(force bool) {
	var search = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterByName(search)
		} else {
			var err error = nil
			inst.data, err = inst.api.ListBuckets(force)
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
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshBuckets(true)
		}
		return capture(event)
	})
}

func (inst *BucketListTable) GetSeletedBucket() string {
	return inst.selectedBucket
}
