package awsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

var (
	hourEnd   = time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	hourStart = hourEnd.Add(-time.Hour)
)

func xRayResponse(body string) *http.Response {
	return response(200, "application/json", body)
}

func TestXRayGetTraceSummariesGivesOnePageAtATime(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch call {
		case 1:
			return xRayResponse(`{"TraceSummaries":[
				{"Id":"t1","StartTime":1700000000},
				{"Id":"t2","StartTime":1700000060}
			],"NextToken":"n1"}`)
		case 2:
			// X-Ray can give an empty page with more after it
			return xRayResponse(`{"TraceSummaries":[],"NextToken":"n2"}`)
		}
		return xRayResponse(`{"TraceSummaries":[{"Id":"t3","StartTime":1699999000}]}`)
	})
	var api = NewXRayApi(fake.clients, testLogger())

	var first, err = api.GetTraceSummaries(context.Background(), hourStart, hourEnd, "", true)
	if err != nil || len(first) != 2 || aws.ToString(first[0].Id) != "t2" {
		t.Fatalf("first page = %v, err %v, want [t2 t1], newest first", first, err)
	}
	var second, secondErr = api.GetTraceSummaries(context.Background(), hourStart, hourEnd, "", false)
	if secondErr != nil || len(second) != 1 || aws.ToString(second[0].Id) != "t3" {
		t.Fatalf("second page = %v, err %v, want [t3] past the empty page", second, secondErr)
	}
	if body := fake.requestBody(2); !strings.Contains(body, `"NextToken":"n2"`) {
		t.Errorf("third request = %s, want the next token", body)
	}

	var past, pastErr = api.GetTraceSummaries(context.Background(), hourStart, hourEnd, "", false)
	if pastErr != nil || len(past) != 0 || fake.calls() != 3 {
		t.Errorf("past the end = %v, err %v, %d calls, want nothing and no request", past, pastErr, fake.calls())
	}
}

func TestXRayGetTraceSummariesAsksForTheTimesOfAReset(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return xRayResponse(`{"TraceSummaries":[{"Id":"t1"}],"NextToken":"n1"}`)
	})
	var api = NewXRayApi(fake.clients, testLogger())
	var otherStart, otherEnd = hourStart.Add(-48 * time.Hour), hourStart.Add(-45 * time.Hour)

	for _, step := range []struct {
		start, end         time.Time
		reset              bool
		wantStart, wantEnd time.Time
	}{
		{hourStart, hourEnd, true, hourStart, hourEnd},
		// The next page is of the same times
		{otherStart, otherEnd, false, hourStart, hourEnd},
		{otherStart, otherEnd, true, otherStart, otherEnd},
	} {
		if _, err := api.GetTraceSummaries(context.Background(), step.start, step.end, "", step.reset); err != nil {
			t.Fatal(err)
		}

		var request struct{ StartTime, EndTime float64 }
		if err := json.Unmarshal([]byte(fake.requestBody(fake.calls()-1)), &request); err != nil {
			t.Fatal(err)
		}
		var gotStart, gotEnd = time.Unix(int64(request.StartTime), 0), time.Unix(int64(request.EndTime), 0)
		if !gotStart.Equal(step.wantStart) || !gotEnd.Equal(step.wantEnd) {
			t.Errorf("reset %t: times = %v - %v, want %v - %v",
				step.reset, gotStart.UTC(), gotEnd.UTC(), step.wantStart, step.wantEnd)
		}
	}
}

func TestXRayGetTraceSummariesResetStartsAgain(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return xRayResponse(`{"TraceSummaries":[{"Id":"t1"}],"NextToken":"n1"}`)
	})
	var api = NewXRayApi(fake.clients, testLogger())

	for range 2 {
		if _, err := api.GetTraceSummaries(context.Background(), hourStart, hourEnd, "", true); err != nil {
			t.Fatal(err)
		}
	}

	if body := fake.requestBody(1); strings.Contains(body, "NextToken") {
		t.Errorf("request after a reset = %s, want no next token", body)
	}
}

func TestXRayGetTraceSummariesAsksForTheFilterOfAReset(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return xRayResponse(`{"TraceSummaries":[{"Id":"t1"}],"NextToken":"n1"}`)
	})
	var api = NewXRayApi(fake.clients, testLogger())
	const faults = `service("orders") { fault }`

	for _, step := range []struct {
		filter string
		reset  bool
		want   *string
	}{
		// No filter is not sent, as an empty one is not allowed
		{"", true, nil},
		{faults, true, aws.String(faults)},
		// The next page is of the same filter
		{"", false, aws.String(faults)},
		{"", true, nil},
	} {
		if _, err := api.GetTraceSummaries(context.Background(), hourStart, hourEnd, step.filter, step.reset); err != nil {
			t.Fatal(err)
		}

		var request struct{ FilterExpression *string }
		if err := json.Unmarshal([]byte(fake.requestBody(fake.calls()-1)), &request); err != nil {
			t.Fatal(err)
		}
		if aws.ToString(request.FilterExpression) != aws.ToString(step.want) ||
			(request.FilterExpression == nil) != (step.want == nil) {
			t.Errorf("filter %q, reset %t: sent %v, want %v",
				step.filter, step.reset, aws.ToString(request.FilterExpression), aws.ToString(step.want))
		}
	}
}

func TestXRayGetTraceGathersTheSegmentsOfEveryPage(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return xRayResponse(`{"Traces":[{"Id":"t1","Duration":0.5,"Segments":[{"Id":"s1"}]}],"NextToken":"n1"}`)
		}
		return xRayResponse(`{"Traces":[{"Id":"t1","Segments":[{"Id":"s2"}]}]}`)
	})

	var trace, err = NewXRayApi(fake.clients, testLogger()).GetTrace(context.Background(), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Segments) != 2 || aws.ToFloat64(trace.Duration) != 0.5 {
		t.Errorf("trace = %+v, want both segments", trace)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"TraceIds":["t1"]`) {
		t.Errorf("request = %s, want the trace ID", body)
	}
}

func TestXRayGetTraceOfATraceNotFoundIsAnError(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return xRayResponse(`{"Traces":[],"UnprocessedTraceIds":["t1"]}`)
	})

	var _, err = NewXRayApi(fake.clients, testLogger()).GetTrace(context.Background(), "t1")
	if err == nil || !strings.Contains(err.Error(), "t1") {
		t.Errorf("err = %v, want one that names the trace", err)
	}
}

func TestXRayGetSamplingRulesInTheOrderTheyAreApplied(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return xRayResponse(`{"SamplingRuleRecords":[
				{"SamplingRule":{"RuleName":"Default","Priority":10000}},
				{"SamplingRule":{"RuleName":"orders","Priority":10}}
			],"NextToken":"n1"}`)
		}
		return xRayResponse(`{"SamplingRuleRecords":[
			{"SamplingRule":{"RuleName":"health","Priority":10}}
		]}`)
	})

	var rules, err = NewXRayApi(fake.clients, testLogger()).GetSamplingRules(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, rule := range rules {
		names = append(names, aws.ToString(rule.SamplingRule.RuleName))
	}
	if strings.Join(names, " ") != "health orders Default" {
		t.Errorf("rules = %v, want [health orders Default]", names)
	}
}

func TestXRayGetServiceGraphGathersEveryPageForTheTimes(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return xRayResponse(`{"Services":[{"ReferenceId":0,"Name":"orders-api"}],"NextToken":"n1"}`)
		}
		return xRayResponse(`{"Services":[{"ReferenceId":1,"Name":"orders"}]}`)
	})

	var services, err = NewXRayApi(fake.clients, testLogger()).GetServiceGraph(context.Background(), hourStart, hourEnd)

	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 || aws.ToString(services[0].Name) != "orders-api" || aws.ToString(services[1].Name) != "orders" {
		t.Errorf("services = %+v, want those of both pages", services)
	}
	var request struct{ StartTime, EndTime float64 }
	if err := json.Unmarshal([]byte(fake.requestBody(0)), &request); err != nil {
		t.Fatal(err)
	}
	if !time.Unix(int64(request.StartTime), 0).Equal(hourStart) || !time.Unix(int64(request.EndTime), 0).Equal(hourEnd) {
		t.Errorf("times = %v - %v, want the hour", request.StartTime, request.EndTime)
	}
	if body := fake.requestBody(1); !strings.Contains(body, `"NextToken":"n1"`) {
		t.Errorf("second request = %s, want the next token", body)
	}
}
