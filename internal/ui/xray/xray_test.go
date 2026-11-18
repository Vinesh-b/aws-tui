package xray

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
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
		{"orders-api", "AWS::ApiGateway::Stage, 1.60s, 502"},
		{"orders", "AWS::Lambda::Function, 1.25s, fault"},
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
func pressEnter(input *core.DateTimeInputField) {
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
func drawnForeground(t *testing.T, table *TracesTable, text string) tcell.Color {
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
