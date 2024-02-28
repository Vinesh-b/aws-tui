package ui

import (
	"fmt"
	"log"
	"time"

	"aws-tui/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type S3BucketsDetailsView struct {
	BucketsTable   *tview.Table
	ObjectsTable   *tview.Table
	SearchInput    *tview.InputField
	RefreshBuckets func(search string, reset bool)
	RefreshObjects func(bucketName string, reset bool)
	RootView       *tview.Flex
	app            *tview.Application
	bucketName     string
}

func populateS3BucketsTable(table *tview.Table, data map[string]types.Bucket) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.Name,
			row.CreationDate.Format(time.DateTime),
		})
	}

	initSelectableTable(table, "Buckets",
		tableRow{
			"Name",
			"CreationDate",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateS3ObjectsTable(table *tview.Table, data []types.Object, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			aws.ToString(row.Key),
			fmt.Sprintf("%.1f KB", float64(aws.ToInt64(row.Size))/1024.0),
			row.LastModified.Format(time.DateTime),
		})
	}

	var title = "Objects"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
		tableRow{
			"Key",
			"Size",
			"LastModified",
		},
		tableData,
		[]int{0, 1, 2, 3},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func createS3BucketsTable(params tableCreationParams, api *s3.S3BucketsApi) (
	*tview.Table, func(search string, reset bool),
) {
	var table = tview.NewTable()
	populateS3BucketsTable(table, make(map[string]types.Bucket, 0))

	var refreshViewsFunc = func(search string, reset bool) {
		var data map[string]types.Bucket
		var dataChannel = make(chan map[string]types.Bucket)
		var resultChannel = make(chan struct{})

		go func() {
			if len(search) > 0 {
				dataChannel <- api.FilterByName(search)
			} else {
				dataChannel <- api.ListBuckets(reset)
			}
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateS3BucketsTable(table, data)
		})
	}

	return table, refreshViewsFunc
}

func createS3ObjectsTable(
	params tableCreationParams,
	api *s3.S3BucketsApi,
) (*tview.Table, func(bucketName string, extend bool)) {
	var table = tview.NewTable()
	populateS3ObjectsTable(table, nil, false)

	var refreshViewsFunc = func(bucketName string, extend bool) {
		var data []types.Object
		var dataChannel = make(chan []types.Object)
		var resultChannel = make(chan struct{})

		go func() {
			dataChannel <- api.ListObjects(bucketName, !extend)
		}()

		go func() {
			data = <-dataChannel
			resultChannel <- struct{}{}
		}()

		go loadData(params.App, table.Box, resultChannel, func() {
			populateS3ObjectsTable(table, data, extend)
		})
	}

	return table, refreshViewsFunc
}

func NewS3bucketsDetailsView(
	app *tview.Application,
	api *s3.S3BucketsApi,
	logger *log.Logger,
) *S3BucketsDetailsView {
	var (
		params = tableCreationParams{app, logger}

		bucketsTable, refreshBucketsTable = createS3BucketsTable(params, api)
		objectsTable, refreshObjectsTable = createS3ObjectsTable(params, api)
	)

	var onBucketSelction = func(row int) {
		if row < 1 {
			return
		}
		refreshObjectsTable(bucketsTable.GetCell(row, 0).Text, true)
	}

	bucketsTable.SetSelectedFunc(func(row, column int) {
		onBucketSelction(row)
		app.SetFocus(objectsTable)
	})

	var inputField = createSearchInput("Buckets")
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			refreshBucketsTable(inputField.GetText(), false)
		case tcell.KeyEsc:
			inputField.SetText("")
		default:
			return
		}
	})

	var bucketsView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(objectsTable, 0, 4000, false).
		AddItem(bucketsTable, 0, 3000, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	var startIdx = 0
	initViewNavigation(app, bucketsView, &startIdx,
		[]view{
			inputField,
			bucketsTable,
			objectsTable,
		},
	)

	return &S3BucketsDetailsView{
		BucketsTable:   bucketsTable,
		ObjectsTable:   objectsTable,
		SearchInput:    inputField,
		RefreshBuckets: refreshBucketsTable,
		RefreshObjects: refreshObjectsTable,
		RootView:       bucketsView,
		app:            app,
		bucketName:     "",
	}
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.ObjectsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshObjects(inst.bucketName, false)
		case tcell.KeyCtrlM:
			inst.RefreshObjects(inst.bucketName, true)
		}
		return event
	})
}

func (inst *S3BucketsDetailsView) InitBucketSelectedCallback() {
	inst.BucketsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.bucketName = inst.BucketsTable.GetCell(row, 0).Text
		inst.RefreshObjects(inst.bucketName, false)
		inst.app.SetFocus(inst.ObjectsTable)
	})

}

func createS3bucketsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0x005500))
	defer resetGlobalStyle()

	var (
		api           = s3.NewS3BucketsApi(config, logger)
		s3DetailsView = NewS3bucketsDetailsView(app, api, logger)
	)

	var pages = tview.NewPages().
		AddAndSwitchToPage("S3Buckets", s3DetailsView.RootView, true)

	var pagesNavIdx = 0
	var orderedPages = []string{
		"S3Buckets",
	}

	var paginationView = createPaginatorView()
	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(paginationView.RootView, 1, 0, false)

	initPageNavigation(app, pages, &pagesNavIdx, orderedPages, paginationView.PageCounterView)

	s3DetailsView.InitInputCapture()
	s3DetailsView.InitBucketSelectedCallback()

	return rootView
}
