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
	*core.ServicePageView
	bucketsTable *BucketListTable
	objectsTable *BucketObjectsTable
	app          *tview.Application
	api          *awsapi.S3BucketsApi
}

func NewS3bucketsDetailsView(
	bucketListTable *BucketListTable,
	bucketObjectsTable *BucketObjectsTable,
	app *tview.Application,
	api *awsapi.S3BucketsApi,
	logger *log.Logger,
) *S3BucketsDetailsView {
	const objectsTableSize = 4000
	const bucketsTableSize = 3000

	var mainPage = core.NewResizableView(
		bucketObjectsTable, objectsTableSize,
		bucketListTable, bucketsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			bucketListTable,
			bucketObjectsTable,
		},
	)

	var errorHandler = func(text string) {
		serviceView.SetAndDisplayError(text)
	}

	bucketListTable.ErrorMessageHandler = errorHandler
	bucketObjectsTable.ErrorMessageHandler = errorHandler

	return &S3BucketsDetailsView{
		ServicePageView: serviceView,
		bucketsTable:    bucketListTable,
		objectsTable:    bucketObjectsTable,
		app:             app,
		api:             api,
	}
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.bucketsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.bucketsTable.RefreshBuckets(false)
		}
	})

	inst.bucketsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var name = inst.bucketsTable.GetSeletedBucket()
		inst.objectsTable.SetSelectedBucket(name)
		inst.objectsTable.RefreshObjects(true)
		inst.app.SetFocus(inst.objectsTable.Table)
	})
}

func NewS3bucketsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
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

	var serviceRootView = core.NewServiceRootView(app, string(S3BUCKETS))

	serviceRootView.AddAndSwitchToPage("S3Buckets", s3DetailsView, true)

	serviceRootView.InitPageNavigation()

	s3DetailsView.InitInputCapture()

	return serviceRootView
}
