package eventbridge

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type RulesTable struct {
	*core.SelectableTable[types.Rule]
	data         []types.Rule
	selectedRule types.Rule
	busArn       string
	serviceCtx   *core.ServiceContext[awsapi.EventBridgeApi]
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
		data:         nil,
		selectedRule: types.Rule{},
		serviceCtx:   serviceCtx,
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
	inst.Select(1, 0)
}

func (inst *RulesTable) RefreshRules(busArn string) {
	inst.busArn = busArn
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var rules []types.Rule

	dataLoader.AsyncLoadData(func() {
		if len(busArn) == 0 {
			return
		}

		var err error
		rules, err = inst.serviceCtx.Api.ListRules(inst.serviceCtx.Ctx, true, busArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if busArn != inst.busArn {
			return
		}
		inst.data = rules
		inst.selectedRule = types.Rule{}
		inst.populateRulesTable(inst.data)
	})
}

func (inst *RulesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedRule = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *RulesTable) GetSelectedRule() types.Rule {
	return inst.selectedRule
}
