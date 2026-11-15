package waf

import (
	"cmp"
	"slices"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// WebACLRulesTable shows the rules of the loaded web ACL in the order they are
// evaluated
type WebACLRulesTable struct {
	*core.SelectableTable[types.Rule]
}

func NewWebACLRulesTable(serviceCtx *core.ServiceContext[awsapi.WafApi]) *WebACLRulesTable {
	var table = &WebACLRulesTable{
		SelectableTable: core.NewSelectableTable[types.Rule](
			"Rules",
			core.TableRow{
				"Priority",
				"Name",
				"Action",
				"Statement",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(3, 1).SetFilterColumns(1, 2, 3),
	}
	table.SetWebACL(nil)
	return table
}

// SetWebACL shows the rules of the web ACL, or none while it isn't loaded
func (inst *WebACLRulesTable) SetWebACL(webAcl *types.WebACL) {
	var rules []types.Rule
	var name = ""
	if webAcl != nil {
		rules = slices.Clone(webAcl.Rules)
		name = aws.ToString(webAcl.Name)
	}
	slices.SortStableFunc(rules, func(a, b types.Rule) int {
		return cmp.Compare(a.Priority, b.Priority)
	})

	var tableData []core.TableRow
	for _, rule := range rules {
		tableData = append(tableData, core.TableRow{
			strconv.Itoa(int(rule.Priority)),
			aws.ToString(rule.Name),
			ruleAction(rule),
			statementSummary(rule.Statement),
		})
	}

	inst.SetTitleExtra(name)
	inst.SetData(tableData, rules, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}
