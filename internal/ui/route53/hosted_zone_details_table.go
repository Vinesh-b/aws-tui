package route53

import (
	"context"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/gdamore/tcell/v2"
)

// HostedZoneDetailsTable shows a hosted zone as the list has it, then its name
// servers and VPCs once it is loaded
type HostedZoneDetailsTable struct {
	*core.DetailsTable
	zone       types.HostedZone
	details    *awsapi.HostedZoneDetails
	serviceCtx *core.ServiceContext[awsapi.Route53Api]
}

func NewHostedZoneDetailsTable(
	serviceContext *core.ServiceContext[awsapi.Route53Api],
) *HostedZoneDetailsTable {
	var table = &HostedZoneDetailsTable{
		DetailsTable: core.NewDetailsTable("Hosted Zone Details", serviceContext.AppContext),
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

func (inst *HostedZoneDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.zone.Id != nil {
		var zone = inst.zone
		var zoneId = awsapi.Route53ZoneId(aws.ToString(zone.Id))
		tableData = []core.TableRow{
			{"Name", dnsName(aws.ToString(zone.Name))},
			{"Id", zoneId},
			{"ARN", zoneArn(zoneId)},
			{"Type", zoneType(zone)},
			{"Records", strconv.FormatInt(aws.ToInt64(zone.ResourceRecordSetCount), 10)},
			{"Comment", zoneComment(zone)},
			{"Linked Service", linkedService(zone.LinkedService)},
		}
		rawData = zone

		if details := inst.details; details != nil {
			tableData = append(tableData,
				core.TableRow{"Name Servers", strings.Join(details.NameServers, ", ")},
			)

			var vpcs []core.TableRow
			for _, vpc := range details.VPCs {
				vpcs = append(vpcs, core.TableRow{aws.ToString(vpc.VPCId), string(vpc.VPCRegion)})
			}
			tableData = append(tableData, core.DetailsSection("VPCs", vpcs...)...)
			rawData = details
		}
	}

	inst.SetTitleExtra(dnsName(aws.ToString(inst.zone.Name)))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetZone shows the hosted zone as the list has it, until it is loaded
func (inst *HostedZoneDetailsTable) SetZone(zone types.HostedZone) {
	inst.zone = zone
	inst.details = nil
	inst.populateDetailsTable()
}

// RefreshDetails loads the hosted zone shown
func (inst *HostedZoneDetailsTable) RefreshDetails() {
	var zoneId = aws.ToString(inst.zone.Id)
	if len(zoneId) == 0 {
		return
	}

	var details *awsapi.HostedZoneDetails
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		details, err = inst.serviceCtx.Api.GetHostedZone(ctx, zoneId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another hosted zone was selected while this one loaded
		if aws.ToString(inst.zone.Id) != zoneId || details == nil {
			return
		}
		inst.details = details
		inst.populateDetailsTable()
	})
}
