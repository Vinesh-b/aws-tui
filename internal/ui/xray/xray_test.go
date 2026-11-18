package xray

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/safetest"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
	"aws-tui/internal/ui/xray/xrayfake"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var traceHeadings = core.TableRow{"Start Time", "Entry Point", "Method", "Status", "Response Time", "URL"}

func newSummary(id string, path string, minute int) types.TraceSummary {
	var at = time.Date(2026, 11, 14, 9, minute, 0, 0, time.UTC)
	return types.TraceSummary{
		Id:           aws.String(id),
		StartTime:    &at,
		Duration:     aws.Float64(1.5),
		ResponseTime: aws.Float64(0.125),
		EntryPoint:   &types.ServiceId{Name: aws.String("orders-api"), Type: aws.String("AWS::ApiGateway::Stage")},
		Http: &types.Http{
			HttpMethod: aws.String("POST"),
			HttpURL:    aws.String(path),
			HttpStatus: aws.Int32(502),
			ClientIp:   aws.String("203.0.113.7"),
		},
		HasFault:    aws.Bool(true),
		HasError:    aws.Bool(false),
		HasThrottle: aws.Bool(false),
		IsPartial:   aws.Bool(false),
		Users:       []types.TraceUser{{UserName: aws.String("alice")}},
		ServiceIds: []types.ServiceId{
			{Name: aws.String("orders-api"), Type: aws.String("AWS::ApiGateway::Stage")},
			{Name: aws.String("orders"), Type: aws.String("AWS::Lambda::Function")},
		},
		FaultRootCauses: []types.FaultRootCause{{Services: []types.FaultRootCauseService{{
			Name: aws.String("orders"),
			EntityPath: []types.FaultRootCauseEntity{
				{Name: aws.String("orders")},
				{Name: aws.String("DynamoDB"), Exceptions: []types.RootCauseException{
					{Name: aws.String("ProvisionedThroughputExceededException"), Message: aws.String("Rate exceeded")},
				}},
			},
		}}}},
		ResourceARNs: []types.ResourceARNDetail{
			{ARN: aws.String("arn:aws:lambda:us-east-1:123456789012:function:orders")},
		},
	}
}

func newTrace(id string) types.Trace {
	return types.Trace{
		Id: aws.String(id),
		Segments: []types.Segment{
			{Id: aws.String("s2"), Document: aws.String(
				`{"name":"orders","origin":"AWS::Lambda::Function","start_time":100.25,"end_time":101.5,"fault":true}`,
			)},
			{Id: aws.String("s1"), Document: aws.String(
				`{"name":"orders-api","origin":"AWS::ApiGateway::Stage","start_time":100,"end_time":101.6,` +
					`"http":{"response":{"status":502}}}`,
			)},
			{Id: aws.String("s3"), Document: aws.String("not json")},
		},
	}
}

func TestFormatSeconds(t *testing.T) {
	for _, test := range []struct {
		seconds *float64
		want    string
	}{
		{nil, ""},
		{aws.Float64(0), "0ms"},
		{aws.Float64(0.1234), "123ms"},
		{aws.Float64(1), "1.00s"},
		{aws.Float64(12.345), "12.35s"},
	} {
		if got := formatSeconds(test.seconds); got != test.want {
			t.Errorf("formatSeconds(%v) = %q, want %q", aws.ToFloat64(test.seconds), got, test.want)
		}
	}
}

func TestRootCause(t *testing.T) {
	var throttled = types.RootCauseException{Name: aws.String("ThrottlingException"), Message: aws.String("Rate exceeded")}
	var timedOut = types.RootCauseException{Name: aws.String("TimeoutError")}

	for _, test := range []struct {
		name       string
		exceptions [][]types.RootCauseException
		want       string
	}{
		{"no entities", nil, "Fault"},
		{"entities without exceptions", [][]types.RootCauseException{{}, {}}, "Fault"},
		{"first entity with exceptions", [][]types.RootCauseException{{}, {throttled}, {timedOut}},
			"Fault: ThrottlingException: Rate exceeded"},
		{"first exception of the entity", [][]types.RootCauseException{{timedOut, throttled}}, "Fault: TimeoutError"},
	} {
		if got := rootCause("Fault", test.exceptions); got != test.want {
			t.Errorf("%s: rootCause = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestTracesTable__ResetLoadsTheNewestTraces(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithTracePages(
		[]types.TraceSummary{newSummary("t1", "/orders", 30), newSummary("t2", "/carts", 20)},
	))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var table = NewTracesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() { uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		traceHeadings,
		{"2026-11-14 09:30:00", "orders-api", "POST", "502", "125ms", "/orders"},
		{"2026-11-14 09:20:00", "orders-api", "POST", "502", "125ms", "/carts"},
	})
}

func TestTracesPage__MovingPastTheLastTraceLoadsTheNextPage(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithTracePages(
		[]types.TraceSummary{newSummary("t1", "/orders", 30), newSummary("t2", "/carts", 20)},
		[]types.TraceSummary{newSummary("t3", "/refunds", 10)},
	), xrayfake.WithTrace(newTrace("t1")), xrayfake.WithTrace(newTrace("t2")),
		xrayfake.WithTrace(newTrace("t3")))
	var appCtx = uitest.NewAppContext()
	var page = NewXRayHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "/carts")

	uitest.PressKey(t, appCtx.App, tcell.KeyDown)
	uitest.PressKey(t, appCtx.App, tcell.KeyDown)

	uitest.AwaitScreenText(t, appCtx.App, screen, "/refunds")
}

func TestTracesTable__ErrorIsShown(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithErr(errors.New("throttled")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var table = NewTracesTable(serviceCtx)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshTraces(true)

	if got := uitest.AwaitError(t, received); got != "throttled" {
		t.Errorf("error = %q, want throttled", got)
	}
}

func newTracesTest(t *testing.T, api *xrayfake.Api) *TracesPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var view = NewTracesPageView(NewTracesTable(serviceCtx), NewTraceDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestTracesPage__SelectingATraceShowsItThenItsSegments(t *testing.T) {
	var api = xrayfake.New(
		xrayfake.WithTracePages([]types.TraceSummary{newSummary("t1", "/orders", 30)}),
		xrayfake.WithTrace(newTrace("t1")),
	)
	var view = newTracesTest(t, api)
	var app = view.TracesTable.serviceCtx.App

	view.TracesTable.RefreshTraces(true)

	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Trace ID", "t1"},
		{"Start Time", "2026-11-14 09:30:00"},
		{"Duration", "1.50s"},
		{"Response Time", "125ms"},
		{"Entry Point", "orders-api (AWS::ApiGateway::Stage)"},
		{"Method", "POST"},
		{"URL", "/orders"},
		{"Status", "502"},
		{"Client IP", "203.0.113.7"},
		{"Fault", "true"},
		{"Error", "false"},
		{"Throttle", "false"},
		{"Partial", "false"},
		{"Users", "alice"},
		{"Root Causes"},
		{"orders", "Fault: ProvisionedThroughputExceededException: Rate exceeded"},
		{"Services"},
		{"orders-api", "AWS::ApiGateway::Stage"},
		{"orders", "AWS::Lambda::Function"},
		{"Resources"},
		{"ARN", "arn:aws:lambda:us-east-1:123456789012:function:orders"},
		{"Segments"},
		{"s3", ""},
		{"orders-api", "████████████████████ AWS::ApiGateway::Stage, 1.60s, 502"},
		{"orders", "···████████████████· AWS::Lambda::Function, 1.25s, fault"},
	})
}

func TestTracesPage__MovingThroughTracesLoadsOnlyTheTraceItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = xrayfake.New(
			xrayfake.WithTracePages([]types.TraceSummary{
				newSummary("t1", "/a", 40), newSummary("t2", "/b", 30), newSummary("t3", "/c", 20), newSummary("t4", "/d", 10),
			}),
			xrayfake.WithTrace(newTrace("t1")), xrayfake.WithTrace(newTrace("t2")),
			xrayfake.WithTrace(newTrace("t3")), xrayfake.WithTrace(newTrace("t4")),
		)
		var view = newTracesTest(t, api)
		var app = view.TracesTable.serviceCtx.App
		view.TracesTable.RefreshTraces(true)
		uitest.Eventually(t, app, func() bool { return view.TracesTable.GetTable().GetRowCount() == 5 })

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, app, uitest.MoveThrough[string]{
			Table: view.TracesTable, Passed: "t3", StopsOn: "t4", Loaded: api.TracesAskedFor,
			// The summary is shown straight away, before the segments
			StraightAway: func() {
				if got := view.DetailsTable.GetCellText(0, 1); got != "t4" {
					t.Errorf("trace shown = %q, want t4", got)
				}
			},
		})
	})
}

func TestTraceDetailsTable__ErrorLoadingTheSegmentsIsShown(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithTracePages([]types.TraceSummary{newSummary("t1", "/orders", 30)}))
	var view = newTracesTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.DetailsTable.ErrorMessageCallback = callback

	view.TracesTable.RefreshTraces(true)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "t1") {
		t.Errorf("error = %q, want it to name the trace", got)
	}
}

func ruleArnOf(name string) string {
	return "arn:aws:xray:us-east-1:123456789012:sampling-rule/" + name
}

func newRule(name string, priority int32) types.SamplingRuleRecord {
	var created = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.SamplingRuleRecord{
		CreatedAt: &created,
		SamplingRule: &types.SamplingRule{
			RuleName:      aws.String(name),
			RuleARN:       aws.String(ruleArnOf(name)),
			Priority:      aws.Int32(priority),
			ReservoirSize: 1,
			FixedRate:     0.05,
			ServiceName:   aws.String("*"),
			ServiceType:   aws.String("*"),
			Host:          aws.String("*"),
			HTTPMethod:    aws.String("GET"),
			URLPath:       aws.String("/" + name),
			ResourceARN:   aws.String("*"),
			Version:       aws.Int32(1),
			Attributes:    map[string]string{"tier": "gold", "env": "prod"},
		},
	}
}

func newRulesTest(t *testing.T, api *xrayfake.Api) *SamplingRulesPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var view = NewSamplingRulesPageView(
		NewSamplingRulesTable(serviceCtx), NewSamplingRuleDetailsTable(serviceCtx.AppContext), serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestSamplingRulesTable__ResetLoadsTheRules(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithSamplingRules([]types.SamplingRuleRecord{
		newRule("orders", 10), newRule("health", 20),
	}))
	var view = newRulesTest(t, api)
	var app = view.RulesTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.RulesTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.RulesTable, []core.TableRow{
		{"Name", "Priority", "Reservoir", "Rate", "Service Name", "URL Path"},
		{"orders", "10", "1", "5%", "*", "/orders"},
		{"health", "20", "1", "5%", "*", "/health"},
	})
}

func TestSamplingRulesPage__SelectingARuleShowsIt(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithSamplingRules([]types.SamplingRuleRecord{newRule("orders", 10)}))
	var view = newRulesTest(t, api)

	view.RulesTable.RefreshRules(true)

	uitest.AwaitDetailsRows(t, view.RulesTable.serviceCtx.App, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Priority", "10"},
		{"Reservoir", "1"},
		{"Rate", "5%"},
		{"Service Name", "*"},
		{"Service Type", "*"},
		{"Host", "*"},
		{"HTTP Method", "GET"},
		{"URL Path", "/orders"},
		{"Resource ARN", "*"},
		{"Created", "2026-01-02 03:04:05"},
		{"Modified", ""},
		{"ARN", ruleArnOf("orders")},
		{"Attributes"},
		{"env", "prod"},
		{"tier", "gold"},
	})
}

func TestSamplingRulesTable__OpenLoadsTheRulesToSelectOneNotLoaded(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithSamplingRules([]types.SamplingRuleRecord{
		newRule("orders", 10), newRule("health", 20),
	}))
	var view = newRulesTest(t, api)
	var app = view.RulesTable.serviceCtx.App

	uitest.OnUI(t, app, func() { view.RulesTable.Open(ruleArnOf("health")) })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(samplingRule(view.RulesTable.GetSelectedRule()).RuleName) == "health"
	}, "health to be selected")
}

func TestSamplingRulesTable__OpenOfAMissingRuleIsAnError(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithSamplingRules([]types.SamplingRuleRecord{newRule("orders", 10)}))
	var view = newRulesTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.RulesTable.ErrorMessageCallback = callback

	uitest.OnUI(t, view.RulesTable.serviceCtx.App, func() { view.RulesTable.Open(ruleArnOf("gone")) })

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "sampling-rule/gone") {
		t.Errorf("error = %q, want it to name the rule", got)
	}
}

func TestIsOpenableSamplingRule(t *testing.T) {
	for _, test := range []struct {
		resource string
		want     bool
	}{
		{"sampling-rule/orders", true},
		{"sampling-rule/", false},
		{"group/orders/ABC", false},
	} {
		var resource = arn.ARN{Service: "xray", Resource: test.resource}
		if got := isOpenableSamplingRule(resource); got != test.want {
			t.Errorf("isOpenableSamplingRule(%q) = %t, want %t", test.resource, got, test.want)
		}
	}
}

func TestHomeView__ResetLoadsTheTraces(t *testing.T) {
	var api = xrayfake.New(
		xrayfake.WithTracePages([]types.TraceSummary{newSummary("t1", "/orders", 30)}),
		xrayfake.WithTrace(newTrace("t1")),
	)
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewXRayHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "/orders")
}

func TestTracesTable__TimeRangeKeyLoadsTheTracesOfTheNextRange(t *testing.T) {
	var api = xrayfake.New(
		xrayfake.WithTracePagesSince(time.Hour, []types.TraceSummary{newSummary("t1", "/hour", 30)}),
		xrayfake.WithTracePagesSince(3*time.Hour, []types.TraceSummary{newSummary("t2", "/three-hours", 20)}),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var table = NewTracesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() { uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Reset) })
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(1, 5) == "/hour" }, "/hour to be shown")

	uitest.OnUI(t, serviceCtx.App, func() { uitest.PressTableRune(table, core.APP_KEY_BINDINGS.TimeRange) })

	// The rows of the last hour are as many, so the new ones are waited for
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return table.GetCellText(1, 5) == "/three-hours"
	}, "/three-hours to be shown")
	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		traceHeadings,
		{"2026-11-14 09:20:00", "orders-api", "POST", "502", "125ms", "/three-hours"},
	})
	uitest.OnUI(t, serviceCtx.App, func() {
		if title := table.GetTitle(); !strings.Contains(title, "Last 3 hours") {
			t.Errorf("title = %q, want the range", title)
		}
	})
	if got := api.SpansAskedFor(); !slices.Equal(got, []time.Duration{time.Hour, 3 * time.Hour}) {
		t.Errorf("spans = %v, want [1h 3h]", got)
	}
}

func TestTracesTable__TimeRangeGoesBackToTheShortestAfterTheLongest(t *testing.T) {
	var api = xrayfake.New()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var table = NewTracesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() {
		for range customTraceRangeIdx - defaultTraceRangeIdx {
			uitest.PressTableRune(table, core.APP_KEY_BINDINGS.TimeRange)
		}
	})

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return len(api.SpansAskedFor()) == customTraceRangeIdx-defaultTraceRangeIdx
	}, "every range to be loaded")
	// The loads run at once, so they can come in any order
	var spans = api.SpansAskedFor()
	slices.Sort(spans)
	if want := []time.Duration{5 * time.Minute, 3 * time.Hour, 6 * time.Hour}; !slices.Equal(spans, want) {
		t.Errorf("spans = %v, want %v", spans, want)
	}
}

func TestTracesTable__TracesOfARangeThatCameAfterTheNextAreNotShown(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = xrayfake.New(
			xrayfake.WithTracePagesSince(3*time.Hour, []types.TraceSummary{newSummary("t1", "/three-hours", 30)}),
			xrayfake.WithTracePagesSince(6*time.Hour, []types.TraceSummary{newSummary("t2", "/six-hours", 20)}),
		)
		var release = api.HoldSpan(3 * time.Hour)
		t.Cleanup(release)
		var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
		var table = NewTracesTable(serviceCtx)
		uitest.RunApp(t, serviceCtx.App)

		uitest.OnUI(t, serviceCtx.App, func() {
			uitest.PressTableRune(table, core.APP_KEY_BINDINGS.TimeRange)
			uitest.PressTableRune(table, core.APP_KEY_BINDINGS.TimeRange)
		})
		uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(1, 5) == "/six-hours" }, "/six-hours to be shown")
		release()

		uitest.Consistently(t, serviceCtx.App, 2*core.SelectionLoadDelay, func() bool {
			return table.GetCellText(1, 5) == "/six-hours"
		}, "the traces of the last range to stay")
	})
}

// fixedNow is when the ranges ending now end in the tests of the range view
var fixedNow = time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)

func newRangeTest(t *testing.T, api *xrayfake.Api) (*TracesPageView, *tview.Application) {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var table = NewTracesTable(serviceCtx)
	table.now = func() time.Time { return fixedNow }
	var view = NewTracesPageView(table, NewTraceDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view, serviceCtx.App
}

// pressEnter presses Enter on the input, as it is when it has the focus
func pressEnter(input tview.Primitive) {
	input.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
}

func currentRange(view *TraceRangeView) string {
	var _, text = view.rangeDropDown.GetCurrentOption()
	return text
}

func TestTraceRangeView__EnteredTimesLoadTheTracesBetweenThemOnEnter(t *testing.T) {
	var start, end = fixedNow.Add(-48 * time.Hour), fixedNow.Add(-46 * time.Hour)
	var api = xrayfake.New(xrayfake.WithTracePagesSince(2*time.Hour, []types.TraceSummary{newSummary("t1", "/custom", 30)}))
	var view, app = newRangeTest(t, api)
	var rangeView = view.RangeView

	uitest.OnUI(t, app, func() {
		rangeView.startInput.SetTextTime(start)
		rangeView.endInput.SetTextTime(end)
		if got := currentRange(rangeView); got != "Custom" {
			t.Errorf("range after entering times = %q, want Custom", got)
		}
		if len(api.WindowsAskedFor()) != 0 {
			t.Errorf("traces loaded before Enter")
		}
		pressEnter(rangeView.endInput)
	})

	uitest.Eventually(t, app, func() bool { return view.TracesTable.GetCellText(1, 5) == "/custom" }, "/custom to be shown")
	if got := api.WindowsAskedFor(); len(got) != 1 || !got[0].Start.Equal(start) || !got[0].End.Equal(end) {
		t.Errorf("windows = %v, want the times entered", got)
	}
	uitest.OnUI(t, app, func() {
		if title := view.TracesTable.GetTitle(); !strings.Contains(title, "Custom") {
			t.Errorf("title = %q, want Custom", title)
		}
	})
}

func TestTraceRangeView__TimesThatCantBeUsedAreErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		start, end string
		want       string
	}{
		{"start not a time", "2026-11-14", "2026-11-14 10:00:00", "The start time is not a time like"},
		{"end not a time", "2026-11-14 09:00:00", "10:00", "The end time is not a time like"},
		{"end before start", "2026-11-14 10:00:00", "2026-11-14 09:00:00", "The end time is not after the start time"},
		{"end at start", "2026-11-14 10:00:00", "2026-11-14 10:00:00", "The end time is not after the start time"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var api = xrayfake.New()
			var view, app = newRangeTest(t, api)
			var callback, received = uitest.CaptureErrors()
			view.RangeView.ErrorMessageCallback = callback

			uitest.OnUI(t, app, func() {
				view.RangeView.startInput.SetText(test.start)
				view.RangeView.endInput.SetText(test.end)
				pressEnter(view.RangeView.startInput)
			})

			if got := uitest.AwaitError(t, received); !strings.Contains(got, test.want) {
				t.Errorf("error = %q, want %q", got, test.want)
			}
			if got := api.WindowsAskedFor(); len(got) != 0 {
				t.Errorf("windows = %v, want nothing loaded", got)
			}
		})
	}
}

func TestTraceRangeView__PickingARangeLoadsItAndShowsItsTimes(t *testing.T) {
	var api = xrayfake.New()
	var view, app = newRangeTest(t, api)

	uitest.OnUI(t, app, func() { view.RangeView.rangeDropDown.SetCurrentOption(3) })

	uitest.Eventually(t, app, func() bool { return len(api.WindowsAskedFor()) == 1 }, "the range to be loaded")
	var want = xrayfake.Window{Start: fixedNow.Add(-3 * time.Hour), End: fixedNow}
	if got := api.WindowsAskedFor()[0]; !got.Start.Equal(want.Start) || !got.End.Equal(want.End) {
		t.Errorf("window = %v, want %v", got, want)
	}
	uitest.OnUI(t, app, func() {
		var start, startErr = view.RangeView.startInput.ValidateInput()
		var end, endErr = view.RangeView.endInput.ValidateInput()
		if startErr != nil || endErr != nil || !start.Equal(want.Start) || !end.Equal(want.End) {
			t.Errorf("times shown = %v - %v, want %v - %v", start, end, want.Start, want.End)
		}
	})
}

func TestTraceRangeView__ShowsTheRangeTheKeyMovesTo(t *testing.T) {
	var api = xrayfake.New()
	var view, app = newRangeTest(t, api)

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.TimeRange) })

	uitest.OnUI(t, app, func() {
		if got := currentRange(view.RangeView); got != "Last 3 hours" {
			t.Errorf("range = %q, want Last 3 hours", got)
		}
		if got := view.RangeView.startInput.GetText(); got != fixedNow.Add(-3*time.Hour).Local().Format(timeInputLayout) {
			t.Errorf("start = %q, want three hours before now", got)
		}
	})
	uitest.Eventually(t, app, func() bool { return len(api.WindowsAskedFor()) == 1 }, "the range to be loaded")
}

func TestTracesTable__TimeRangeKeyGoesFromCustomToTheShortest(t *testing.T) {
	var api = xrayfake.New()
	var view, app = newRangeTest(t, api)

	uitest.OnUI(t, app, func() {
		if err := view.TracesTable.SetCustomRange(fixedNow.Add(-2*time.Hour), fixedNow); err != nil {
			t.Error(err)
		}
		uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.TimeRange)
		if got := currentRange(view.RangeView); got != "Last 5 minutes" {
			t.Errorf("range = %q, want Last 5 minutes", got)
		}
	})
}

func TestTracesTable__ResetOfACustomRangeKeepsItsTimes(t *testing.T) {
	var start, end = fixedNow.Add(-30 * time.Hour), fixedNow.Add(-29 * time.Hour)
	var api = xrayfake.New()
	var view, app = newRangeTest(t, api)

	uitest.OnUI(t, app, func() {
		if err := view.TracesTable.SetCustomRange(start, end); err != nil {
			t.Error(err)
		}
		uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.Reset)
	})

	uitest.Eventually(t, app, func() bool { return len(api.WindowsAskedFor()) == 2 }, "both loads")
	for _, got := range api.WindowsAskedFor() {
		if !got.Start.Equal(start) || !got.End.Equal(end) {
			t.Errorf("window = %v, want the custom times", got)
		}
	}
}

// withOutcome is the summary with only the given of fault, error and throttle
func withOutcome(summary types.TraceSummary, fault, err, throttle bool) types.TraceSummary {
	summary.HasFault, summary.HasError, summary.HasThrottle = aws.Bool(fault), aws.Bool(err), aws.Bool(throttle)
	return summary
}

func TestTraceColour(t *testing.T) {
	var summary = newSummary("t1", "/orders", 30)
	for _, test := range []struct {
		name         string
		summary      types.TraceSummary
		want         tcell.Color
		wantColoured bool
	}{
		{"fault", withOutcome(summary, true, false, false), traceFaultColour, true},
		{"fault outranks error and throttle", withOutcome(summary, true, true, true), traceFaultColour, true},
		{"throttle", withOutcome(summary, false, false, true), traceThrottleColour, true},
		{"throttle outranks the error it comes with", withOutcome(summary, false, true, true), traceThrottleColour, true},
		{"error", withOutcome(summary, false, true, false), traceErrorColour, true},
		{"ok", withOutcome(summary, false, false, false), 0, false},
		{"flags not set", types.TraceSummary{}, 0, false},
	} {
		var got, coloured = traceColour(test.summary)
		if got != test.want || coloured != test.wantColoured {
			t.Errorf("%s: traceColour = (%v, %t), want (%v, %t)", test.name, got, coloured, test.want, test.wantColoured)
		}
	}
}

// drawnForeground draws the table and returns the colour the text is drawn in,
// which is what the user sees whichever field of the cell holds the colour.
// The selected row is drawn in the selection colours, so the text must be on
// another row.
func drawnForeground(t *testing.T, table tview.Primitive, text string) tcell.Color {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(120, 10)
	table.SetRect(0, 0, 120, 10)
	table.Draw(screen)
	screen.Show()

	var cells, width, height = screen.GetContents()
	for row := 0; row < height; row++ {
		var line strings.Builder
		for col := 0; col < width; col++ {
			line.WriteString(string(cells[row*width+col].Runes))
		}
		if idx := strings.Index(line.String(), text); idx >= 0 {
			var fg, _, _ = cells[row*width+idx].Style.Decompose()
			return fg
		}
	}
	t.Fatalf("%q is not on the screen", text)
	return 0
}

func TestTracesTable__ColoursTracesByFaultThrottleAndError(t *testing.T) {
	var table = NewTracesTable(uitest.NewServiceContext[awsapi.XRayApi]())
	table.populateTracesTable([]types.TraceSummary{
		withOutcome(newSummary("t1", "/selected-row", 50), false, false, false),
		withOutcome(newSummary("t2", "/faulted", 40), true, true, false),
		withOutcome(newSummary("t3", "/throttled", 30), false, true, true),
		withOutcome(newSummary("t4", "/errored", 20), false, true, false),
		withOutcome(newSummary("t5", "/succeeded", 10), false, false, false),
	}, true)

	for _, test := range []struct {
		text string
		want tcell.Color
	}{
		{"/faulted", traceFaultColour},
		{"/throttled", traceThrottleColour},
		{"/errored", traceErrorColour},
	} {
		if got := drawnForeground(t, table, test.text); got != test.want {
			t.Errorf("%q is drawn in %v, want %v", test.text, got, test.want)
		}
	}
	var plain = []tcell.Color{traceFaultColour, traceThrottleColour, traceErrorColour}
	if got := drawnForeground(t, table, "/succeeded"); slices.Contains(plain, got) {
		t.Errorf("/succeeded is drawn in %v, but had no fault, throttle or error", got)
	}
}

const faultsFilter = `service("orders") { fault }`

func TestTraceFilterView__EnteredFilterLoadsTheTracesThatMatchItOnEnter(t *testing.T) {
	var api = xrayfake.New(
		xrayfake.WithTracePages([]types.TraceSummary{newSummary("t1", "/every-trace", 30)}),
		xrayfake.WithTracePagesFiltered(faultsFilter, []types.TraceSummary{newSummary("t2", "/faulted-trace", 20)}),
	)
	var view, app = newRangeTest(t, api)

	uitest.OnUI(t, app, func() {
		view.FilterView.SetText("  " + faultsFilter + " ")
		if len(api.WindowsAskedFor()) != 0 {
			t.Errorf("traces loaded before Enter")
		}
		pressEnter(view.FilterView)
	})

	uitest.Eventually(t, app, func() bool {
		return view.TracesTable.GetCellText(1, 5) == "/faulted-trace"
	}, "/faulted-trace to be shown")
	if got := api.FiltersAskedFor(); !slices.Equal(got, []string{faultsFilter}) {
		t.Errorf("filters = %q, want the filter entered without its spaces", got)
	}
}

func TestTracesTable__FilterIsKeptForResetAndTheNextRange(t *testing.T) {
	var api = xrayfake.New()
	var view, app = newRangeTest(t, api)

	uitest.OnUI(t, app, func() {
		view.TracesTable.SetFilterExpression(faultsFilter)
		uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.Reset)
		uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.TimeRange)
	})

	uitest.Eventually(t, app, func() bool { return len(api.WindowsAskedFor()) == 3 }, "the three loads")
	if got := api.FiltersAskedFor(); !slices.Equal(got, []string{faultsFilter, faultsFilter, faultsFilter}) {
		t.Errorf("filters = %q, want the filter for every load", got)
	}
}

func TestTraceFilterView__ClearedFilterLoadsEveryTrace(t *testing.T) {
	var api = xrayfake.New(
		xrayfake.WithTracePages([]types.TraceSummary{newSummary("t1", "/every-trace", 30)}),
		xrayfake.WithTracePagesFiltered(faultsFilter, []types.TraceSummary{newSummary("t2", "/faulted-trace", 20)}),
	)
	var view, app = newRangeTest(t, api)
	uitest.OnUI(t, app, func() {
		view.FilterView.SetText(faultsFilter)
		pressEnter(view.FilterView)
	})
	uitest.Eventually(t, app, func() bool {
		return view.TracesTable.GetCellText(1, 5) == "/faulted-trace"
	}, "/faulted-trace to be shown")

	uitest.OnUI(t, app, func() {
		view.FilterView.SetText("")
		pressEnter(view.FilterView)
	})

	uitest.Eventually(t, app, func() bool {
		return view.TracesTable.GetCellText(1, 5) == "/every-trace"
	}, "/every-trace to be shown")
	if got := api.FiltersAskedFor(); !slices.Equal(got, []string{faultsFilter, ""}) {
		t.Errorf("filters = %q, want the filter then none", got)
	}
}

func TestTraceFilterView__FilterThatCantBeUsedIsAnError(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithErr(errors.New("InvalidRequestException: invalid filter expression")))
	var view, app = newRangeTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.TracesTable.ErrorMessageCallback = callback

	uitest.OnUI(t, app, func() {
		view.FilterView.SetText("service(")
		pressEnter(view.FilterView)
	})

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "invalid filter expression") {
		t.Errorf("error = %q, want the error of the filter", got)
	}
}

func segmentOf(id, document string) types.Segment {
	return types.Segment{Id: aws.String(id), Document: aws.String(document)}
}

func TestSegmentTree__ShowsSubsegmentsAndCalledSegmentsUnderTheirCaller(t *testing.T) {
	var segments = []types.Segment{
		// The segment of DynamoDB names the subsegment that called it
		segmentOf("dynamo", `{"id":"dynamo","parent_id":"sub-put","name":"DynamoDB","origin":"AWS::DynamoDB::Table",`+
			`"inferred":true,"start_time":10.5,"end_time":11.5,"throttle":true,"error":true}`),
		segmentOf("api", `{"id":"api","name":"orders-api","origin":"AWS::ApiGateway::Stage",`+
			`"start_time":10,"end_time":12,"http":{"response":{"status":502}},"subsegments":[`+
			`{"id":"sub-lambda","name":"Lambda","start_time":10.1,"end_time":11.9,"fault":true}]}`),
		segmentOf("fn", `{"id":"fn","parent_id":"sub-lambda","name":"orders","origin":"AWS::Lambda::Function",`+
			`"start_time":10.2,"end_time":11.8,"subsegments":[`+
			`{"id":"sub-put","name":"DynamoDB","namespace":"aws","start_time":10.4,"end_time":11.6,"subsegments":[`+
			`{"id":"sub-marshal","name":"marshal","start_time":10.4,"end_time":10.41}]},`+
			`{"id":"sub-init","name":"Initialization","start_time":10.2,"end_time":10.4}]}`),
		// A segment whose caller is not in the trace is shown at the top
		segmentOf("orphan", `{"id":"orphan","parent_id":"elsewhere","name":"worker","start_time":11,"in_progress":true}`),
		// A segment that names its own subsegment as its parent is shown once
		segmentOf("loop", `{"id":"loop","parent_id":"loop-sub","name":"looped","start_time":12,"end_time":12,`+
			`"subsegments":[{"id":"loop-sub","name":"looped-sub"}]}`),
	}

	var got = newSegmentTree(segments).rows()

	var want = []core.TableRow{
		{"orders-api", "████████████████████ AWS::ApiGateway::Stage, 2.00s, 502"},
		{"  Lambda", "·██████████████████· 1.80s, fault"},
		{"    orders", "··████████████████·· AWS::Lambda::Function, 1.60s"},
		{"      Initialization", "··██················ 200ms"},
		{"      DynamoDB", "····████████████···· 1.20s"},
		{"        marshal", "····█··············· 10ms"},
		{"        DynamoDB", "·····██████████····· AWS::DynamoDB::Table, 1.00s, error, throttle"},
		{"worker", "in progress"},
		{"looped", "···················█ 0ms"},
		{"  looped-sub", ""},
	}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("rows =\n%s\nwant\n%s", formatRows(got), formatRows(want))
	}
}

func TestSegmentTree__SegmentsAtOneInstantFillTheBar(t *testing.T) {
	var got = newSegmentTree([]types.Segment{
		segmentOf("s1", `{"id":"s1","name":"instant","start_time":5,"end_time":5}`),
	}).rows()

	if want := (core.TableRow{"instant", "████████████████████ 0ms"}); len(got) != 1 || !slices.Equal(got[0], want) {
		t.Errorf("rows = %q, want %q", got, want)
	}
}

func formatRows(rows []core.TableRow) string {
	var lines []string
	for _, row := range rows {
		lines = append(lines, strings.Join(row, " | "))
	}
	return strings.Join(lines, "\n")
}

func TestTraceLogGroups(t *testing.T) {
	var summary = types.TraceSummary{
		ResourceARNs: []types.ResourceARNDetail{
			{ARN: aws.String("arn:aws:lambda:us-east-1:123456789012:function:orders:$LATEST")},
			{ARN: aws.String("arn:aws:dynamodb:us-east-1:123456789012:table/orders")},
			{ARN: aws.String("not an arn")},
		},
		ServiceIds: []types.ServiceId{
			{Name: aws.String("orders"), Type: aws.String("AWS::Lambda::Function")},
			{Name: aws.String("payments"), Type: aws.String("AWS::Lambda::Function")},
			{Name: aws.String("orders-api"), Type: aws.String("AWS::ApiGateway::Stage")},
		},
	}
	var trace = types.Trace{Segments: []types.Segment{
		segmentOf("s1", `{"name":"carts","origin":"AWS::ECS::Container","aws":{"cloudwatch_logs":[`+
			`{"log_group":"/ecs/carts","arn":"arn:aws:logs:us-east-1:123456789012:log-group:/ecs/carts"},{"log_group":""}]}}`),
		segmentOf("s2", `{"name":"refunds","origin":"AWS::Lambda::Function"}`),
		segmentOf("s3", `{"name":"refunds","origin":"AWS::Lambda"}`),
		segmentOf("s4", "not json"),
	}}

	if got, want := traceLogGroups(summary, nil), []string{"/aws/lambda/orders", "/aws/lambda/payments"}; !slices.Equal(got, want) {
		t.Errorf("groups before the segments are loaded = %q, want %q", got, want)
	}
	var want = []string{"/aws/lambda/orders", "/aws/lambda/payments", "/aws/lambda/refunds", "/ecs/carts"}
	if got := traceLogGroups(summary, &trace); !slices.Equal(got, want) {
		t.Errorf("groups = %q, want %q", got, want)
	}
}

func TestTraceLogsQuery__FindsTheLinesOfTheTraceAroundItsTime(t *testing.T) {
	var summary = newSummary("1-5f84c7a1-abc", "/orders", 30)

	var query, err = traceLogsQuery(summary, nil)

	if err != nil {
		t.Fatal(err)
	}
	var want = core.LogsQuery{
		Query: "fields @timestamp, @log, @message\n" +
			"| filter @message like \"1-5f84c7a1-abc\"\n" +
			"| sort @timestamp asc\n" +
			"| limit 1000",
		LogGroups: []string{"/aws/lambda/orders"},
		// The trace starts at 09:30 and lasts 1.5s
		Start: time.Date(2026, 11, 14, 9, 25, 0, 0, time.UTC),
		End:   time.Date(2026, 11, 14, 9, 35, 1, 500_000_000, time.UTC),
	}
	if query.Query != want.Query || !slices.Equal(query.LogGroups, want.LogGroups) ||
		!query.Start.Equal(want.Start) || !query.End.Equal(want.End) {
		t.Errorf("query = %+v, want %+v", query, want)
	}
}

func TestTraceLogsQuery__TraceThatCantBeQueriedIsAnError(t *testing.T) {
	var withoutStart = newSummary("t1", "/orders", 30)
	withoutStart.StartTime = nil
	var withoutLogs = newSummary("t2", "/orders", 30)
	withoutLogs.ResourceARNs, withoutLogs.ServiceIds = nil, nil

	for _, test := range []struct {
		name    string
		summary types.TraceSummary
		want    string
	}{
		{"no trace", types.TraceSummary{}, "No trace is selected"},
		{"no start time", withoutStart, "Trace t1 has no start time to find its logs by"},
		{"no log groups", withoutLogs, "No log groups were found for trace t2"},
	} {
		if _, err := traceLogsQuery(test.summary, nil); err == nil || err.Error() != test.want {
			t.Errorf("%s: error = %v, want %q", test.name, err, test.want)
		}
	}
}

func TestTracesPage__QueryKeyOpensTheLogsOfTheTraceWithItsSegments(t *testing.T) {
	var trace = newTrace("t1")
	trace.Segments = append(trace.Segments, segmentOf("s4",
		`{"name":"carts","origin":"AWS::ECS::Container","aws":{"cloudwatch_logs":[{"log_group":"/ecs/carts"}]}}`))
	var api = xrayfake.New(
		xrayfake.WithTracePages([]types.TraceSummary{newSummary("t1", "/orders", 30)}),
		xrayfake.WithTrace(trace),
	)
	var view = newTracesTest(t, api)
	var app = view.TracesTable.serviceCtx.App
	var opened safetest.Recorder[core.LogsQuery]
	view.TracesTable.serviceCtx.AppContext.SetLogsQueryOpener(func(query core.LogsQuery) error {
		opened.Record(query)
		return nil
	})
	view.TracesTable.RefreshTraces(true)
	uitest.Eventually(t, app, func() bool { return view.DetailsTable.LoadedTrace() != nil }, "the segments to be loaded")

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.TableQuery) })

	var calls = opened.Calls()
	if len(calls) != 1 {
		t.Fatalf("queries opened = %d, want 1", len(calls))
	}
	if want := []string{"/aws/lambda/orders", "/ecs/carts"}; !slices.Equal(calls[0].LogGroups, want) {
		t.Errorf("log groups = %q, want %q", calls[0].LogGroups, want)
	}
	if !strings.Contains(calls[0].Query, `"t1"`) {
		t.Errorf("query = %q, want it to find the trace", calls[0].Query)
	}
}

func TestTracesPage__QueryKeyReportsWhyTheLogsCantBeOpened(t *testing.T) {
	var withoutLogs = newSummary("t1", "/orders", 30)
	withoutLogs.ResourceARNs, withoutLogs.ServiceIds = nil, nil
	for _, test := range []struct {
		name    string
		summary types.TraceSummary
		openErr error
		want    string
	}{
		{"no log groups", withoutLogs, nil, "No log groups were found for trace t1"},
		{"no logs page", newSummary("t1", "/orders", 30), errors.New("can't open Logs Insights queries"),
			"can't open Logs Insights queries"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var api = xrayfake.New(xrayfake.WithTracePages([]types.TraceSummary{test.summary}))
			var view = newTracesTest(t, api)
			var app = view.TracesTable.serviceCtx.App
			var callback, received = uitest.CaptureErrors()
			var opened safetest.Recorder[core.LogsQuery]
			view.TracesTable.serviceCtx.AppContext.SetLogsQueryOpener(func(query core.LogsQuery) error {
				opened.Record(query)
				return test.openErr
			})
			view.TracesTable.RefreshTraces(true)
			uitest.Eventually(t, app, func() bool { return view.TracesTable.GetCellText(1, 5) == "/orders" }, "/orders to be shown")
			uitest.OnUI(t, app, func() { view.TracesTable.ErrorMessageCallback = callback })

			uitest.OnUI(t, app, func() { uitest.PressTableRune(view.TracesTable, core.APP_KEY_BINDINGS.TableQuery) })

			if got := uitest.AwaitError(t, received); got != test.want {
				t.Errorf("error = %q, want %q", got, test.want)
			}
		})
	}
}

func edgeTo(referenceId int32, stats types.EdgeStatistics, histogram ...types.HistogramEntry) types.Edge {
	return types.Edge{ReferenceId: aws.Int32(referenceId), SummaryStatistics: &stats, ResponseTimeHistogram: histogram}
}

// newServiceGraph is clients calling an API, which calls a function, which
// uses a table, in no order
func newServiceGraph() []types.Service {
	return []types.Service{
		{
			ReferenceId: aws.Int32(1), Name: aws.String("orders-api"), Type: aws.String("AWS::ApiGateway::Stage"),
			Names: []string{"orders-api", "orders-api/prod"}, Root: aws.Bool(true),
			SummaryStatistics: &types.ServiceStatistics{
				TotalCount: aws.Int64(200), OkCount: aws.Int64(190), TotalResponseTime: aws.Float64(40),
				ErrorStatistics: &types.ErrorStatistics{TotalCount: aws.Int64(6), ThrottleCount: aws.Int64(2)},
				FaultStatistics: &types.FaultStatistics{TotalCount: aws.Int64(4)},
			},
			ResponseTimeHistogram: []types.HistogramEntry{{Value: 1.5, Count: 20}, {Value: 0.1, Count: 100}, {Value: 0.2, Count: 80}},
			Edges: []types.Edge{edgeTo(2, types.EdgeStatistics{
				TotalCount: aws.Int64(200), OkCount: aws.Int64(196), TotalResponseTime: aws.Float64(20),
				FaultStatistics: &types.FaultStatistics{TotalCount: aws.Int64(4)},
			}, types.HistogramEntry{Value: 0.05, Count: 150}, types.HistogramEntry{Value: 0.3, Count: 50})},
		},
		{ReferenceId: aws.Int32(3), Name: aws.String("orders-table"), Type: aws.String("AWS::DynamoDB::Table")},
		{
			ReferenceId: aws.Int32(2), Name: aws.String("orders"), Type: aws.String("AWS::Lambda::Function"),
			SummaryStatistics: &types.ServiceStatistics{
				TotalCount: aws.Int64(200), OkCount: aws.Int64(200), TotalResponseTime: aws.Float64(10),
			},
			ResponseTimeHistogram: []types.HistogramEntry{{Value: 0.05, Count: 200}},
			Edges:                 []types.Edge{edgeTo(3, types.EdgeStatistics{TotalCount: aws.Int64(0)})},
		},
		{
			ReferenceId: aws.Int32(0), Type: aws.String("client"),
			Edges: []types.Edge{edgeTo(1, types.EdgeStatistics{
				TotalCount: aws.Int64(10), OkCount: aws.Int64(10), TotalResponseTime: aws.Float64(1),
			})},
		},
	}
}

var serviceMapHeadings = core.TableRow{"Name", "Type", "Requests", "Errors", "Faults", "Average", "p99"}

func newServiceMapTest(t *testing.T, api *xrayfake.Api) *ServiceMapPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.XRayApi](api)
	var table = NewServiceMapTable(serviceCtx)
	table.now = func() time.Time { return fixedNow }
	var view = NewServiceMapPageView(table, NewServiceDetailsTable(serviceCtx.AppContext), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestServiceMapTable__ResetLoadsTheServicesOfTheLastHourWithTheClientsFirst(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithServiceGraph(newServiceGraph()))
	var view = newServiceMapTest(t, api)
	var app = view.ServicesTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.ServicesTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.ServicesTable, []core.TableRow{
		serviceMapHeadings,
		{"Clients", "client", "", "", "", "", ""},
		{"orders", "AWS::Lambda::Function", "200", "0.0%", "0.0%", "50ms", "50ms"},
		{"orders-api", "AWS::ApiGateway::Stage", "200", "3.0%", "2.0%", "200ms", "1.50s"},
		{"orders-table", "AWS::DynamoDB::Table", "", "", "", "", ""},
	})
	var want = xrayfake.Window{Start: fixedNow.Add(-time.Hour), End: fixedNow}
	if got := api.GraphWindowsAskedFor(); len(got) != 1 || !got[0].Start.Equal(want.Start) || !got[0].End.Equal(want.End) {
		t.Errorf("windows = %v, want the last hour", got)
	}
}

func TestServiceMapTable__TimeRangeKeyLoadsTheMapOfTheNextRange(t *testing.T) {
	var api = xrayfake.New()
	var view = newServiceMapTest(t, api)
	var app = view.ServicesTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.ServicesTable, core.APP_KEY_BINDINGS.TimeRange) })

	uitest.Eventually(t, app, func() bool { return len(api.GraphWindowsAskedFor()) == 1 }, "the map to be loaded")
	if got := api.GraphWindowsAskedFor()[0]; !got.Start.Equal(fixedNow.Add(-3 * time.Hour)) {
		t.Errorf("window = %v, want the last 3 hours", got)
	}
	uitest.OnUI(t, app, func() {
		if title := view.ServicesTable.GetTitle(); !strings.Contains(title, "Last 3 hours") {
			t.Errorf("title = %q, want the range", title)
		}
	})
}

func TestServiceMapTable__TimeRangeGoesBackToTheShortestAfterTheLongest(t *testing.T) {
	var api = xrayfake.New()
	var view = newServiceMapTest(t, api)
	var app = view.ServicesTable.serviceCtx.App

	uitest.OnUI(t, app, func() {
		for range customTraceRangeIdx - defaultTraceRangeIdx {
			uitest.PressTableRune(view.ServicesTable, core.APP_KEY_BINDINGS.TimeRange)
		}
	})

	uitest.Eventually(t, app, func() bool {
		return len(api.GraphWindowsAskedFor()) == customTraceRangeIdx-defaultTraceRangeIdx
	}, "every range to be loaded")
	// The loads run at once, so they can come in any order
	var spans []time.Duration
	for _, window := range api.GraphWindowsAskedFor() {
		spans = append(spans, window.End.Sub(window.Start))
	}
	slices.Sort(spans)
	if want := []time.Duration{5 * time.Minute, 3 * time.Hour, 6 * time.Hour}; !slices.Equal(spans, want) {
		t.Errorf("spans = %v, want %v", spans, want)
	}
}

func TestServiceMapPage__SelectingAServiceShowsItsRequestsAndCalls(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithServiceGraph(newServiceGraph()))
	var view = newServiceMapTest(t, api)
	var app = view.ServicesTable.serviceCtx.App
	uitest.OnUI(t, app, func() { view.ServicesTable.RefreshServices() })
	uitest.Eventually(t, app, func() bool { return view.ServicesTable.GetCellText(3, 0) == "orders-api" }, "orders-api to be listed")

	uitest.OnUI(t, app, func() { view.ServicesTable.Select(3, 0) })

	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "orders-api"},
		{"Other Names", "orders-api/prod"},
		{"Type", "AWS::ApiGateway::Stage"},
		{"Account", ""},
		{"State", ""},
		{"Root", "true"},
		{"First Segment", ""},
		{"Last Segment", ""},
		{"Requests"},
		{"Total", "200"},
		{"OK", "190"},
		{"Errors", "6 (3.0%)"},
		{"Throttles", "2 (1.0%)"},
		{"Faults", "4 (2.0%)"},
		{"Average", "200ms"},
		{"p50", "100ms"},
		{"p90", "200ms"},
		{"p99", "1.50s"},
		{"Calls"},
		{"orders", "200 requests, 0.0% errors, 2.0% faults, avg 100ms, p99 300ms"},
		{"Called By"},
		{"Clients", "10 requests, 0.0% errors, 0.0% faults, avg 100ms"},
	})
}

func TestServiceMapPage__ServiceWithoutStatisticsShowsOnlyWhoCallsIt(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithServiceGraph(newServiceGraph()))
	var view = newServiceMapTest(t, api)
	var app = view.ServicesTable.serviceCtx.App
	uitest.OnUI(t, app, func() { view.ServicesTable.RefreshServices() })
	uitest.Eventually(t, app, func() bool { return view.ServicesTable.GetCellText(4, 0) == "orders-table" }, "orders-table to be listed")

	uitest.OnUI(t, app, func() { view.ServicesTable.Select(4, 0) })

	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "orders-table"},
		{"Other Names", ""},
		{"Type", "AWS::DynamoDB::Table"},
		{"Account", ""},
		{"State", ""},
		{"Root", ""},
		{"First Segment", ""},
		{"Last Segment", ""},
		{"Called By"},
		{"orders", "0 requests"},
	})
}

func TestServiceMapTable__ColoursServicesByFaultThrottleAndError(t *testing.T) {
	var throttled = types.Service{ReferenceId: aws.Int32(5), Name: aws.String("throttled-service"),
		SummaryStatistics: &types.ServiceStatistics{TotalCount: aws.Int64(10),
			ErrorStatistics: &types.ErrorStatistics{TotalCount: aws.Int64(1), ThrottleCount: aws.Int64(1)}}}
	var errored = types.Service{ReferenceId: aws.Int32(6), Name: aws.String("errored-service"),
		SummaryStatistics: &types.ServiceStatistics{TotalCount: aws.Int64(10),
			ErrorStatistics: &types.ErrorStatistics{TotalCount: aws.Int64(1)}}}
	var table = NewServiceMapTable(uitest.NewServiceContext[awsapi.XRayApi]())
	table.services = sortedServices(append(newServiceGraph(), throttled, errored))
	table.populateServiceMapTable()

	for _, test := range []struct {
		text string
		want tcell.Color
	}{
		{"orders-api", traceFaultColour},
		{"throttled-service", traceThrottleColour},
		{"errored-service", traceErrorColour},
	} {
		if got := drawnForeground(t, table, test.text); got != test.want {
			t.Errorf("%q is drawn in %v, want %v", test.text, got, test.want)
		}
	}
	var coloured = []tcell.Color{traceFaultColour, traceThrottleColour, traceErrorColour}
	if got := drawnForeground(t, table, "AWS::Lambda::Function"); slices.Contains(coloured, got) {
		t.Errorf("orders is drawn in %v, but had no fault, throttle or error", got)
	}
}

func TestServiceMapTable__ErrorIsShown(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithErr(errors.New("throttled")))
	var view = newServiceMapTest(t, api)
	var callback, received = uitest.CaptureErrors()
	var app = view.ServicesTable.serviceCtx.App
	uitest.OnUI(t, app, func() { view.ServicesTable.ErrorMessageCallback = callback })

	uitest.OnUI(t, app, func() { view.ServicesTable.RefreshServices() })

	if got := uitest.AwaitError(t, received); got != "throttled" {
		t.Errorf("error = %q, want throttled", got)
	}
}

func TestRequestStatsPercentile(t *testing.T) {
	var stats = requestStats{histogram: []types.HistogramEntry{{Value: 2, Count: 1}, {Value: 0.5, Count: 98}, {Value: 1, Count: 1}}}
	for _, test := range []struct {
		fraction float64
		want     string
	}{
		{0.5, "500ms"},
		{0.98, "500ms"},
		{0.99, "1.00s"},
		{1, "2.00s"},
	} {
		if got := stats.percentile(test.fraction); got != test.want {
			t.Errorf("percentile(%v) = %q, want %q", test.fraction, got, test.want)
		}
	}
	if got := (requestStats{}).percentile(0.99); got != "" {
		t.Errorf("percentile without a histogram = %q, want empty", got)
	}
}

func TestServiceMapTable__MapOfARangeThatCameAfterTheNextIsNotShown(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = xrayfake.New(
			xrayfake.WithServiceGraphSince(3*time.Hour, []types.Service{{ReferenceId: aws.Int32(1), Name: aws.String("three-hours")}}),
			xrayfake.WithServiceGraphSince(6*time.Hour, []types.Service{{ReferenceId: aws.Int32(1), Name: aws.String("six-hours")}}),
		)
		var release = api.HoldSpan(3 * time.Hour)
		t.Cleanup(release)
		var view = newServiceMapTest(t, api)
		var app = view.ServicesTable.serviceCtx.App

		uitest.OnUI(t, app, func() {
			uitest.PressTableRune(view.ServicesTable, core.APP_KEY_BINDINGS.TimeRange)
			uitest.PressTableRune(view.ServicesTable, core.APP_KEY_BINDINGS.TimeRange)
		})
		uitest.Eventually(t, app, func() bool { return view.ServicesTable.GetCellText(1, 0) == "six-hours" }, "six-hours to be shown")
		release()

		uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
			return view.ServicesTable.GetCellText(1, 0) == "six-hours"
		}, "the map of the last range to stay")
	})
}

func TestHomeView__OpeningASamplingRuleShowsItsPage(t *testing.T) {
	var api = xrayfake.New(xrayfake.WithSamplingRules([]types.SamplingRuleRecord{newRule("orders", 10)}))
	var appCtx = uitest.NewAppContext()
	var page = NewXRayHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var resource, _ = arn.Parse(ruleArnOf("orders"))

	uitest.OnUI(t, appCtx.App, func() { page.(core.ResourceOpener).OpenResource(resource) })

	uitest.AwaitScreenText(t, appCtx.App, screen, "Sampling Rule Details")
	uitest.OnUI(t, appCtx.App, func() {
		var root = page.(*core.ServiceRootView)
		if got := root.CurrentServicePage(); got != "Sampling Rules" {
			t.Errorf("page = %q, want Sampling Rules", got)
		}
		if got := root.ServicePages(); !slices.Equal(got, []string{"Traces", "Service Map", "Sampling Rules"}) {
			t.Errorf("pages = %q", got)
		}
	})
}
