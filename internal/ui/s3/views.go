package s3

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	S3TabNameDetails = "Details"
	S3TabNamePreview = "Preview"
)

type S3BucketsDetailsView struct {
	*core.ServicePageView
	bucketsTable       *BucketListTable
	objectsTable       *BucketObjectsTable
	objectDetailsTable *S3ObjectDetailsTable
	objectPreview      *ObjectPreviewView
	serviceCtx         *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewS3bucketsDetailsView(
	bucketListTable *BucketListTable,
	bucketObjectsTable *BucketObjectsTable,
	bucketObjectDetailsTable *S3ObjectDetailsTable,
	objectPreview *ObjectPreviewView,
	serviceViewCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *S3BucketsDetailsView {
	const objectsTableSize = 4000
	const bucketsTableSize = 3000

	var tabView = core.NewTabViewHorizontal(serviceViewCtx.AppContext).
		AddAndSwitchToTab(S3TabNameDetails, bucketObjectDetailsTable, 0, 1, true).
		AddTab(S3TabNamePreview, objectPreview, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		objectPreview.SetVisible(tabName == S3TabNamePreview)
	})

	var details = core.NewResizableView(
		bucketObjectsTable, 5500,
		tabView, 4500,
		tview.FlexColumn,
		serviceViewCtx.AppContext,
	)

	var mainPage = core.NewResizableView(
		details, objectsTableSize,
		bucketListTable, bucketsTableSize,
		tview.FlexRow,
		serviceViewCtx.AppContext,
	)

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{bucketObjectsTable, tabView},
			{bucketListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	bucketListTable.ErrorMessageCallback = errorHandler
	bucketObjectsTable.ErrorMessageCallback = errorHandler
	bucketObjectsTable.InfoMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.InfoPrompt, text, a...)
	}
	bucketObjectDetailsTable.ErrorMessageCallback = errorHandler
	objectPreview.ErrorMessageCallback = errorHandler

	return &S3BucketsDetailsView{
		ServicePageView:    serviceView,
		bucketsTable:       bucketListTable,
		objectsTable:       bucketObjectsTable,
		objectDetailsTable: bucketObjectDetailsTable,
		objectPreview:      objectPreview,
		serviceCtx:         serviceViewCtx,
	}
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.bucketsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		var name = inst.bucketsTable.GetSelectedBucket()
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
		var bucket = inst.objectsTable.GetSelectedBucket()
		inst.objectDetailsTable.RefreshDetails(bucket, key)
		inst.objectPreview.SetObject(bucket, key)
	})
}

const (
	AccessPointTabNameDetails = "Details"
	AccessPointTabNamePolicy  = "Policy"
)

// AccessPointsPageView is the access points of the account, with the details
// and policy of the one selected in tabs over them, as the objects page has
// the details and preview of an object
type AccessPointsPageView struct {
	*core.ServicePageView
	AccessPointsTable *AccessPointsTable
	DetailsTable      *AccessPointDetailsTable
	PolicyView        *AccessPointPolicyView
	TabView           *core.TabViewHorizontal
}

func NewAccessPointsPageView(
	accessPointsTable *AccessPointsTable,
	detailsTable *AccessPointDetailsTable,
	policyView *AccessPointPolicyView,
	serviceCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *AccessPointsPageView {
	const tabViewSize = 3000
	const listSize = 5000

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(AccessPointTabNameDetails, detailsTable, 0, 1, true).
		AddTab(AccessPointTabNamePolicy, policyView, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		policyView.SetVisible(tabName == AccessPointTabNamePolicy)
	})

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		accessPointsTable, listSize,
		tview.FlexRow,
		serviceCtx.AppContext,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{accessPointsTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	accessPointsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	policyView.ErrorMessageCallback = errorHandler

	// What the list has is shown at once, but the rest is only loaded for the
	// access point the selection stops on
	accessPointsTable.SetSelectionChangedFunc(func(row, column int) {
		var point = accessPointsTable.GetSelectedAccessPoint()
		detailsTable.SetAccessPoint(point)
		policyView.SetAccessPoint(aws.ToString(point.AccessPointArn))
	})
	accessPointsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
		policyView.Load()
	})

	return &AccessPointsPageView{
		ServicePageView:   serviceView,
		AccessPointsTable: accessPointsTable,
		DetailsTable:      detailsTable,
		PolicyView:        policyView,
		TabView:           tabView,
	}
}

func NewS3bucketsHomeView(appCtx *core.AppContext, api awsapi.S3BucketsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x005500))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		s3DetailsView = NewS3bucketsDetailsView(
			NewBucketListTable(serviceCtx),
			NewBucketObjectsTable(serviceCtx),
			NewS3ObjectDetailsTable(serviceCtx),
			NewObjectPreviewView(serviceCtx),
			serviceCtx,
		)
		accessPointsView = NewAccessPointsPageView(
			NewAccessPointsTable(serviceCtx),
			NewAccessPointDetailsTable(serviceCtx),
			NewAccessPointPolicyView(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.S3BUCKETS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("S3Buckets", s3DetailsView, true).
		AddPage("Access Points", accessPointsView, true, true)

	serviceRootView.InitPageNavigation()

	s3DetailsView.InitInputCapture()

	return serviceRootView
}
