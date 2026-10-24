package eventbridge

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

var rulesHeadings = core.TableRow{"Name", "State", "Schedule"}

func newRule(name string) types.Rule {
	return types.Rule{
		Name:         aws.String(name),
		State:        types.RuleStateEnabled,
		EventPattern: aws.String(`{"source":["` + name + `.service"]}`),
	}
}

func newScheduledRule(name string, schedule string) types.Rule {
	return types.Rule{
		Name:               aws.String(name),
		State:              types.RuleStateDisabled,
		ScheduleExpression: aws.String(schedule),
	}
}

func newTarget(id string) types.Target {
	return types.Target{
		Id:  aws.String(id),
		Arn: aws.String("arn:aws:lambda:us-east-1:123456789012:function:" + id),
	}
}

func newPopulatedRulesTable(rules ...types.Rule) *RulesTable {
	var table = NewRulesTable(uitest.NewServiceContext[awsapi.EventBridgeApi]())
	table.populateRulesTable(rules)
	return table
}

func TestRulesTable__ShowsStateAndScheduleForEachRule(t *testing.T) {
	var table = newPopulatedRulesTable(newRule("orders"), newScheduledRule("nightly", "rate(1 day)"))

	uitest.AssertTableRows(t, table, []core.TableRow{
		rulesHeadings,
		{"orders", "ENABLED", ""},
		{"nightly", "DISABLED", "rate(1 day)"},
	})
}

func TestRulesTable__SelectingRowTracksSelectedRule(t *testing.T) {
	var table = newPopulatedRulesTable(newRule("orders"), newRule("billing"))

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedRule().Name); got != "billing" {
		t.Errorf("selected rule = %q, want %q", got, "billing")
	}
}

func TestRulesTable__RefreshLoadsRulesForBusFromApi(t *testing.T) {
	var api = &fakeEventBridgeApi{rules: []types.Rule{newRule("orders")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var table = NewRulesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshRules("arn:aws:events:us-east-1:123456789012:event-bus/orders")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		rulesHeadings,
		{"orders", "ENABLED", ""},
	})
}

func TestRulesTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeEventBridgeApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var table = NewRulesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshRules("arn:aws:events:us-east-1:123456789012:event-bus/orders")

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestExtractRuleDetailJson__IncludesPatternAndTargetsAsJson(t *testing.T) {
	var detail = newRuleDetail(newRule("orders"), []types.Target{newTarget("handler")})

	var decoded map[string]any
	if err := json.Unmarshal([]byte(extractRuleDetailJson(detail)), &decoded); err != nil {
		t.Fatalf("output is not valid json: %v", err)
	}

	var pattern = decoded["EventPattern"].(map[string]any)
	if got := pattern["source"].([]any)[0]; got != "orders.service" {
		t.Errorf("pattern source = %v, want orders.service", got)
	}
	var target = decoded["Targets"].([]any)[0].(map[string]any)
	if got := target["Arn"]; got != "arn:aws:lambda:us-east-1:123456789012:function:handler" {
		t.Errorf("target arn = %v", got)
	}
	if _, ok := target["RoleArn"]; ok {
		t.Error("empty RoleArn should be omitted")
	}
}

func TestExtractRuleDetailJson__ScheduledRuleOmitsPatternAndKeepsEmptyTargets(t *testing.T) {
	var detail = newRuleDetail(newScheduledRule("nightly", "rate(1 day)"), nil)

	var text = extractRuleDetailJson(detail)

	if strings.Contains(text, "EventPattern") {
		t.Errorf("scheduled rule should have no EventPattern: %s", text)
	}
	if !strings.Contains(text, `"ScheduleExpression":"rate(1 day)"`) || !strings.Contains(text, `"Targets":[]`) {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestNewRuleDetail__InvalidPatternIsDropped(t *testing.T) {
	var rule = types.Rule{Name: aws.String("bad"), EventPattern: aws.String("{not json")}

	var text = extractRuleDetailJson(newRuleDetail(rule, nil))

	if strings.Contains(text, "EventPattern") {
		t.Errorf("invalid pattern should be dropped: %s", text)
	}
}

func newRunningRulesTabView(t *testing.T, api awsapi.EventBridgeApi) *RulesTabView {
	var serviceCtx = uitest.NewServiceContextWith(api)
	var view = NewRulesTabView(NewRulesTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestRulesTabView__SelectedRuleShowsPatternAndTargets(t *testing.T) {
	var api = &fakeEventBridgeApi{
		rules:   []types.Rule{newRule("orders")},
		targets: []types.Target{newTarget("handler")},
	}
	var view = newRunningRulesTabView(t, api)

	view.RefreshRules("arn:aws:events:us-east-1:123456789012:event-bus/orders")

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		var text = view.JsonView.TextView.GetText()
		return strings.Contains(text, "orders.service") && strings.Contains(text, "function:handler")
	})
}

func TestRulesTabView__TargetsErrorIsReported(t *testing.T) {
	var api = &fakeEventBridgeApi{
		rules:      []types.Rule{newRule("orders")},
		targetsErr: errors.New("throttled"),
	}
	var view = newRunningRulesTabView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.RulesTable.ErrorMessageCallback = callback

	view.RefreshRules("arn:aws:events:us-east-1:123456789012:event-bus/orders")

	if got := uitest.AwaitError(t, received); got != "throttled" {
		t.Errorf("message = %q, want %q", got, "throttled")
	}
}

func TestRulesTabView__NoRulesLeavesJsonViewEmpty(t *testing.T) {
	var view = newRunningRulesTabView(t, &fakeEventBridgeApi{})

	view.RefreshRules("arn:aws:events:us-east-1:123456789012:event-bus/orders")

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return view.RulesTable.GetTable().GetRowCount() == 1 && view.JsonView.TextView.GetText() == ""
	})
}
