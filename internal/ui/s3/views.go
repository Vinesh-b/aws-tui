package s3

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type S3BucketsDetailsView struct {
	*core.ServicePageView
	bucketsTable       *BucketListTable
	objectsTable       *BucketObjectsTable
	objectDetailsTable *S3ObjectDetailsTable
	serviceCtx         *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewS3bucketsDetailsView(
	bucketListTable *BucketListTable,
	bucketObjectsTable *BucketObjectsTable,
	bucketObjectDetailsTable *S3ObjectDetailsTable,
	serviceViewCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *S3BucketsDetailsView {
	const objectsTableSize = 4000
	const bucketsTableSize = 3000

	var details = core.NewResizableView(
		bucketObjectsTable, 6000,
		bucketObjectDetailsTable, 4000,
		tview.FlexColumn,
	)

	var mainPage = core.NewResizableView(
		details, objectsTableSize,
		bucketListTable, bucketsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{bucketObjectsTable, bucketObjectDetailsTable},
			{bucketListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	bucketListTable.ErrorMessageCallback = errorHandler
	bucketObjectsTable.ErrorMessageCallback = errorHandler
	bucketObjectDetailsTable.ErrorMessageCallback = errorHandler

	return &S3BucketsDetailsView{
		ServicePageView:    serviceView,
		bucketsTable:       bucketListTable,
		objectsTable:       bucketObjectsTable,
		objectDetailsTable: bucketObjectDetailsTable,
		serviceCtx:         serviceViewCtx,
	}
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.bucketsTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
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
		inst.serviceCtx.App.SetFocus(inst.objectsTable)
	})

	inst.objectsTable.SetSelectedFunc(func(row, column int) {
		var key = inst.objectsTable.GetSelectedPrefix()
		if key == "" || strings.HasSuffix(key, "/") {
			// directories and "../" have no object to describe
			return
		}
		inst.objectDetailsTable.RefreshDetails(
			inst.bucketsTable.GetSeletedBucket(), key,
		)
	})
}

func NewS3bucketsHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x005500))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewS3BucketsApi(appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		s3DetailsView = NewS3bucketsDetailsView(
			NewBucketListTable(serviceCtx),
			NewBucketObjectsTable(serviceCtx),
			NewS3ObjectDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.S3BUCKETS), appCtx)

	serviceRootView.AddAndSwitchToPage("S3Buckets", s3DetailsView, true)

	serviceRootView.InitPageNavigation()

	s3DetailsView.InitInputCapture()

	return serviceRootView
}
