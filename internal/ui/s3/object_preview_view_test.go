package s3

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gdamore/tcell/v2"
)

func newTextHead(contentType string, size int64) s3.HeadObjectOutput {
	return s3.HeadObjectOutput{ContentType: aws.String(contentType), ContentLength: aws.Int64(size)}
}

func TestPreviewBlocker__AllowsSmallTextObjects(t *testing.T) {
	var tests = []struct {
		name string
		head s3.HeadObjectOutput
		key  string
	}{
		{"text type", newTextHead("text/plain", 10), "notes"},
		{"json type with charset", newTextHead("application/json; charset=utf-8", 10), "data"},
		{"structured suffix", newTextHead("application/vnd.api+json", 10), "data"},
		{"largest allowed size", newTextHead("text/plain", maxPreviewBytes-1), "notes"},
		{"generic type with text extension", newTextHead("binary/octet-stream", 10), "config.YAML"},
		{"missing type with text extension", s3.HeadObjectOutput{ContentLength: aws.Int64(10)}, "app.log"},
	}

	for _, test := range tests {
		if got := previewBlocker(test.head, test.key); got != "" {
			t.Errorf("%s: unexpectedly blocked: %s", test.name, got)
		}
	}
}

func TestPreviewBlocker__BlocksLargeEncodedAndBinaryObjects(t *testing.T) {
	var gzipped = newTextHead("text/plain", 10)
	gzipped.ContentEncoding = aws.String("gzip")

	var tests = []struct {
		name string
		head s3.HeadObjectOutput
		key  string
		want string
	}{
		{"one megabyte", newTextHead("text/plain", maxPreviewBytes), "big.txt", "under 1 MB"},
		{"encoded", gzipped, "a.txt", "gzip encoded"},
		{"image", newTextHead("image/png", 10), "a.png", "image/png"},
		{"generic type with binary extension", newTextHead("application/octet-stream", 10), "a.bin", "content type"},
	}

	for _, test := range tests {
		if got := previewBlocker(test.head, test.key); !strings.Contains(got, test.want) {
			t.Errorf("%s: blocker = %q, want it to mention %q", test.name, got, test.want)
		}
	}
}

func newPreview(t *testing.T, api *fakeS3BucketsApi) *ObjectPreviewView {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var preview = NewObjectPreviewView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return preview
}

func awaitPreviewText(t *testing.T, preview *ObjectPreviewView, want string) {
	t.Helper()
	uitest.Eventually(t, preview.serviceCtx.App, func() bool {
		return strings.Contains(preview.GetText(), want)
	})
}

func TestObjectPreview__HiddenPreviewDownloadsNothing(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), content: []byte("hello")}
	var preview = newPreview(t, api)

	uitest.OnUI(t, preview.serviceCtx.App, func() { preview.SetObject("bucket", "notes.txt") })

	if api.headCalls.Load() != 0 || api.contentCalls.Load() != 0 {
		t.Errorf("api called while hidden: head=%d content=%d", api.headCalls.Load(), api.contentCalls.Load())
	}
	if got := preview.GetText(); !strings.Contains(got, "opened") {
		t.Errorf("placeholder = %q", got)
	}
}

func TestObjectPreview__ShowingPreviewDownloadsSelectedObject(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), content: []byte("hello")}
	var preview = newPreview(t, api)
	uitest.OnUI(t, preview.serviceCtx.App, func() { preview.SetObject("bucket", "notes.txt") })

	uitest.OnUI(t, preview.serviceCtx.App, func() { preview.SetVisible(true) })

	awaitPreviewText(t, preview, "hello")
	if api.contentCalls.Load() != 1 {
		t.Errorf("content downloads = %d, want 1", api.contentCalls.Load())
	}
}

func TestObjectPreview__SelectingObjectWhileVisibleLoadsImmediately(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), content: []byte("hello")}
	var preview = newPreview(t, api)
	uitest.OnUI(t, preview.serviceCtx.App, func() { preview.SetVisible(true) })

	uitest.OnUI(t, preview.serviceCtx.App, func() { preview.SetObject("bucket", "notes.txt") })

	awaitPreviewText(t, preview, "hello")
}

func TestObjectPreview__ReopeningPreviewDoesNotDownloadAgain(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), content: []byte("hello")}
	var preview = newPreview(t, api)
	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetObject("bucket", "notes.txt")
		preview.SetVisible(true)
	})
	awaitPreviewText(t, preview, "hello")

	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetVisible(false)
		preview.SetVisible(true)
	})

	if api.contentCalls.Load() != 1 {
		t.Errorf("content downloads = %d, want 1", api.contentCalls.Load())
	}
}

func TestObjectPreview__PrettyPrintsAndHighlightsJson(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("application/json", 12), content: []byte(`{"a":[1,2]}`)}
	var preview = newPreview(t, api)

	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetObject("bucket", "data.json")
		preview.SetVisible(true)
	})

	awaitPreviewText(t, preview, "\n  \"a\": [")
	uitest.OnUI(t, preview.serviceCtx.App, func() {
		if !preview.JsonHighlighting() {
			t.Error("json should be highlighted")
		}
	})
}

func TestObjectPreview__PlainTextIsNotHighlighted(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), content: []byte(`{"a" nope`)}
	var preview = newPreview(t, api)

	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetObject("bucket", "notes.txt")
		preview.SetVisible(true)
	})

	awaitPreviewText(t, preview, `{"a" nope`)
	uitest.OnUI(t, preview.serviceCtx.App, func() {
		if preview.JsonHighlighting() {
			t.Error("plain text should not be highlighted")
		}
	})
}

func TestObjectPreview__LargeObjectIsNeverDownloaded(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", maxPreviewBytes)}
	var preview = newPreview(t, api)

	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetObject("bucket", "big.txt")
		preview.SetVisible(true)
	})

	awaitPreviewText(t, preview, "under 1 MB")
	if api.contentCalls.Load() != 0 {
		t.Errorf("content downloads = %d, want 0", api.contentCalls.Load())
	}
}

func TestObjectPreview__BinaryContentIsNotShown(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 3), content: []byte{'a', 0, 'b'}}
	var preview = newPreview(t, api)

	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetObject("bucket", "weird.txt")
		preview.SetVisible(true)
	})

	awaitPreviewText(t, preview, "binary content")
}

func TestObjectPreview__DownloadErrorIsReportedAndRetriedOnReopen(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), contentErr: errors.New("access denied")}
	var preview = newPreview(t, api)
	var callback, received = uitest.CaptureErrors()
	preview.ErrorMessageCallback = callback

	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetObject("bucket", "notes.txt")
		preview.SetVisible(true)
	})

	if got := uitest.AwaitError(t, received); got != "access denied (s3://bucket/notes.txt)" {
		t.Errorf("message = %q, want %q", got, "access denied (s3://bucket/notes.txt)")
	}
	awaitPreviewText(t, preview, "Failed")

	api.contentErr = nil
	api.content = []byte("recovered")
	uitest.OnUI(t, preview.serviceCtx.App, func() {
		preview.SetVisible(false)
		preview.SetVisible(true)
	})
	awaitPreviewText(t, preview, "recovered")
}

func TestDetailsView__PreviewTabOnlyDownloadsOnceItIsOpened(t *testing.T) {
	var api = &fakeS3BucketsApi{head: newTextHead("text/plain", 5), content: []byte("hello")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var view = NewS3bucketsDetailsView(
		NewBucketListTable(serviceCtx),
		NewBucketObjectsTable(serviceCtx),
		NewS3ObjectDetailsTable(serviceCtx),
		NewObjectPreviewView(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	uitest.OnUI(t, serviceCtx.App, func() {
		view.objectPreview.SetObject("bucket", "notes.txt")
		serviceCtx.App.SetFocus(view.objectDetailsTable)
	})
	if api.contentCalls.Load() != 0 {
		t.Fatalf("downloaded before the tab was opened")
	}

	uitest.PressKey(t, serviceCtx.App, tcell.KeyTab)

	awaitPreviewText(t, view.objectPreview, "hello")
	uitest.PressKey(t, serviceCtx.App, tcell.KeyTab)
	uitest.PressKey(t, serviceCtx.App, tcell.KeyTab)
	if api.contentCalls.Load() != 1 {
		t.Errorf("content downloads after reopening = %d, want 1", api.contentCalls.Load())
	}
}

func TestDetailsView__SelectingObjectUsesTheBucketItWasListedFrom(t *testing.T) {
	var api = &fakeS3BucketsApi{
		buckets: []types.Bucket{newBucket("alpha"), newBucket("beta")},
		objects: []types.Object{newObject("notes.txt", 5)},
		head:    newTextHead("text/plain", 5),
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var view = NewS3bucketsDetailsView(
		NewBucketListTable(serviceCtx),
		NewBucketObjectsTable(serviceCtx),
		NewS3ObjectDetailsTable(serviceCtx),
		NewObjectPreviewView(serviceCtx),
		serviceCtx,
	)
	view.InitInputCapture()
	uitest.ShowPage(t, serviceCtx.App, view)
	view.bucketsTable.RefreshBuckets(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.bucketsTable.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() {
		view.bucketsTable.Select(1, 0)
		uitest.PressEnter(view.bucketsTable)
	})
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.objectsTable.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, serviceCtx.App, func() {
		// moving the bucket cursor must not change which bucket the objects belong to
		view.bucketsTable.Select(2, 0)
		view.objectsTable.Select(2, 0)
		uitest.PressEnter(view.objectsTable)
	})

	uitest.Eventually(t, serviceCtx.App, func() bool { return api.headTargets.Load() != nil })
	if got := api.headTargets.Load(); got != "alpha/notes.txt" {
		t.Errorf("HeadObject target = %v, want alpha/notes.txt", got)
	}
}
