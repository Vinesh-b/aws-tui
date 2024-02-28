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

type S3BucketsDetailsView struct {
	BucketsTable      *tview.Table
	ObjectsTable      *tview.Table
	SearchInput       *tview.InputField
	RootView          *tview.Flex
	selctedBucketName string
	app               *tview.Application
	api               *s3.S3BucketsApi
}

func NewS3bucketsDetailsView(
	app *tview.Application,
	api *s3.S3BucketsApi,
	logger *log.Logger,
) *S3BucketsDetailsView {
	var bucketsTable = tview.NewTable()
	populateS3BucketsTable(bucketsTable, make(map[string]types.Bucket, 0))

	var objectsTable = tview.NewTable()
	populateS3ObjectsTable(objectsTable, nil, false)

	var inputField = createSearchInput("Buckets")

	const objectsTableSize = 4000
	const bucketsTableSize = 3000

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(objectsTable, 0, objectsTableSize, false).
		AddItem(bucketsTable, 0, bucketsTableSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		objectsTable, bucketsTable,
		objectsTableSize, bucketsTableSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			bucketsTable,
			objectsTable,
		},
	)

	return &S3BucketsDetailsView{
		BucketsTable:      bucketsTable,
		ObjectsTable:      objectsTable,
		SearchInput:       inputField,
		RootView:          serviceView.RootView,
		selctedBucketName: "",
		app:               app,
		api:               api,
	}
}

func (inst *S3BucketsDetailsView) RefreshBuckets(search string, force bool) {
	var data map[string]types.Bucket
	var dataChannel = make(chan map[string]types.Bucket)
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			dataChannel <- inst.api.FilterByName(search)
		} else {
			dataChannel <- inst.api.ListBuckets(force)
		}
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.BucketsTable.Box, resultChannel, func() {
		populateS3BucketsTable(inst.BucketsTable, data)
	})
}

func (inst *S3BucketsDetailsView) RefreshObjects(bucketName string, force bool) {
	var data []types.Object
	var dataChannel = make(chan []types.Object)
	var resultChannel = make(chan struct{})

	go func() {
		dataChannel <- inst.api.ListObjects(bucketName, force)
	}()

	go func() {
		data = <-dataChannel
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.ObjectsTable.Box, resultChannel, func() {
		populateS3ObjectsTable(inst.ObjectsTable, data, !force)
	})
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshBuckets(inst.SearchInput.GetText(), false)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})

	var refreshObjects = func(row int) {
		if row < 1 {
			return
		}
		inst.RefreshObjects(inst.BucketsTable.GetCell(row, 0).Text, true)
	}

	inst.BucketsTable.SetSelectedFunc(func(row, column int) {
		refreshObjects(row)
		inst.app.SetFocus(inst.ObjectsTable)
	})

	inst.ObjectsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshObjects(inst.selctedBucketName, true)
		case tcell.KeyCtrlN:
			inst.RefreshObjects(inst.selctedBucketName, false)
		}
		return event
	})
}

func (inst *S3BucketsDetailsView) InitBucketSelectedCallback() {
	inst.BucketsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selctedBucketName = inst.BucketsTable.GetCell(row, 0).Text
		inst.RefreshObjects(inst.selctedBucketName, true)
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

	var orderedPages = []string{
		"S3Buckets",
	}

	var serviceRootView = NewServiceRootView(
		app, string(S3BUCKETS), pages, orderedPages).Init()

	s3DetailsView.InitInputCapture()
	s3DetailsView.InitBucketSelectedCallback()

	return serviceRootView.RootView
}
