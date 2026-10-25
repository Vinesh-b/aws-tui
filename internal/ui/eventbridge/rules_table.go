package eventbridge

import (
	"sync"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

// Limits the concurrent ListTargetsByRule calls made while preloading rule details
const maxConcurrentTargetLoads = 5

type RulesTable struct {
	*core.SelectableTable[types.Rule]
	data            []types.Rule
	ruleDetails     map[string]RuleDetail
	selectedRule    types.Rule
	lastSelectedRow int
	busArn          string
	serviceCtx      *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewRulesTable(
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *RulesTable {
	var view = &RulesTable{
		SelectableTable: core.NewSelectableTable[types.Rule](
			"Rules",
			core.TableRow{
				"Name",
				"State",
				"Schedule",
			},
			serviceCtx.AppContext,
		),
		data:            nil,
		ruleDetails:     map[string]RuleDetail{},
		selectedRule:    types.Rule{},
		lastSelectedRow: 0,
		serviceCtx:      serviceCtx,
	}

	view.populateRulesTable(view.data)
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *RulesTable) populateRulesTable(data []types.Rule) {
	var tableData []core.TableRow
	for _, rule := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(rule.Name),
			string(rule.State),
			aws.ToString(rule.ScheduleExpression),
		})
	}

	inst.SetData(tableData, data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.lastSelectedRow = 0
	inst.Select(1, 0)
}

func (inst *RulesTable) loadRuleDetails(busArn string, rules []types.Rule) map[string]RuleDetail {
	var details = make(map[string]RuleDetail, len(rules))
	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	var semaphore = make(chan struct{}, maxConcurrentTargetLoads)
	var firstErr error

	for _, rule := range rules {
		waitGroup.Add(1)
		go func(rule types.Rule) {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var targets, err = inst.serviceCtx.Api.ListTargetsByRule(
				inst.serviceCtx.Ctx, true, busArn, aws.ToString(rule.Name),
			)

			mutex.Lock()
			defer mutex.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			details[aws.ToString(rule.Name)] = newRuleDetail(rule, targets)
		}(rule)
	}
	waitGroup.Wait()

	if firstErr != nil {
		inst.ErrorMessageCallback("%s", firstErr.Error())
	}

	return details
}

func (inst *RulesTable) RefreshRules(busArn string) {
	inst.busArn = busArn
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 30)
	var rules []types.Rule
	var details map[string]RuleDetail

	dataLoader.AsyncLoadData(func() {
		if len(busArn) == 0 {
			return
		}

		var err error
		rules, err = inst.serviceCtx.Api.ListRules(inst.serviceCtx.Ctx, true, busArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}

		details = inst.loadRuleDetails(busArn, rules)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if busArn != inst.busArn {
			return
		}
		inst.data = rules
		inst.ruleDetails = details
		inst.selectedRule = types.Rule{}
		inst.populateRulesTable(inst.data)
	})
}

// The handler is only called when the selected row changes, as the rule
// details are preloaded and do not need refreshing on every selection event
func (inst *RulesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 || row == inst.lastSelectedRow {
			return
		}
		inst.lastSelectedRow = row
		inst.selectedRule = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *RulesTable) GetSelectedRule() types.Rule {
	return inst.selectedRule
}

func (inst *RulesTable) GetSelectedRuleDetail() (RuleDetail, bool) {
	var detail, ok = inst.ruleDetails[aws.ToString(inst.selectedRule.Name)]
	return detail, ok
}
