package xray

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"

	"github.com/gdamore/tcell/v2"
)

func samplingRule(record types.SamplingRuleRecord) types.SamplingRule {
	if record.SamplingRule == nil {
		return types.SamplingRule{}
	}
	return *record.SamplingRule
}

// formatRate is the fixed rate of a rule, which is a fraction, as a percentage
func formatRate(rate float64) string {
	var percent = rate * 100
	return core.FormatOptionalFloat(&percent) + "%"
}

type SamplingRulesTable struct {
	*core.SelectableTable[types.SamplingRuleRecord]
	data         []types.SamplingRuleRecord
	selectedRule types.SamplingRuleRecord
	serviceCtx   *core.ServiceContext[awsapi.XRayApi]
}

func NewSamplingRulesTable(
	serviceContext *core.ServiceContext[awsapi.XRayApi],
) *SamplingRulesTable {
	var table = &SamplingRulesTable{
		SelectableTable: core.NewSelectableTable[types.SamplingRuleRecord](
			"Sampling Rules",
			core.TableRow{
				"Name",
				"Priority",
				"Reservoir",
				"Rate",
				"Service Name",
				"URL Path",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 4, 5),
		serviceCtx: serviceContext,
	}

	table.populateRulesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *SamplingRulesTable) populateRulesTable() {
	var tableData []core.TableRow
	for _, record := range inst.data {
		var rule = samplingRule(record)
		tableData = append(tableData, core.TableRow{
			aws.ToString(rule.RuleName),
			core.FormatOptionalInt(rule.Priority),
			core.FormatOptionalInt(&rule.ReservoirSize),
			formatRate(rule.FixedRate),
			aws.ToString(rule.ServiceName),
			aws.ToString(rule.URLPath),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedRule = types.SamplingRuleRecord{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshRules loads the sampling rules
func (inst *SamplingRulesTable) RefreshRules(force bool) {
	inst.refreshRules(force, func() {})
}

func (inst *SamplingRulesTable) refreshRules(force bool, loaded func()) {
	var rules []types.SamplingRuleRecord
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		rules, err = inst.serviceCtx.Api.GetSamplingRules(ctx, force)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = rules
		inst.populateRulesTable()
		loaded()
	})
}

// Open selects the rule of the ARN. The rules are loaded again first if it
// isn't one of those loaded.
func (inst *SamplingRulesTable) Open(ruleArn string) {
	if inst.selectRule(ruleArn) {
		return
	}

	inst.refreshRules(true, func() {
		if !inst.selectRule(ruleArn) {
			inst.ErrorMessageCallback("Sampling rule %s not found", ruleArn)
		}
	})
}

// selectRule shows all the rules, without a search, to select the one with
// the ARN
func (inst *SamplingRulesTable) selectRule(ruleArn string) bool {
	var matches = func(r types.SamplingRuleRecord) bool {
		return aws.ToString(samplingRule(r).RuleARN) == ruleArn
	}
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateRulesTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *SamplingRulesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedRule = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SamplingRulesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshRules(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *SamplingRulesTable) GetSelectedRule() types.SamplingRuleRecord {
	return inst.selectedRule
}
