package waf

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	"github.com/gdamore/tcell/v2"
)

// IPSetsTable lists the regional IP sets. The list only has their names, so
// the addresses are loaded for the one selected.
type IPSetsTable struct {
	*core.SelectableTable[types.IPSetSummary]
	data          []types.IPSetSummary
	selectedIPSet types.IPSetSummary
	serviceCtx    *core.ServiceContext[awsapi.WafApi]
}

func NewIPSetsTable(
	serviceContext *core.ServiceContext[awsapi.WafApi],
) *IPSetsTable {
	var table = &IPSetsTable{
		SelectableTable: core.NewSelectableTable[types.IPSetSummary](
			"IP Sets",
			core.TableRow{
				"Name",
				"Description",
				"Id",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateIPSetsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *IPSetsTable) populateIPSetsTable() {
	var tableData []core.TableRow
	for _, set := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(set.Name),
			aws.ToString(set.Description),
			aws.ToString(set.Id),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedIPSet = types.IPSetSummary{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshIPSets loads the IP sets
func (inst *IPSetsTable) RefreshIPSets() {
	inst.refreshIPSets(func() {})
}

func (inst *IPSetsTable) refreshIPSets(loaded func()) {
	var sets []types.IPSetSummary
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		sets, err = inst.serviceCtx.Api.ListIPSets(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = sets
		inst.populateIPSetsTable()
		loaded()
	})
}

// Open selects the IP set of the ARN. The IP sets are loaded again first if it
// isn't one of those loaded.
func (inst *IPSetsTable) Open(ipSetArn string) {
	if inst.selectIPSet(ipSetArn) {
		return
	}

	inst.refreshIPSets(func() {
		if !inst.selectIPSet(ipSetArn) {
			inst.ErrorMessageCallback("IP set %s not found", ipSetArn)
		}
	})
}

// selectIPSet shows all the IP sets, without a search, to select the one with
// the ARN
func (inst *IPSetsTable) selectIPSet(ipSetArn string) bool {
	var matches = func(set types.IPSetSummary) bool { return aws.ToString(set.ARN) == ipSetArn }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateIPSetsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *IPSetsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedIPSet = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *IPSetsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshIPSets()
			return nil
		}
		return capture(event)
	})
}

func (inst *IPSetsTable) GetSelectedIPSet() types.IPSetSummary {
	return inst.selectedIPSet
}
