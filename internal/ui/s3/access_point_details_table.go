package s3

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"

	"github.com/gdamore/tcell/v2"
)

// AccessPointDetailsTable shows an access point as the list has it, then what
// is loaded for it: when it was made, how public access is blocked and its
// endpoints
type AccessPointDetailsTable struct {
	*core.DetailsTable
	accessPoint types.AccessPoint
	details     *s3control.GetAccessPointOutput
	serviceCtx  *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewAccessPointDetailsTable(
	serviceContext *core.ServiceContext[awsapi.S3BucketsApi],
) *AccessPointDetailsTable {
	var table = &AccessPointDetailsTable{
		DetailsTable: core.NewDetailsTable("Access Point Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshDetails()
			return nil
		}
		return event
	})

	return table
}

func formatBool(value *bool) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%t", *value)
}

func (inst *AccessPointDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.accessPoint.AccessPointArn != nil {
		var point = inst.accessPoint
		tableData = []core.TableRow{
			{"Name", aws.ToString(point.Name)},
			{"Bucket", aws.ToString(point.Bucket)},
			{"Bucket Account", aws.ToString(point.BucketAccountId)},
			{"Network Origin", string(point.NetworkOrigin)},
			{"VPC", accessPointVpc(point.VpcConfiguration)},
			{"Alias", aws.ToString(point.Alias)},
			{"ARN", aws.ToString(point.AccessPointArn)},
		}
		rawData = point

		if details := inst.details; details != nil {
			tableData = append(tableData, core.TableRow{"Created", core.FormatTime(details.CreationDate)})

			if block := details.PublicAccessBlockConfiguration; block != nil {
				tableData = append(tableData, core.DetailsSection("Block Public Access",
					core.TableRow{"Block Public ACLs", formatBool(block.BlockPublicAcls)},
					core.TableRow{"Ignore Public ACLs", formatBool(block.IgnorePublicAcls)},
					core.TableRow{"Block Public Policy", formatBool(block.BlockPublicPolicy)},
					core.TableRow{"Restrict Public Buckets", formatBool(block.RestrictPublicBuckets)},
				)...)
			}

			var endpoints []core.TableRow
			for _, kind := range slices.Sorted(maps.Keys(details.Endpoints)) {
				endpoints = append(endpoints, core.TableRow{kind, details.Endpoints[kind]})
			}
			tableData = append(tableData, core.DetailsSection("Endpoints", endpoints...)...)
			rawData = details
		}
	}

	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetAccessPoint shows the access point as the list has it, until its details
// are loaded
func (inst *AccessPointDetailsTable) SetAccessPoint(point types.AccessPoint) {
	inst.accessPoint = point
	inst.details = nil
	inst.populateDetailsTable()
}

// RefreshDetails loads the details of the access point shown
func (inst *AccessPointDetailsTable) RefreshDetails() {
	var accessPointArn = aws.ToString(inst.accessPoint.AccessPointArn)
	if len(accessPointArn) == 0 {
		return
	}

	var details *s3control.GetAccessPointOutput
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		details, err = inst.serviceCtx.Api.GetAccessPoint(ctx, accessPointArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another access point was selected while this one loaded
		if aws.ToString(inst.accessPoint.AccessPointArn) != accessPointArn {
			return
		}
		inst.details = details
		inst.populateDetailsTable()
	})
}
