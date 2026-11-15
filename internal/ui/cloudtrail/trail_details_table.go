package cloudtrail

import (
	"context"
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	"github.com/gdamore/tcell/v2"
)

// TrailDetailsTable shows how a trail is set up, which the list of trails
// holds, and its status, which is loaded for the trail
type TrailDetailsTable struct {
	*core.DetailsTable
	trail      types.Trail
	status     *cloudtrail.GetTrailStatusOutput
	serviceCtx *core.ServiceContext[awsapi.CloudTrailApi]
}

func NewTrailDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CloudTrailApi],
) *TrailDetailsTable {
	var table = &TrailDetailsTable{
		DetailsTable: core.NewDetailsTable("Trail Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshStatus()
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

func (inst *TrailDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.trail.TrailARN != nil {
		var trail = inst.trail
		tableData = []core.TableRow{
			{"Name", aws.ToString(trail.Name)},
			{"Home Region", aws.ToString(trail.HomeRegion)},
			{"Multi-Region", formatBool(trail.IsMultiRegionTrail)},
			{"Organization Trail", formatBool(trail.IsOrganizationTrail)},
			{"Global Service Events", formatBool(trail.IncludeGlobalServiceEvents)},
			{"Log File Validation", formatBool(trail.LogFileValidationEnabled)},
			{"Custom Event Selectors", formatBool(trail.HasCustomEventSelectors)},
			{"Insight Selectors", formatBool(trail.HasInsightSelectors)},
			{"KMS Key", aws.ToString(trail.KmsKeyId)},
			{"ARN", aws.ToString(trail.TrailARN)},
		}

		tableData = append(tableData, core.DetailsSection("Delivery",
			core.TableRow{"S3 Bucket", aws.ToString(trail.S3BucketName)},
			core.TableRow{"S3 Key Prefix", aws.ToString(trail.S3KeyPrefix)},
			core.TableRow{"SNS Topic", aws.ToString(trail.SnsTopicARN)},
			core.TableRow{"Log Group", aws.ToString(trail.CloudWatchLogsLogGroupArn)},
			core.TableRow{"Logs Role", aws.ToString(trail.CloudWatchLogsRoleArn)},
		)...)

		if status := inst.status; status != nil {
			tableData = append(tableData, core.DetailsSection("Status",
				core.TableRow{"Logging", formatBool(status.IsLogging)},
				core.TableRow{"Started Logging", core.FormatTime(status.StartLoggingTime)},
				core.TableRow{"Stopped Logging", core.FormatTime(status.StopLoggingTime)},
				core.TableRow{"Latest Delivery", core.FormatTime(status.LatestDeliveryTime)},
				core.TableRow{"Delivery Error", aws.ToString(status.LatestDeliveryError)},
				core.TableRow{"Latest Digest", core.FormatTime(status.LatestDigestDeliveryTime)},
				core.TableRow{"Digest Error", aws.ToString(status.LatestDigestDeliveryError)},
				core.TableRow{"Latest Logs Delivery", core.FormatTime(status.LatestCloudWatchLogsDeliveryTime)},
				core.TableRow{"Logs Delivery Error", aws.ToString(status.LatestCloudWatchLogsDeliveryError)},
				core.TableRow{"Notification Error", aws.ToString(status.LatestNotificationError)},
			)...)
		}
		rawData = trail
	}

	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetTrail shows the trail as the list has it, without a status until that is
// loaded
func (inst *TrailDetailsTable) SetTrail(trail types.Trail) {
	inst.trail = trail
	inst.status = nil
	inst.populateDetailsTable()
}

// RefreshStatus loads the status of the trail shown
func (inst *TrailDetailsTable) RefreshStatus() {
	var trailArn = aws.ToString(inst.trail.TrailARN)
	if len(trailArn) == 0 {
		return
	}

	var status *cloudtrail.GetTrailStatusOutput
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		status, err = inst.serviceCtx.Api.GetTrailStatus(ctx, trailArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another trail was selected while this one loaded
		if aws.ToString(inst.trail.TrailARN) != trailArn {
			return
		}
		inst.status = status
		inst.populateDetailsTable()
	})
}
