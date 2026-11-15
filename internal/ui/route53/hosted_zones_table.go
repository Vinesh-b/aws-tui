package route53

import (
	"context"
	"slices"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/gdamore/tcell/v2"
)

// HostedZonesTable lists the hosted zones of the account
type HostedZonesTable struct {
	*core.SelectableTable[types.HostedZone]
	data         []types.HostedZone
	selectedZone types.HostedZone
	serviceCtx   *core.ServiceContext[awsapi.Route53Api]
}

func NewHostedZonesTable(
	serviceContext *core.ServiceContext[awsapi.Route53Api],
) *HostedZonesTable {
	var table = &HostedZonesTable{
		SelectableTable: core.NewSelectableTable[types.HostedZone](
			"Hosted Zones",
			core.TableRow{
				"Name",
				"Type",
				"Records",
				"Comment",
				"Id",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(3, 1).SetFilterColumns(0, 3, 4),
		serviceCtx: serviceContext,
	}

	table.populateHostedZonesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *HostedZonesTable) populateHostedZonesTable() {
	var tableData []core.TableRow
	for _, zone := range inst.data {
		tableData = append(tableData, core.TableRow{
			dnsName(aws.ToString(zone.Name)),
			zoneType(zone),
			strconv.FormatInt(aws.ToInt64(zone.ResourceRecordSetCount), 10),
			zoneComment(zone),
			awsapi.Route53ZoneId(aws.ToString(zone.Id)),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedZone = types.HostedZone{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshHostedZones loads the hosted zones
func (inst *HostedZonesTable) RefreshHostedZones() {
	inst.refreshHostedZones(func() {})
}

func (inst *HostedZonesTable) refreshHostedZones(loaded func()) {
	var zones []types.HostedZone
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		zones, err = inst.serviceCtx.Api.ListHostedZones(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = zones
		inst.populateHostedZonesTable()
		loaded()
	})
}

// Open selects the hosted zone of the id. The hosted zones are loaded again
// first if it isn't one of those loaded.
func (inst *HostedZonesTable) Open(zoneId string) {
	if inst.selectZone(zoneId) {
		return
	}

	inst.refreshHostedZones(func() {
		if !inst.selectZone(zoneId) {
			inst.ErrorMessageCallback("Hosted zone %s not found", zoneId)
		}
	})
}

// selectZone shows all the hosted zones, without a search, to select the one
// with the id
func (inst *HostedZonesTable) selectZone(zoneId string) bool {
	var matches = func(zone types.HostedZone) bool {
		return awsapi.Route53ZoneId(aws.ToString(zone.Id)) == awsapi.Route53ZoneId(zoneId)
	}
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateHostedZonesTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *HostedZonesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedZone = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *HostedZonesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshHostedZones()
			return nil
		}
		return capture(event)
	})
}

func (inst *HostedZonesTable) GetSelectedZone() types.HostedZone {
	return inst.selectedZone
}
