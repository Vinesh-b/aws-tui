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

var bucketHeadings = core.TableRow{"Name", "CreationDate"}

func newBucket(name string) types.Bucket {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.Bucket{Name: aws.String(name), CreationDate: &created}
}

func newPopulatedBucketListTable(names ...string) *BucketListTable {
	var table = NewBucketListTable(uitest.NewServiceContext[awsapi.S3BucketsApi]())
	for _, name := range names {
		table.data = append(table.data, newBucket(name))
	}
	table.populateS3BucketsTable()
	return table
}

func TestBucketListTable__ShowsHeadingsAndOneRowPerBucket(t *testing.T) {
	var table = newPopulatedBucketListTable("logs", "assets")

	uitest.AssertTableRows(t, table, []core.TableRow{
		bucketHeadings,
		{"logs", "2024-01-02 03:04:05"},
		{"assets", "2024-01-02 03:04:05"},
	})
}

func TestBucketListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedBucketListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{bucketHeadings})
}

func TestBucketListTable__SelectingRowTracksSelectedBucket(t *testing.T) {
	var table = newPopulatedBucketListTable("logs", "assets")

	table.Select(2, 0)

	if got := table.GetSeletedBucket(); got != "assets" {
		t.Errorf("selected bucket = %q, want %q", got, "assets")
	}
}

func TestBucketListTable__SelectingHeadingKeepsPreviousSelection(t *testing.T) {
	var table = newPopulatedBucketListTable("logs", "assets")
	table.Select(2, 0)

	table.Select(0, 0)

	if got := table.GetSeletedBucket(); got != "assets" {
		t.Errorf("selected bucket = %q, want %q", got, "assets")
	}
}
