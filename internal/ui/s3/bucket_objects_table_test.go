package s3

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/s3/s3fake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rivo/tview"
)

var objectHeadings = core.TableRow{"Key", "Size", "LastModified"}

func newObjectsTable() *BucketObjectsTable {
	return NewBucketObjectsTable(uitest.NewServiceContext[awsapi.S3BucketsApi]())
}

func newObject(key string, size int64) types.Object {
	var modified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.Object{Key: aws.String(key), Size: aws.Int64(size), LastModified: &modified}
}

func newDir(prefix string) types.CommonPrefix {
	return types.CommonPrefix{Prefix: aws.String(prefix)}
}

func TestBucketObjectsTable__EmptyBucketShowsOnlyParentEntry(t *testing.T) {
	var table = newObjectsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{
		objectHeadings,
		{"../", "-", "-"},
	})
}

func TestBucketObjectsTable__ListsDirsBeforeObjects(t *testing.T) {
	var table = newObjectsTable()

	table.populateS3ObjectsTable(
		[]types.Object{newObject("readme.txt", 1536)},
		[]types.CommonPrefix{newDir("images/"), newDir("docs/")},
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		objectHeadings,
		{"../", "-", "-"},
		{"images/", "-", "-"},
		{"docs/", "-", "-"},
		{"readme.txt", "1.5 KB", "2024-01-02 03:04:05"},
	})
}

func TestBucketObjectsTable__ShowsOnlyBaseNameOfNestedEntries(t *testing.T) {
	var table = newObjectsTable()
	table.selectedDir = "docs/"

	table.populateS3ObjectsTable(
		[]types.Object{newObject("docs/guide.md", 2048)},
		[]types.CommonPrefix{newDir("docs/api/")},
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		objectHeadings,
		{"../", "-", "-"},
		{"api/", "-", "-"},
		{"guide.md", "2.0 KB", "2024-01-02 03:04:05"},
	})
}

func TestBucketObjectsTable__ParentEntryKeyDependsOnCurrentDir(t *testing.T) {
	var tests = []struct {
		name        string
		selectedDir string
		wantParent  string
	}{
		{"bucket root", "", ""},
		{"top level dir", "docs/", ""},
		{"nested dir", "docs/api/", "docs/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table = newObjectsTable()
			table.selectedDir = tt.selectedDir
			table.populateS3ObjectsTable(nil, nil)

			table.Select(1, 0)

			if got := table.GetSelectedPrefix(); got != tt.wantParent {
				t.Errorf("parent prefix = %q, want %q", got, tt.wantParent)
			}
		})
	}
}

func TestBucketObjectsTable__SelectingRowTracksSelectedObject(t *testing.T) {
	var table = newObjectsTable()
	table.populateS3ObjectsTable(
		[]types.Object{newObject("readme.txt", 10)},
		[]types.CommonPrefix{newDir("docs/")},
	)

	table.Select(3, 0)

	if got := table.GetSelectedPrefix(); got != "readme.txt" {
		t.Errorf("selected prefix = %q, want %q", got, "readme.txt")
	}
}

func TestBucketObjectsTable__SettingBucketResetsDirAndSelection(t *testing.T) {
	var table = newObjectsTable()
	table.populateS3ObjectsTable([]types.Object{newObject("readme.txt", 10)}, nil)
	table.Select(2, 0)
	table.selectedDir = "docs/"

	table.SetSelectedBucket("other")

	if table.selectedDir != "" || table.GetSelectedPrefix() != "" {
		t.Errorf("dir = %q, prefix = %q, want both empty", table.selectedDir, table.GetSelectedPrefix())
	}
}

func TestBucketObjectsTable__SearchKeepsTheParentEntryAndFiltersTheRest(t *testing.T) {
	var table = newObjectsTable()
	table.populateS3ObjectsTable(
		[]types.Object{newObject("readme.txt", 1536)},
		[]types.CommonPrefix{newDir("images/"), newDir("docs/")},
	)

	table.SetSearchText("doc")

	uitest.AssertTableRows(t, table, []core.TableRow{
		objectHeadings,
		{"../", "-", "-"},
		{"docs/", "-", "-"},
	})
	if got := aws.ToString(table.GetPrivateData(1, 0).Key); got != "" {
		t.Errorf("parent entry key = %q, want the parent dir", got)
	}
}

func TestBucketObjectsTable__SortKeepsTheParentEntryFirst(t *testing.T) {
	var table = newObjectsTable()
	table.populateS3ObjectsTable(
		[]types.Object{newObject("b.txt", 1), newObject("a.txt", 1)}, nil,
	)

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Sort)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Key ▲", "Size", "LastModified"},
		{"../", "-", "-"},
		{"a.txt", "0.0 KB", "2024-01-02 03:04:05"},
		{"b.txt", "0.0 KB", "2024-01-02 03:04:05"},
	})
}

func TestBucketObjectsTable__SettingBucketClearsTheSearch(t *testing.T) {
	var table = newObjectsTable()
	table.populateS3ObjectsTable([]types.Object{newObject("a.txt", 1), newObject("b.txt", 1)}, nil)
	table.SetSearchText("a.txt")

	table.SetSelectedBucket("other")
	table.populateS3ObjectsTable([]types.Object{newObject("c.txt", 1)}, nil)

	if got := table.GetTable().GetRowCount(); got != 3 {
		t.Errorf("rows = %d, want the parent entry and c.txt", got)
	}
	if text := table.GetSearchText(); text != "" {
		t.Errorf("search = %q, want it cleared", text)
	}
}

func newDownloadTable(
	t *testing.T, api *s3fake.Api,
) (table *BucketObjectsTable, dir string, infos <-chan string, errs <-chan string) {
	t.Helper()
	table = NewBucketObjectsTable(uitest.NewServiceContextWith[awsapi.S3BucketsApi](api))
	dir = t.TempDir()
	table.downloadDir = dir
	table.SetSelectedBucket("bucket")
	table.ErrorMessageCallback, errs = uitest.CaptureErrors()
	table.InfoMessageCallback, infos = uitest.CaptureErrors()
	return table, dir, infos, errs
}

func TestBucketObjectsTable__DownloadSavesTheObjectUnderItsBaseName(t *testing.T) {
	var api = s3fake.New(s3fake.WithContent([]byte("hello")))
	var table, dir, infos, _ = newDownloadTable(t, api)
	table.selectedObject = newObject("docs/guide.md", 5)

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Download)

	var starting = uitest.AwaitError(t, infos)
	var done = uitest.AwaitError(t, infos)
	var want = filepath.Join(dir, "guide.md")
	if !strings.HasPrefix(starting, "Downloading") || !strings.HasPrefix(done, "Downloaded") {
		t.Errorf("messages = %q, %q, want a start and a finish message", starting, done)
	}
	if got, err := os.ReadFile(want); err != nil || string(got) != "hello" {
		t.Errorf("file = %q, %v, want hello", got, err)
	}
	var calls = api.Downloads()
	if len(calls) != 1 || calls[0] != (s3fake.Download{Bucket: "bucket", Key: "docs/guide.md", FileName: want}) {
		t.Errorf("downloads = %+v, want one of docs/guide.md", calls)
	}
}

func TestBucketObjectsTable__DownloadRefusesDirectories(t *testing.T) {
	var api = s3fake.New()
	var table, _, _, errs = newDownloadTable(t, api)
	table.selectedObject = types.Object{Key: aws.String("docs/")}

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Download)

	if got := uitest.AwaitError(t, errs); got != "Select an object to download" {
		t.Errorf("error = %q", got)
	}
	if n := len(api.Downloads()); n != 0 {
		t.Errorf("downloads = %d, want none", n)
	}
}

func TestBucketObjectsTable__DownloadDoesNotReplaceAFile(t *testing.T) {
	var api = s3fake.New(s3fake.WithContent([]byte("new")))
	var table, dir, _, errs = newDownloadTable(t, api)
	var existing = filepath.Join(dir, "a.txt")
	if err := os.WriteFile(existing, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	table.selectedObject = newObject("a.txt", 3)

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Download)

	if got := uitest.AwaitError(t, errs); !strings.Contains(got, "already exists") {
		t.Errorf("error = %q, want already exists", got)
	}
	if got, _ := os.ReadFile(existing); string(got) != "old" {
		t.Errorf("file = %q, want it untouched", got)
	}
	if n := len(api.Downloads()); n != 0 {
		t.Errorf("downloads = %d, want none", n)
	}
}

func TestBucketObjectsTable__DownloadReportsFailures(t *testing.T) {
	var api = s3fake.New(s3fake.WithDownloadErr(errors.New("denied")))
	var table, _, _, errs = newDownloadTable(t, api)
	table.selectedObject = newObject("a.txt", 3)

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Download)

	if got := uitest.AwaitError(t, errs); got != "denied" {
		t.Errorf("error = %q, want denied", got)
	}
}

func TestBucketObjectsTable__HelpListsTheDownloadKey(t *testing.T) {
	var table = newObjectsTable()

	var help, ok = table.HelpView.View.GetItem(0).(*tview.Table)
	if !ok {
		t.Fatal("help view has no table")
	}
	var found = false
	for row := 1; row < help.GetRowCount(); row++ {
		if help.GetCell(row, 0).Text == core.FormatRune(core.APP_KEY_BINDINGS.Download) {
			found = true
		}
	}
	if !found {
		t.Errorf("help has no entry for %q", core.FormatRune(core.APP_KEY_BINDINGS.Download))
	}
}
