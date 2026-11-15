package waf

import (
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// IPSetAddressesTable shows the addresses of the loaded IP set, which can hold
// thousands, so they can be searched
type IPSetAddressesTable struct {
	*core.SelectableTable[string]
}

func NewIPSetAddressesTable(serviceCtx *core.ServiceContext[awsapi.WafApi]) *IPSetAddressesTable {
	var table = &IPSetAddressesTable{
		SelectableTable: core.NewSelectableTable[string](
			"Addresses",
			core.TableRow{"Address"},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0),
	}
	table.SetIPSet(nil)
	return table
}

// SetIPSet shows the addresses of the IP set, or none while it isn't loaded
func (inst *IPSetAddressesTable) SetIPSet(ipSet *types.IPSet) {
	var addresses []string
	var name = ""
	if ipSet != nil {
		addresses = slices.Sorted(slices.Values(ipSet.Addresses))
		name = aws.ToString(ipSet.Name)
	}

	var tableData []core.TableRow
	for _, address := range addresses {
		tableData = append(tableData, core.TableRow{address})
	}

	inst.SetTitleExtra(name)
	inst.SetData(tableData, addresses, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}
