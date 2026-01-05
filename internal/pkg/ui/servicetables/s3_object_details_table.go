package servicetables

import (
	"fmt"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3ObjectDetailsTable struct {
	*core.DetailsTable
	data              s3.HeadObjectOutput
	selectedObjectKey string
	serviceCtx        *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewS3ObjectDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *S3ObjectDetailsTable {
	var table = &S3ObjectDetailsTable{
		DetailsTable: core.NewDetailsTable("Object Details", serviceCtx.AppContext),
		data:         s3.HeadObjectOutput{},
		serviceCtx:   serviceCtx,
	}

	table.populateS3ObjectDetailsTable()

	return table
}

func (inst *S3ObjectDetailsTable) populateS3ObjectDetailsTable() {
	var tableData = []core.TableRow{
		{"ETag", aws.ToString(inst.data.ETag)},
		{"Content Type", aws.ToString(inst.data.ContentType)},
		{"Content Encoding", aws.ToString(inst.data.ContentEncoding)},
		{"Version Id", aws.ToString(inst.data.VersionId)},
		{"SSE KMS Key", aws.ToString(inst.data.SSEKMSKeyId)},
		{"LastModified", aws.ToTime(inst.data.LastModified).Format(time.DateTime)},
	}

	if len(inst.data.Metadata) > 0 {
		tableData = append(tableData,
			core.TableRow{"Metadata", "────╮"},
		)
	}

	for k, v := range inst.data.Metadata {
		tableData = append(tableData,
			core.TableRow{"", fmt.Sprintf("%s: %s", k, v)},
		)
	}

	var checksums = []*string{
		inst.data.ChecksumCRC64NVME,
		inst.data.ChecksumSHA1,
		inst.data.ChecksumSHA256,
		inst.data.ChecksumCRC32,
		inst.data.ChecksumCRC32C,
	}

	for _, c := range checksums {
		if c != nil {
			tableData = append(tableData,
				core.TableRow{"Checksum", aws.ToString(c)},
			)
			break
		}
	}

	inst.SetTitleExtra(inst.selectedObjectKey)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *S3ObjectDetailsTable) RefreshDetails(bucketArn string, objectKey string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.HeadObject(bucketArn, objectKey, true)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
			return
		}

		inst.data = data
		inst.selectedObjectKey = objectKey
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateS3ObjectDetailsTable()
	})
}
