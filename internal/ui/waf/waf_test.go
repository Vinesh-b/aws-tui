package waf

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
	"aws-tui/internal/ui/waf/waffake"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var webAclHeadings = core.TableRow{"Name", "Description", "Id"}
var ruleHeadings = core.TableRow{"Priority", "Name", "Action", "Statement"}
var resourceHeadings = core.TableRow{"Resource Type", "ARN"}

const albArn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/shop/50dc6c495c0c9188"
const stageArn = "arn:aws:apigateway:us-east-1::/restapis/a1b2c3/stages/prod"

func webAclArnOf(name string) string {
	return "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/" + name + "/id-" + name
}

func ipSetArnOf(name string) string {
	return "arn:aws:wafv2:us-east-1:123456789012:regional/ipset/" + name + "/id-" + name
}

func newWebACLSummary(name string) types.WebACLSummary {
	return types.WebACLSummary{
		Name:        aws.String(name),
		Description: aws.String("Protects " + name),
		Id:          aws.String("id-" + name),
		ARN:         aws.String(webAclArnOf(name)),
	}
}

func newWebACL(name string, rules ...types.Rule) types.WebACL {
	return types.WebACL{
		Name:           aws.String(name),
		Description:    aws.String("Protects " + name),
		Id:             aws.String("id-" + name),
		ARN:            aws.String(webAclArnOf(name)),
		DefaultAction:  &types.DefaultAction{Allow: &types.AllowAction{}},
		Capacity:       725,
		LabelNamespace: aws.String("awswaf:123456789012:webacl:" + name + ":"),
		VisibilityConfig: &types.VisibilityConfig{
			MetricName:               aws.String(name),
			CloudWatchMetricsEnabled: true,
			SampledRequestsEnabled:   true,
		},
		Rules: rules,
	}
}

func ipSetRule(name string, priority int32, ipSet string) types.Rule {
	return types.Rule{
		Name:     aws.String(name),
		Priority: priority,
		Action:   &types.RuleAction{Block: &types.BlockAction{}},
		Statement: &types.Statement{
			IPSetReferenceStatement: &types.IPSetReferenceStatement{ARN: aws.String(ipSetArnOf(ipSet))},
		},
	}
}

func newIPSetSummary(name string) types.IPSetSummary {
	return types.IPSetSummary{
		Name:        aws.String(name),
		Description: aws.String("Addresses of " + name),
		Id:          aws.String("id-" + name),
		ARN:         aws.String(ipSetArnOf(name)),
	}
}

func TestStatementSummary(t *testing.T) {
	var geo = types.Statement{GeoMatchStatement: &types.GeoMatchStatement{CountryCodes: []types.CountryCode{"CN", "RU"}}}
	var label = types.Statement{LabelMatchStatement: &types.LabelMatchStatement{Key: aws.String("awswaf:managed:bot")}}
	var sqli = types.Statement{SqliMatchStatement: &types.SqliMatchStatement{}}

	for _, test := range []struct {
		name      string
		statement *types.Statement
		want      string
	}{
		{"none", nil, ""},
		{"managed rule group", &types.Statement{ManagedRuleGroupStatement: &types.ManagedRuleGroupStatement{
			VendorName: aws.String("AWS"), Name: aws.String("AWSManagedRulesBotControlRuleSet"), Version: aws.String("Version_3.0"),
		}}, "Managed rule group AWS/AWSManagedRulesBotControlRuleSet Version_3.0"},
		{"rule group", &types.Statement{RuleGroupReferenceStatement: &types.RuleGroupReferenceStatement{
			ARN: aws.String("arn:aws:wafv2:us-east-1:123456789012:regional/rulegroup/shared/id-shared"),
		}}, "Rule group shared"},
		{"regex pattern set", &types.Statement{RegexPatternSetReferenceStatement: &types.RegexPatternSetReferenceStatement{
			ARN: aws.String("arn:aws:wafv2:us-east-1:123456789012:regional/regexpatternset/paths/id-paths"),
		}}, "Regex pattern set paths"},
		{"IP set with an ARN that isn't of WAF", &types.Statement{IPSetReferenceStatement: &types.IPSetReferenceStatement{
			ARN: aws.String("not-an-arn"),
		}}, "IP set not-an-arn"},
		{"rate in the default window", &types.Statement{RateBasedStatement: &types.RateBasedStatement{
			Limit: aws.Int64(2000), AggregateKeyType: types.RateBasedStatementAggregateKeyTypeIp,
		}}, "Rate limit 2000 per 300s by IP"},
		{"rate scoped down", &types.Statement{RateBasedStatement: &types.RateBasedStatement{
			Limit: aws.Int64(100), EvaluationWindowSec: 60,
			AggregateKeyType:   types.RateBasedStatementAggregateKeyTypeForwardedIp,
			ScopeDownStatement: &geo,
		}}, "Rate limit 100 per 60s by forwarded IP where Country is CN, RU"},
		{"ASN", &types.Statement{AsnMatchStatement: &types.AsnMatchStatement{AsnList: []int64{64496, 64511}}}, "ASN is 64496, 64511"},
		{"label", &label, "Has label awswaf:managed:bot"},
		{"and with a nested or", &types.Statement{AndStatement: &types.AndStatement{Statements: []types.Statement{
			label,
			{OrStatement: &types.OrStatement{Statements: []types.Statement{geo, sqli}}},
		}}}, "Has label awswaf:managed:bot AND (Country is CN, RU OR SQL injection)"},
		{"not", &types.Statement{NotStatement: &types.NotStatement{Statement: &geo}}, "NOT Country is CN, RU"},
		{"not of an and", &types.Statement{NotStatement: &types.NotStatement{Statement: &types.Statement{
			AndStatement: &types.AndStatement{Statements: []types.Statement{geo, label}},
		}}}, "NOT (Country is CN, RU AND Has label awswaf:managed:bot)"},
		{"cross-site scripting", &types.Statement{XssMatchStatement: &types.XssMatchStatement{}}, "Cross-site scripting"},
	} {
		if got := statementSummary(test.statement); got != test.want {
			t.Errorf("%s: statementSummary = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestRuleAction(t *testing.T) {
	for _, test := range []struct {
		name string
		rule types.Rule
		want string
	}{
		{"allow", types.Rule{Action: &types.RuleAction{Allow: &types.AllowAction{}}}, "Allow"},
		{"captcha", types.Rule{Action: &types.RuleAction{Captcha: &types.CaptchaAction{}}}, "CAPTCHA"},
		{"challenge", types.Rule{Action: &types.RuleAction{Challenge: &types.ChallengeAction{}}}, "Challenge"},
		{"group counted", types.Rule{OverrideAction: &types.OverrideAction{Count: &types.CountAction{}}}, "Override to Count"},
		{"group as it is", types.Rule{OverrideAction: &types.OverrideAction{None: &types.NoneAction{}}}, "Use Rule Actions"},
		{"none", types.Rule{}, ""},
	} {
		if got := ruleAction(test.rule); got != test.want {
			t.Errorf("%s: ruleAction = %q, want %q", test.name, got, test.want)
		}
	}
}

// summaryRows is how many rows the details have from the list, before what
// is loaded
const summaryRows = 4

// awaitDetailsLoaded waits for the details to have more than the list gives,
// as AwaitDetailsRows is already satisfied by the rows from the list
func awaitDetailsLoaded(t *testing.T, app *tview.Application, table *core.DetailsTable) {
	t.Helper()
	uitest.Eventually(t, app, func() bool { return table.GetCellText(summaryRows, 0) != "" },
		"the details to be loaded")
}

func newWebACLsTest(t *testing.T, api *waffake.Api) *WebACLsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.WafApi](api)
	var view = NewWebACLsPageView(
		NewWebACLsTable(serviceCtx),
		NewWebACLDetailsTable(serviceCtx),
		NewWebACLRulesTable(serviceCtx),
		NewWebACLResourcesTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

// loadWebACLs loads the web ACLs of the page and waits for them
func loadWebACLs(t *testing.T, view *WebACLsPageView, count int) {
	t.Helper()
	var app = view.WebACLsTable.serviceCtx.App
	view.WebACLsTable.RefreshWebACLs()
	uitest.Eventually(t, app, func() bool {
		return view.WebACLsTable.GetTable().GetRowCount() == count+1
	}, "the web ACLs to be loaded")
}

// openResourcesTab moves from the details tab, past the rules, to the
// associated resources
func openResourcesTab(t *testing.T, view *WebACLsPageView) {
	t.Helper()
	var app = view.WebACLsTable.serviceCtx.App
	uitest.OnUI(t, app, func() { app.SetFocus(view.DetailsTable) })
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.OnUI(t, app, func() {
		if name, _ := view.TabView.GetCurrentTab(); name != WebACLTabNameResources {
			t.Fatalf("tab = %q, want %q", name, WebACLTabNameResources)
		}
	})
}

func TestWebACLsTable__ResetLoadsTheWebACLs(t *testing.T) {
	var api = waffake.New(waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("api"), newWebACLSummary("shop")}))
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.WebACLsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.WebACLsTable, []core.TableRow{
		webAclHeadings,
		{"api", "Protects api", "id-api"},
		{"shop", "Protects shop", "id-shop"},
	})
}

func TestWebACLsTable__ErrorIsShown(t *testing.T) {
	var api = waffake.New(waffake.WithErr(errors.New("AccessDeniedException: wafv2:ListWebACLs")))
	var view = newWebACLsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.WebACLsTable.serviceCtx.App, func() { view.WebACLsTable.ErrorMessageCallback = callback })

	view.WebACLsTable.RefreshWebACLs()

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "wafv2:ListWebACLs") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestWebACLsPage__SelectingAWebACLShowsItThenItsDetailsAndRulesInOrder(t *testing.T) {
	var api = waffake.New(
		waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("shop")}),
		waffake.WithWebACL(webAclArnOf("shop"), newWebACL("shop",
			types.Rule{
				Name:     aws.String("rate-limit"),
				Priority: 2,
				Action:   &types.RuleAction{Count: &types.CountAction{}},
				Statement: &types.Statement{RateBasedStatement: &types.RateBasedStatement{
					Limit: aws.Int64(2000), AggregateKeyType: types.RateBasedStatementAggregateKeyTypeIp,
				}},
			},
			ipSetRule("block-bad-ips", 0, "blocklist"),
			types.Rule{
				Name:           aws.String("aws-common"),
				Priority:       1,
				OverrideAction: &types.OverrideAction{None: &types.NoneAction{}},
				Statement: &types.Statement{ManagedRuleGroupStatement: &types.ManagedRuleGroupStatement{
					VendorName: aws.String("AWS"), Name: aws.String("AWSManagedRulesCommonRuleSet"),
				}},
			},
		)),
	)
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App

	view.WebACLsTable.RefreshWebACLs()

	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable)
	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "shop"},
		{"Description", "Protects shop"},
		{"Id", "id-shop"},
		{"ARN", webAclArnOf("shop")},
		{"Default Action", "Allow"},
		{"Rules", "3"},
		{"Capacity (WCUs)", "725"},
		{"Label Namespace", "awswaf:123456789012:webacl:shop:"},
		{"Token Domains", ""},
		{"Firewall Manager", "false"},
		{"Visibility"},
		{"Metric Name", "shop"},
		{"CloudWatch Metrics", "true"},
		{"Sampled Requests", "true"},
	})
	uitest.AwaitRows(t, app, view.RulesTable, []core.TableRow{
		ruleHeadings,
		{"0", "block-bad-ips", "Block", "IP set blocklist"},
		{"1", "aws-common", "Use Rule Actions", "Managed rule group AWS/AWSManagedRulesCommonRuleSet"},
		{"2", "rate-limit", "Count", "Rate limit 2000 per 300s by IP"},
	})
}

func TestWebACLsPage__RulesOfTheWebACLSelectedBeforeAreNotKept(t *testing.T) {
	var api = waffake.New(
		waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("api"), newWebACLSummary("shop")}),
		waffake.WithWebACL(webAclArnOf("api"), newWebACL("api", ipSetRule("block-bad-ips", 0, "blocklist"))),
		waffake.WithWebACL(webAclArnOf("shop"), newWebACL("shop", ipSetRule("block-office", 0, "office"))),
	)
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.DetailsTable.ErrorMessageCallback = callback })
	loadWebACLs(t, view, 2)
	uitest.Eventually(t, app, func() bool { return view.RulesTable.GetTable().GetRowCount() == 2 },
		"the rules of api to be shown")
	// shop can't be loaded, so only the rules of api could be shown for it
	api.SetErr(errors.New("WAFUnavailableEntityException: shop"))

	uitest.OnUI(t, app, func() { view.WebACLsTable.Select(2, 0) })

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "WAFUnavailableEntityException") {
		t.Errorf("error = %q, want the error of the api", got)
	}
	uitest.OnUI(t, app, func() {
		if rows := view.RulesTable.GetTable().GetRowCount(); rows != 1 {
			t.Errorf("rules table has %d rows, want only the headings", rows)
		}
	})
}

func TestWebACLsPage__MovingThroughWebACLsLoadsOnlyTheOneItStopsOn(t *testing.T) {
	var names = []string{"api", "billing", "network", "shop"}
	var summaries []types.WebACLSummary
	var options []waffake.Option
	for _, name := range names {
		summaries = append(summaries, newWebACLSummary(name))
		options = append(options, waffake.WithWebACL(webAclArnOf(name), newWebACL(name)))
	}
	var api = waffake.New(append(options, waffake.WithWebACLs(summaries))...)
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App
	loadWebACLs(t, view, len(names))
	var before = len(api.WebACLsAskedFor())

	uitest.OnUI(t, app, func() {
		for row := 2; row <= 4; row++ {
			view.WebACLsTable.Select(row, 0)
		}
	})
	// The web ACL is shown straight away, before it is loaded
	uitest.OnUI(t, app, func() {
		if got := view.DetailsTable.GetCellText(0, 1); got != "shop" {
			t.Errorf("web ACL shown = %q, want shop", got)
		}
	})

	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.WebACLsAskedFor()[before:], webAclArnOf("shop"))
	}, "shop to be loaded")
	uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
		var asked = api.WebACLsAskedFor()[before:]
		return !slices.Contains(asked, webAclArnOf("network")) && asked[len(asked)-1] == webAclArnOf("shop")
	}, "only the first and last web ACLs of the move to be loaded")
}

func TestWebACLsPage__ResourcesAreOnlyLoadedOnceTheirTabIsOpened(t *testing.T) {
	var api = waffake.New(
		waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("shop")}),
		waffake.WithWebACL(webAclArnOf("shop"), newWebACL("shop")),
		waffake.WithResources(webAclArnOf("shop"), []awsapi.WebACLResource{
			{Type: types.ResourceTypeApplicationLoadBalancer, Arn: albArn},
			{Type: types.ResourceTypeApiGateway, Arn: stageArn},
		}),
	)
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App
	loadWebACLs(t, view, 1)
	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.WebACLsAskedFor(), webAclArnOf("shop"))
	}, "shop to be loaded")
	if asked := api.ResourcesAskedFor(); len(asked) != 0 {
		t.Fatalf("resources loaded before the tab was opened: %v", asked)
	}

	openResourcesTab(t, view)

	uitest.AwaitRows(t, app, view.ResourcesTable, []core.TableRow{
		resourceHeadings,
		{"Application Load Balancer", albArn},
		{"API Gateway Stage", stageArn},
	})
	// Going round the tabs back to it doesn't load them again
	for range 3 {
		uitest.PressKey(t, app, tcell.KeyTab)
	}
	if asked := api.ResourcesAskedFor(); len(asked) != 1 {
		t.Errorf("resources loaded after reopening the tab = %v, want shop once", asked)
	}
}

func TestWebACLsPage__SelectingAnotherWebACLWithTheResourcesOpenLoadsItsResources(t *testing.T) {
	var api = waffake.New(
		waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("api"), newWebACLSummary("shop")}),
		waffake.WithWebACL(webAclArnOf("api"), newWebACL("api")),
		waffake.WithWebACL(webAclArnOf("shop"), newWebACL("shop")),
		waffake.WithResources(webAclArnOf("api"), []awsapi.WebACLResource{{Type: types.ResourceTypeApiGateway, Arn: stageArn}}),
		waffake.WithResources(webAclArnOf("shop"), []awsapi.WebACLResource{{Type: types.ResourceTypeApplicationLoadBalancer, Arn: albArn}}),
	)
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App
	loadWebACLs(t, view, 2)
	openResourcesTab(t, view)
	uitest.AwaitRows(t, app, view.ResourcesTable, []core.TableRow{resourceHeadings, {"API Gateway Stage", stageArn}})

	uitest.OnUI(t, app, func() { view.WebACLsTable.Select(2, 0) })

	uitest.AwaitRows(t, app, view.ResourcesTable, []core.TableRow{resourceHeadings, {"Application Load Balancer", albArn}})
}

func TestWebACLsPage__ResourcesErrorIsShownAndLoadedAgainWhenTheTabIsReopened(t *testing.T) {
	var api = waffake.New(
		waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("shop")}),
		waffake.WithWebACL(webAclArnOf("shop"), newWebACL("shop")),
		waffake.WithResources(webAclArnOf("shop"), []awsapi.WebACLResource{
			{Type: types.ResourceTypeApplicationLoadBalancer, Arn: albArn},
		}),
	)
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App
	loadWebACLs(t, view, 1)
	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.WebACLsAskedFor(), webAclArnOf("shop"))
	}, "shop to be loaded")
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.ResourcesTable.ErrorMessageCallback = callback })
	api.SetErr(errors.New("AccessDeniedException: wafv2:ListResourcesForWebACL"))

	openResourcesTab(t, view)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "wafv2:ListResourcesForWebACL") {
		t.Errorf("error = %q, want the error of the api", got)
	}

	api.SetErr(nil)
	for range 3 {
		uitest.PressKey(t, app, tcell.KeyTab)
	}
	uitest.AwaitRows(t, app, view.ResourcesTable, []core.TableRow{resourceHeadings, {"Application Load Balancer", albArn}})
}

func TestWebACLsTable__OpenLoadsTheWebACLsToSelectOneNotLoaded(t *testing.T) {
	var api = waffake.New(waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("api"), newWebACLSummary("shop")}))
	var view = newWebACLsTest(t, api)
	var app = view.WebACLsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { view.WebACLsTable.Open(webAclArnOf("shop")) })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(view.WebACLsTable.GetSelectedWebACL().Name) == "shop"
	}, "shop to be selected")
}

func TestWebACLsTable__OpenOfAMissingWebACLIsAnError(t *testing.T) {
	var api = waffake.New(waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("api")}))
	var view = newWebACLsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.WebACLsTable.serviceCtx.App, func() {
		view.WebACLsTable.ErrorMessageCallback = callback
		view.WebACLsTable.Open(webAclArnOf("gone"))
	})

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "webacl/gone") {
		t.Errorf("error = %q, want it to name the web ACL", got)
	}
}

func newIPSetsTest(t *testing.T, api *waffake.Api) *IPSetsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.WafApi](api)
	var view = NewIPSetsPageView(
		NewIPSetsTable(serviceCtx),
		NewIPSetDetailsTable(serviceCtx),
		NewIPSetAddressesTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

func TestIPSetsTable__ResetLoadsTheIPSets(t *testing.T) {
	var api = waffake.New(waffake.WithIPSets([]types.IPSetSummary{newIPSetSummary("blocklist"), newIPSetSummary("office")}))
	var view = newIPSetsTest(t, api)
	var app = view.IPSetsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.IPSetsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.IPSetsTable, []core.TableRow{
		{"Name", "Description", "Id"},
		{"blocklist", "Addresses of blocklist", "id-blocklist"},
		{"office", "Addresses of office", "id-office"},
	})
}

func TestIPSetsPage__SelectingAnIPSetShowsItThenItsDetailsAndSortedAddresses(t *testing.T) {
	var api = waffake.New(
		waffake.WithIPSets([]types.IPSetSummary{newIPSetSummary("blocklist")}),
		waffake.WithIPSet(ipSetArnOf("blocklist"), types.IPSet{
			Name:             aws.String("blocklist"),
			ARN:              aws.String(ipSetArnOf("blocklist")),
			IPAddressVersion: types.IPAddressVersionIpv4,
			Addresses:        []string{"203.0.113.0/24", "192.0.2.44/32", "198.51.100.0/24"},
		}),
	)
	var view = newIPSetsTest(t, api)
	var app = view.IPSetsTable.serviceCtx.App

	view.IPSetsTable.RefreshIPSets()

	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable)
	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "blocklist"},
		{"Description", "Addresses of blocklist"},
		{"Id", "id-blocklist"},
		{"ARN", ipSetArnOf("blocklist")},
		{"IP Version", "IPV4"},
		{"Addresses", "3"},
	})
	uitest.AwaitRows(t, app, view.AddressesTable, []core.TableRow{
		{"Address"},
		{"192.0.2.44/32"},
		{"198.51.100.0/24"},
		{"203.0.113.0/24"},
	})
}

func TestIsOpenableWafResource(t *testing.T) {
	for _, test := range []struct {
		resource string
		want     bool
	}{
		{webAclArnOf("shop"), true},
		{ipSetArnOf("office"), true},
		{"arn:aws:wafv2:us-east-1:123456789012:regional/rulegroup/shared/id-shared", false},
		{"arn:aws:wafv2:us-east-1:123456789012:global/webacl/cdn/id-cdn", false},
		{"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/shop", false},
	} {
		var resource, err = arn.Parse(test.resource)
		if err != nil {
			t.Fatal(err)
		}
		if got := isOpenableWafResource(resource); got != test.want {
			t.Errorf("isOpenableWafResource(%q) = %t, want %t", test.resource, got, test.want)
		}
	}
}

func TestHomeView__ResetLoadsTheWebACLs(t *testing.T) {
	var api = waffake.New(waffake.WithWebACLs([]types.WebACLSummary{newWebACLSummary("storefront")}))
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewWafHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "Protects storefront")
}

func TestHomeView__OpeningAnIPSetShowsItOnTheIPSetsPage(t *testing.T) {
	var api = waffake.New(
		waffake.WithIPSets([]types.IPSetSummary{newIPSetSummary("blocklist"), newIPSetSummary("office")}),
		// Loading the list selects blocklist before office is opened
		waffake.WithIPSet(ipSetArnOf("blocklist"), types.IPSet{
			Name: aws.String("blocklist"), IPAddressVersion: types.IPAddressVersionIpv4,
		}),
		waffake.WithIPSet(ipSetArnOf("office"), types.IPSet{
			Name: aws.String("office"), IPAddressVersion: types.IPAddressVersionIpv6,
		}),
	)
	var appCtx = uitest.NewAppContext()
	var page = NewWafHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener, ok = page.(core.ResourceOpener)
	var resource, _ = arn.Parse(ipSetArnOf("office"))
	if !ok || !opener.OpensArn(resource) {
		t.Fatal("WAF page should open IP sets")
	}

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	// The details of office, which only its IP set has, are shown
	uitest.AwaitScreenText(t, appCtx.App, screen, "IPV6")
}
