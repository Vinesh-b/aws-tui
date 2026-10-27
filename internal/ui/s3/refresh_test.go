package s3

import (
	"aws-tui/internal/ui/s3/s3fake"
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestBucketListTable__RefreshLoadsBucketsFromApi(t *testing.T) {
	var api = s3fake.New(s3fake.WithBuckets([]types.Bucket{newBucket("logs"), newBucket("assets")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewBucketListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshBuckets(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		bucketHeadings,
		{"logs", "2024-01-02 03:04:05"},
		{"assets", "2024-01-02 03:04:05"},
	})
}

func TestBucketListTable__RefreshKeepsOnlyBucketsMatchingSearchText(t *testing.T) {
	var api = s3fake.New(s3fake.WithBuckets([]types.Bucket{newBucket("logs"), newBucket("assets")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewBucketListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSearchText("ass")

	table.RefreshBuckets(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		bucketHeadings,
		{"assets", "2024-01-02 03:04:05"},
	})
}

func TestBucketListTable__RefreshReportsApiError(t *testing.T) {
	var api = s3fake.New(s3fake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewBucketListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshBuckets(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestBucketObjectsTable__RefreshListsObjectsOfSelectedBucket(t *testing.T) {
	var api = s3fake.New(s3fake.WithObjects([]types.Object{newObject("readme.txt", 1536)}), s3fake.WithDirs([]types.CommonPrefix{newDir("docs/")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewBucketObjectsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedBucket("logs")

	table.RefreshObjects(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		objectHeadings,
		{"../", "-", "-"},
		{"docs/", "-", "-"},
		{"readme.txt", "1.5 KB", "2024-01-02 03:04:05"},
	})
}

func TestBucketObjectsTable__RefreshDropsTheCurrentDirEntryFromObjects(t *testing.T) {
	var api = s3fake.New(s3fake.WithObjects([]types.Object{newObject("docs/", 0), newObject("docs/guide.md", 2048)}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewBucketObjectsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedBucket("logs")
	table.selectedDir = "docs/"

	table.RefreshObjects(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		objectHeadings,
		{"../", "-", "-"},
		{"guide.md", "2.0 KB", "2024-01-02 03:04:05"},
	})
}

func TestBucketObjectsTable__RefreshReportsApiError(t *testing.T) {
	var api = s3fake.New(s3fake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewBucketObjectsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshObjects(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestObjectDetailsTable__RefreshShowsHeadObjectOutput(t *testing.T) {
	var api = s3fake.New(s3fake.WithHead(newHeadObject()))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewS3ObjectDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshDetails("arn:aws:s3:::logs", "readme.txt")

	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(0, 1) == "etag-1" })
}

func TestObjectDetailsTable__RefreshFailureKeepsPreviousDetails(t *testing.T) {
	var api = s3fake.New(s3fake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewS3ObjectDetailsTable(serviceCtx)
	table.data = newHeadObject()
	table.populateS3ObjectDetailsTable()
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshDetails("arn:aws:s3:::logs", "readme.txt")

	uitest.AwaitError(t, received)
	uitest.OnUI(t, serviceCtx.App, func() {
		if got := table.GetCellText(0, 1); got != "etag-1" {
			t.Errorf("etag = %q, want the previous value", got)
		}
	})
}
