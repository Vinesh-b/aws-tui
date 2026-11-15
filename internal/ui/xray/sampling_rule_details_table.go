package xray

import (
	"maps"
	"slices"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

// SamplingRuleDetailsTable shows a sampling rule. The list holds all of each
// rule, so showing one needs no request.
type SamplingRuleDetailsTable struct {
	*core.DetailsTable
	record types.SamplingRuleRecord
}

func NewSamplingRuleDetailsTable(appCtx *core.AppContext) *SamplingRuleDetailsTable {
	var table = &SamplingRuleDetailsTable{
		DetailsTable: core.NewDetailsTable("Sampling Rule Details", appCtx),
	}
	table.populateDetailsTable()
	return table
}

func (inst *SamplingRuleDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.record.SamplingRule != nil {
		var rule = *inst.record.SamplingRule
		tableData = []core.TableRow{
			{"Name", aws.ToString(rule.RuleName)},
			{"Priority", core.FormatOptionalInt(rule.Priority)},
			{"Reservoir", core.FormatOptionalInt(&rule.ReservoirSize)},
			{"Rate", formatRate(rule.FixedRate)},
			{"Service Name", aws.ToString(rule.ServiceName)},
			{"Service Type", aws.ToString(rule.ServiceType)},
			{"Host", aws.ToString(rule.Host)},
			{"HTTP Method", aws.ToString(rule.HTTPMethod)},
			{"URL Path", aws.ToString(rule.URLPath)},
			{"Resource ARN", aws.ToString(rule.ResourceARN)},
			{"Created", core.FormatTime(inst.record.CreatedAt)},
			{"Modified", core.FormatTime(inst.record.ModifiedAt)},
			{"ARN", aws.ToString(rule.RuleARN)},
		}

		var attributes []core.TableRow
		for _, key := range slices.Sorted(maps.Keys(rule.Attributes)) {
			attributes = append(attributes, core.TableRow{key, rule.Attributes[key]})
		}
		tableData = append(tableData, core.DetailsSection("Attributes", attributes...)...)

		rawData = inst.record
	}

	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetRule shows the sampling rule
func (inst *SamplingRuleDetailsTable) SetRule(record types.SamplingRuleRecord) {
	inst.record = record
	inst.populateDetailsTable()
}
