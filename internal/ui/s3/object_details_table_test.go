package s3

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newObjectDetailsTable() *S3ObjectDetailsTable {
	return NewS3ObjectDetailsTable(uitest.NewServiceContext[awsapi.S3BucketsApi]())
}

func newHeadObject() s3.HeadObjectOutput {
	var modified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return s3.HeadObjectOutput{
		ETag:         aws.String("etag-1"),
		ContentType:  aws.String("text/plain"),
		LastModified: &modified,
	}
}

func TestObjectDetailsTable__ShowsBaseFields(t *testing.T) {
	var table = newObjectDetailsTable()
	table.data = newHeadObject()
	table.data.ContentEncoding = aws.String("gzip")
	table.data.VersionId = aws.String("v1")
	table.data.SSEKMSKeyId = aws.String("kms-key")

	table.populateS3ObjectDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"ETag", "etag-1"},
		{"Content Type", "text/plain"},
		{"Content Encoding", "gzip"},
		{"Version ID", "v1"},
		{"SSE KMS Key", "kms-key"},
		{"Last Modified", "2024-01-02 03:04:05"},
	})
}

func TestObjectDetailsTable__ListsMetadataSortedByKey(t *testing.T) {
	var table = newObjectDetailsTable()
	table.data = newHeadObject()
	table.data.Metadata = map[string]string{"owner": "team-a", "env": "prod"}

	table.populateS3ObjectDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"ETag", "etag-1"},
		{"Content Type", "text/plain"},
		{"Content Encoding", ""},
		{"Version ID", ""},
		{"SSE KMS Key", ""},
		{"Last Modified", "2024-01-02 03:04:05"},
		{"Metadata"},
		{"env", "prod"},
		{"owner", "team-a"},
	})
}

func TestObjectDetailsTable__ShowsFirstAvailableChecksumOnly(t *testing.T) {
	var table = newObjectDetailsTable()
	table.data = newHeadObject()
	table.data.ChecksumSHA1 = aws.String("sha1-value")
	table.data.ChecksumSHA256 = aws.String("sha256-value")

	table.populateS3ObjectDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"ETag", "etag-1"},
		{"Content Type", "text/plain"},
		{"Content Encoding", ""},
		{"Version ID", ""},
		{"SSE KMS Key", ""},
		{"Last Modified", "2024-01-02 03:04:05"},
		{"Checksum", "sha1-value"},
	})
}
