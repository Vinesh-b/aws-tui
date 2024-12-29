package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type S3BucketsDetailsView struct {
	BucketsTable   *BucketListTable
	ObjectsTable   *BucketObjectsTable
	ObjectInput    *tview.InputField
	RootView       *tview.Flex
	searchableView *core.SearchableView_OLD
	app            *tview.Application
	api            *awsapi.S3BucketsApi
}

func NewS3bucketsDetailsView(
	bucketListTable *BucketListTable,
	bucketObjectsTable *BucketObjectsTable,
	app *tview.Application,
	api *awsapi.S3BucketsApi,
	logger *log.Logger,
) *S3BucketsDetailsView {
	var objectKeyInputField = core.CreateSearchInput("Object Path")

	const objectsTableSize = 4000
	const bucketsTableSize = 3000

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(bucketObjectsTable.RootView, 0, objectsTableSize, false).
		AddItem(tview.NewFlex().
			AddItem(objectKeyInputField, 0, 1, false),
			3, 0, false,
		).
		AddItem(bucketListTable.RootView, 0, bucketsTableSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		bucketObjectsTable.RootView, bucketListTable.RootView,
		objectsTableSize, bucketsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			bucketListTable.RootView,
			objectKeyInputField,
			bucketObjectsTable.RootView,
		},
	)

	return &S3BucketsDetailsView{
		BucketsTable:   bucketListTable,
		ObjectsTable:   bucketObjectsTable,
		ObjectInput:    objectKeyInputField,
		RootView:       serviceView.RootView,
		searchableView: serviceView.SearchableView,
		app:            app,
		api:            api,
	}
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.searchableView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.BucketsTable.RefreshBuckets(false)
		case tcell.KeyEsc:
			inst.searchableView.SetText("")
		default:
			return
		}
	})

	inst.BucketsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var name = inst.BucketsTable.GetSeletedBucket()
		inst.ObjectsTable.SetSelectedBucket(name)
		inst.ObjectsTable.RefreshObjects(true)
		inst.app.SetFocus(inst.ObjectsTable.Table)
	})
}

func NewS3bucketsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x005500))
	defer core.ResetGlobalStyle()

	var (
		api           = awsapi.NewS3BucketsApi(config, logger)
		s3DetailsView = NewS3bucketsDetailsView(
			NewBucketListTable(app, api, logger),
			NewBucketObjectsTable(app, api, logger),
			app, api, logger,
		)
	)

	var pages = tview.NewPages().
		AddAndSwitchToPage("S3Buckets", s3DetailsView.RootView, true)

	var orderedPages = []string{
		"S3Buckets",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(S3BUCKETS), pages, orderedPages).Init()

	s3DetailsView.InitInputCapture()

	return serviceRootView.RootView
}
