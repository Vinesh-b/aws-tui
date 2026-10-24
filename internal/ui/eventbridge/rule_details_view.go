package eventbridge

import (
	"encoding/json"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type RuleTarget struct {
	Id               string                  `json:"Id"`
	Arn              string                  `json:"Arn"`
	RoleArn          string                  `json:"RoleArn,omitempty"`
	Input            string                  `json:"Input,omitempty"`
	InputPath        string                  `json:"InputPath,omitempty"`
	InputTransformer *types.InputTransformer `json:"InputTransformer,omitempty"`
	DeadLetterConfig *types.DeadLetterConfig `json:"DeadLetterConfig,omitempty"`
	RetryPolicy      *types.RetryPolicy      `json:"RetryPolicy,omitempty"`
}

func newRuleTargets(targets []types.Target) []RuleTarget {
	var result = []RuleTarget{}
	for _, target := range targets {
		result = append(result, RuleTarget{
			Id:               aws.ToString(target.Id),
			Arn:              aws.ToString(target.Arn),
			RoleArn:          aws.ToString(target.RoleArn),
			Input:            aws.ToString(target.Input),
			InputPath:        aws.ToString(target.InputPath),
			InputTransformer: target.InputTransformer,
			DeadLetterConfig: target.DeadLetterConfig,
			RetryPolicy:      target.RetryPolicy,
		})
	}
	return result
}

type RuleDetail struct {
	Name               string          `json:"Name"`
	State              string          `json:"State"`
	Description        string          `json:"Description,omitempty"`
	ScheduleExpression string          `json:"ScheduleExpression,omitempty"`
	EventPattern       json.RawMessage `json:"EventPattern,omitempty"`
	Targets            []RuleTarget    `json:"Targets"`
}

func newRuleDetail(rule types.Rule, targets []types.Target) RuleDetail {
	var pattern = aws.ToString(rule.EventPattern)
	if !json.Valid([]byte(pattern)) {
		pattern = ""
	}

	return RuleDetail{
		Name:               aws.ToString(rule.Name),
		State:              string(rule.State),
		Description:        aws.ToString(rule.Description),
		ScheduleExpression: aws.ToString(rule.ScheduleExpression),
		EventPattern:       json.RawMessage(pattern),
		Targets:            newRuleTargets(targets),
	}
}

func extractRuleDetailJson(detail RuleDetail) string {
	var jsonBytes, err = json.Marshal(detail)
	if err != nil {
		return err.Error()
	}
	return string(jsonBytes)
}

type RulesTabView struct {
	*tview.Flex
	RulesTable *RulesTable
	JsonView   *core.JsonTextView[RuleDetail]
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi]
	busArn     string
}

func NewRulesTabView(
	rulesTable *RulesTable,
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *RulesTabView {
	var jsonView = &core.JsonTextView[RuleDetail]{
		TextView:        core.NewSearchableTextView("Rule", serviceCtx.AppContext),
		ExtractTextFunc: extractRuleDetailJson,
	}

	var view = &RulesTabView{
		Flex: tview.NewFlex().
			AddItem(rulesTable, 0, 1, true).
			AddItem(jsonView.TextView, 0, 2, false),
		RulesTable: rulesTable,
		JsonView:   jsonView,
		serviceCtx: serviceCtx,
	}

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == core.APP_KEY_BINDINGS.ViewFocusRight && rulesTable.HasFocus():
			serviceCtx.App.SetFocus(jsonView.TextView)
			return nil
		case event.Key() == core.APP_KEY_BINDINGS.ViewFocusLeft && jsonView.TextView.HasFocus():
			serviceCtx.App.SetFocus(rulesTable)
			return nil
		}
		return event
	})

	rulesTable.SetSelectionChangedFunc(func(row, column int) {
		view.RefreshRuleDetail()
	})

	return view
}

func (inst *RulesTabView) RefreshRules(busArn string) {
	inst.busArn = busArn
	inst.JsonView.TextView.SetText("", false)
	inst.RulesTable.RefreshRules(busArn)
}

func (inst *RulesTabView) RefreshRuleDetail() {
	var rule = inst.RulesTable.GetSelectedRule()
	var busArn = inst.busArn
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var targets []types.Target

	dataLoader.AsyncLoadData(func() {
		if len(aws.ToString(rule.Name)) == 0 {
			return
		}

		var err error
		targets, err = inst.serviceCtx.Api.ListTargetsByRule(
			inst.serviceCtx.Ctx, true, busArn, aws.ToString(rule.Name),
		)
		if err != nil {
			inst.RulesTable.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.JsonView.TextView, func() {
		var selected = inst.RulesTable.GetSelectedRule()
		if aws.ToString(selected.Name) != aws.ToString(rule.Name) {
			return
		}
		if len(aws.ToString(rule.Name)) == 0 {
			inst.JsonView.TextView.SetText("", false)
			return
		}
		inst.JsonView.SetText(newRuleDetail(rule, targets))
	})
}
