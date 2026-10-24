package s3

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fakeS3BucketsApi struct {
	awsapi.S3BucketsApi
	buckets []types.Bucket
	objects []types.Object
	dirs    []types.CommonPrefix
	head    s3.HeadObjectOutput
	err     error

	content      []byte
	contentErr   error
	headCalls    atomic.Int32
	headTargets  atomic.Value
	contentCalls atomic.Int32
}

func (f *fakeS3BucketsApi) ListBuckets(ctx context.Context, force bool) ([]types.Bucket, error) {
	return f.buckets, f.err
}

func (f *fakeS3BucketsApi) ListObjects(
	ctx context.Context, bucketName string, prefix string, force bool,
) ([]types.Object, []types.CommonPrefix, error) {
	return f.objects, f.dirs, f.err
}

func (f *fakeS3BucketsApi) HeadObject(
	ctx context.Context, bucketArn string, objectKey string, force bool,
) (s3.HeadObjectOutput, error) {
	f.headCalls.Add(1)
	f.headTargets.Store(bucketArn + "/" + objectKey)
	return f.head, f.err
}

func (f *fakeS3BucketsApi) GetObjectContent(
	ctx context.Context, bucketName string, objectKey string, maxBytes int64,
) ([]byte, error) {
	f.contentCalls.Add(1)
	return f.content, f.contentErr
}

func TestBucketListTable__RefreshLoadsBucketsFromApi(t *testing.T) {
	var api = &fakeS3BucketsApi{buckets: []types.Bucket{newBucket("logs"), newBucket("assets")}}
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
	var api = &fakeS3BucketsApi{buckets: []types.Bucket{newBucket("logs"), newBucket("assets")}}
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
	var api = &fakeS3BucketsApi{err: errors.New("access denied")}
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
	var api = &fakeS3BucketsApi{
		objects: []types.Object{newObject("readme.txt", 1536)},
		dirs:    []types.CommonPrefix{newDir("docs/")},
	}
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
	var api = &fakeS3BucketsApi{
		objects: []types.Object{newObject("docs/", 0), newObject("docs/guide.md", 2048)},
	}
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
	var api = &fakeS3BucketsApi{err: errors.New("access denied")}
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
	var api = &fakeS3BucketsApi{head: newHeadObject()}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var table = NewS3ObjectDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshDetails("arn:aws:s3:::logs", "readme.txt")

	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(0, 1) == "etag-1" })
}

func TestObjectDetailsTable__RefreshFailureKeepsPreviousDetails(t *testing.T) {
	var api = &fakeS3BucketsApi{err: errors.New("access denied")}
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
