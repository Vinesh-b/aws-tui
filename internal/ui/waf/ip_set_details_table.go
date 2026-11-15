package waf

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	"github.com/gdamore/tcell/v2"
)

// IPSetDetailsTable shows an IP set as the list has it, then the IP set
// loaded for it
type IPSetDetailsTable struct {
	*core.DetailsTable
	summary    types.IPSetSummary
	ipSet      *types.IPSet
	loaded     func(ipSet *types.IPSet)
	serviceCtx *core.ServiceContext[awsapi.WafApi]
}

func NewIPSetDetailsTable(
	serviceContext *core.ServiceContext[awsapi.WafApi],
) *IPSetDetailsTable {
	var table = &IPSetDetailsTable{
		DetailsTable: core.NewDetailsTable("IP Set Details", serviceContext.AppContext),
		loaded:       func(*types.IPSet) {},
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

func (inst *IPSetDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.ARN != nil {
		var summary = inst.summary
		tableData = []core.TableRow{
			{"Name", aws.ToString(summary.Name)},
			{"Description", aws.ToString(summary.Description)},
			{"Id", aws.ToString(summary.Id)},
			{"ARN", aws.ToString(summary.ARN)},
		}
		rawData = summary

		if set := inst.ipSet; set != nil {
			tableData = append(tableData,
				core.TableRow{"IP Version", string(set.IPAddressVersion)},
				core.TableRow{"Addresses", strconv.Itoa(len(set.Addresses))},
			)
			rawData = set
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.Name))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetIPSetLoadedFunc runs the handler with each IP set that is loaded for the
// one shown, or with nil when the one shown changes
func (inst *IPSetDetailsTable) SetIPSetLoadedFunc(handler func(ipSet *types.IPSet)) {
	inst.loaded = handler
}

// SetIPSet shows the IP set as the list has it, until it is loaded
func (inst *IPSetDetailsTable) SetIPSet(summary types.IPSetSummary) {
	inst.summary = summary
	inst.ipSet = nil
	inst.populateDetailsTable()
	inst.loaded(nil)
}

// RefreshDetails loads the IP set shown
func (inst *IPSetDetailsTable) RefreshDetails() {
	var ipSetArn = aws.ToString(inst.summary.ARN)
	if len(ipSetArn) == 0 {
		return
	}

	var ipSet *types.IPSet
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		ipSet, err = inst.serviceCtx.Api.GetIPSet(ctx, ipSetArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another IP set was selected while this one loaded
		if aws.ToString(inst.summary.ARN) != ipSetArn || ipSet == nil {
			return
		}
		inst.ipSet = ipSet
		inst.populateDetailsTable()
		inst.loaded(ipSet)
	})
}
