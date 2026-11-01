package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

// schedulerRequest is the method, path and query of a request, which say what
// a Scheduler request does as it is a REST api
func schedulerRequest(req *http.Request) string {
	var text = req.Method + " " + req.URL.Path
	if req.URL.RawQuery != "" {
		text += "?" + req.URL.RawQuery
	}
	return text
}

func TestSchedulerListSchedulesSortedByGroupAndNameAcrossPages(t *testing.T) {
	var requests []string
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		requests = append(requests, schedulerRequest(req))
		if call == 1 {
			return jsonResponse(`{"Schedules":[{"Name":"zeta","GroupName":"default","State":"ENABLED"},` +
				`{"Name":"beta","GroupName":"ops","State":"DISABLED"}],"NextToken":"t"}`)
		}
		return jsonResponse(`{"Schedules":[{"Name":"alpha","GroupName":"default","State":"ENABLED"}]}`)
	})

	var schedules, err = NewSchedulerApi(testLogger()).ListSchedules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var names []string
	for _, s := range schedules {
		names = append(names, aws.ToString(s.GroupName)+"/"+aws.ToString(s.Name))
	}
	if got := strings.Join(names, ","); got != "default/alpha,default/zeta,ops/beta" {
		t.Errorf("schedules = %s", got)
	}
	if len(requests) != 2 || !strings.HasPrefix(requests[0], "GET /schedules") ||
		!strings.Contains(requests[1], "NextToken=t") {
		t.Errorf("requests = %v", requests)
	}
}

func TestSchedulerGetScheduleAsksForTheGroupAndName(t *testing.T) {
	var request string
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		request = schedulerRequest(req)
		return jsonResponse(`{"Name":"nightly","GroupName":"ops","ScheduleExpression":"rate(1 day)",` +
			`"State":"ENABLED","Target":{"Arn":"arn:aws:lambda:us-east-1:1:function:f","RoleArn":"arn:aws:iam::1:role/r"}}`)
	})

	var schedule, err = NewSchedulerApi(testLogger()).GetSchedule(context.Background(), "ops", "nightly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if request != "GET /schedules/nightly?groupName=ops" {
		t.Errorf("request = %q", request)
	}
	if aws.ToString(schedule.ScheduleExpression) != "rate(1 day)" ||
		aws.ToString(schedule.Target.Arn) != "arn:aws:lambda:us-east-1:1:function:f" {
		t.Errorf("schedule = %+v", schedule)
	}
}

func TestSchedulerCreateScheduleSendsTheScheduleAndReturnsItsArn(t *testing.T) {
	var request string
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		request = schedulerRequest(req)
		return jsonResponse(`{"ScheduleArn":"arn:aws:scheduler:us-east-1:1:schedule/ops/nightly"}`)
	})

	var arn, err = NewSchedulerApi(testLogger()).CreateSchedule(context.Background(), scheduler.CreateScheduleInput{
		Name:               aws.String("nightly"),
		GroupName:          aws.String("ops"),
		ScheduleExpression: aws.String("rate(1 day)"),
		FlexibleTimeWindow: &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff},
		State:              types.ScheduleStateDisabled,
		Target: &types.Target{
			Arn:     aws.String("arn:aws:lambda:us-east-1:1:function:f"),
			RoleArn: aws.String("arn:aws:iam::1:role/r"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if arn != "arn:aws:scheduler:us-east-1:1:schedule/ops/nightly" {
		t.Errorf("ARN = %q", arn)
	}
	if request != "POST /schedules/nightly" {
		t.Errorf("request = %q", request)
	}
	var body = fake.requests[0]
	for _, want := range []string{
		`"GroupName":"ops"`, `"ScheduleExpression":"rate(1 day)"`, `"State":"DISABLED"`,
		`"FlexibleTimeWindow":{"Mode":"OFF"}`, `"RoleArn":"arn:aws:iam::1:role/r"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSchedulerUpdateScheduleSendsTheSchedule(t *testing.T) {
	var request string
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		request = schedulerRequest(req)
		return jsonResponse(`{"ScheduleArn":"arn:aws:scheduler:us-east-1:1:schedule/ops/nightly"}`)
	})

	var _, err = NewSchedulerApi(testLogger()).UpdateSchedule(context.Background(), scheduler.UpdateScheduleInput{
		Name:               aws.String("nightly"),
		GroupName:          aws.String("ops"),
		ScheduleExpression: aws.String("rate(2 days)"),
		FlexibleTimeWindow: &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff},
		State:              types.ScheduleStateEnabled,
		Target:             &types.Target{Arn: aws.String("a"), RoleArn: aws.String("r")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if request != "PUT /schedules/nightly" {
		t.Errorf("request = %q", request)
	}
	if body := fake.requests[0]; !strings.Contains(body, `"ScheduleExpression":"rate(2 days)"`) {
		t.Errorf("request %s does not contain the expression", body)
	}
}

func TestSchedulerDeleteScheduleDeletesItFromTheGroup(t *testing.T) {
	var request string
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		request = schedulerRequest(req)
		return jsonResponse(`{}`)
	})

	if err := NewSchedulerApi(testLogger()).DeleteSchedule(context.Background(), "ops", "nightly"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(request, "DELETE /schedules/nightly?") || !strings.Contains(request, "groupName=ops") {
		t.Errorf("request = %q", request)
	}
}

func TestSchedulerReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "application/json", `{"Message":"gone"}`)
	})

	if err := NewSchedulerApi(testLogger()).DeleteSchedule(context.Background(), "ops", "nightly"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSchedulerMissingNamesAreRejected(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse("{}")
	})
	var api = NewSchedulerApi(testLogger())
	var ctx = context.Background()

	var _, getErr = api.GetSchedule(ctx, "ops", "")
	var _, createErr = api.CreateSchedule(ctx, scheduler.CreateScheduleInput{})
	var _, updateErr = api.UpdateSchedule(ctx, scheduler.UpdateScheduleInput{})
	var deleteErr = api.DeleteSchedule(ctx, "ops", "")

	for _, err := range []error{getErr, createErr, updateErr, deleteErr} {
		if err == nil || err.Error() != "schedule name not set" {
			t.Errorf("error = %v, want schedule name not set", err)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("expected no requests, got %d", fake.calls())
	}
}
