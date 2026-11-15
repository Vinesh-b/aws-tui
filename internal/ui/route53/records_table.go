package route53

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gdamore/tcell/v2"
)

// RecordsTable shows the records of the selected hosted zone, which can have
// thousands, so they can be searched. Listing them can take many calls, so
// like the resources of a WAF web ACL, nothing is loaded until the tab is
// opened.
type RecordsTable struct {
	*core.SelectableTable[types.ResourceRecordSet]
	serviceCtx *core.ServiceContext[awsapi.Route53Api]
	zone       types.HostedZone
	loadedFor  string
	visible    bool
}

func NewRecordsTable(serviceCtx *core.ServiceContext[awsapi.Route53Api]) *RecordsTable {
	var table = &RecordsTable{
		SelectableTable: core.NewSelectableTable[types.ResourceRecordSet](
			"Records",
			core.TableRow{
				"Name",
				"Type",
				"Routing",
				"Record Id",
				"TTL",
				"Value",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(5, 1).SetFilterColumns(0, 1, 3, 5),
		serviceCtx: serviceCtx,
	}

	table.showRecords(nil)
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.loadedFor = ""
			table.Load()
			return nil
		}
		return event
	})

	return table
}

func (inst *RecordsTable) showRecords(records []types.ResourceRecordSet) {
	var tableData []core.TableRow
	for _, record := range records {
		tableData = append(tableData, core.TableRow{
			dnsName(aws.ToString(record.Name)),
			string(record.Type),
			routingPolicy(record),
			aws.ToString(record.SetIdentifier),
			recordTtl(record),
			recordValue(record),
		})
	}

	inst.SetTitleExtra(dnsName(aws.ToString(inst.zone.Name)))
	inst.SetData(tableData, records, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}

// SetZone selects the hosted zone whose records are shown, without loading
// them. Load loads them once the tab is open.
func (inst *RecordsTable) SetZone(zone types.HostedZone) {
	if aws.ToString(zone.Id) == aws.ToString(inst.zone.Id) {
		return
	}
	inst.zone = zone
	inst.loadedFor = ""
	inst.showRecords(nil)
}

func (inst *RecordsTable) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the records of the hosted zone selected, if the tab is open and
// they aren't loaded yet
func (inst *RecordsTable) Load() {
	var zoneId = aws.ToString(inst.zone.Id)
	if !inst.visible || len(zoneId) == 0 || inst.loadedFor == zoneId {
		return
	}
	inst.loadedFor = zoneId

	var records []types.ResourceRecordSet
	var loadFailed = false
	// A large zone takes a call for each 300 records
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 30)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		records, err = inst.serviceCtx.Api.ListRecords(ctx, zoneId)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another hosted zone was selected while this one loaded
		if inst.loadedFor != zoneId {
			return
		}
		// What was found before an error is shown, and opening the tab again
		// tries again
		if loadFailed {
			inst.loadedFor = ""
		}
		inst.showRecords(records)
	})
}
