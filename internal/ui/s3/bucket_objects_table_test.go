package s3

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
