package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func cloudTrailResponse(body string) *http.Response {
	return response(200, "application/x-amz-json-1.1", body)
}

func TestCloudTrailLookupEventsGivesOnePageAtATime(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return cloudTrailResponse(`{"Events":[{"EventId":"e1","EventName":"CreateBucket"}],"NextToken":"n1"}`)
		}
		return cloudTrailResponse(`{"Events":[{"EventId":"e2","EventName":"DeleteBucket"}]}`)
	})
	var api = NewCloudTrailApi(fake.clients, testLogger())

	var first, err = api.LookupEvents(context.Background(), true)
	if err != nil || len(first) != 1 || aws.ToString(first[0].EventId) != "e1" {
		t.Fatalf("first page = %v, err %v, want [e1]", first, err)
	}
	var second, secondErr = api.LookupEvents(context.Background(), false)
	if secondErr != nil || len(second) != 1 || aws.ToString(second[0].EventId) != "e2" {
		t.Fatalf("second page = %v, err %v, want [e2]", second, secondErr)
	}
	if body := fake.requestBody(1); !strings.Contains(body, `"NextToken":"n1"`) {
		t.Errorf("second request = %s, want the next token", body)
	}

	var past, pastErr = api.LookupEvents(context.Background(), false)
	if pastErr != nil || len(past) != 0 || fake.calls() != 2 {
		t.Errorf("past the end = %v, err %v, %d calls, want nothing and no request", past, pastErr, fake.calls())
	}
}

func TestCloudTrailLookupEventsResetStartsFromTheNewest(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cloudTrailResponse(`{"Events":[{"EventId":"e1"}],"NextToken":"n1"}`)
	})
	var api = NewCloudTrailApi(fake.clients, testLogger())

	for range 2 {
		if _, err := api.LookupEvents(context.Background(), true); err != nil {
			t.Fatal(err)
		}
	}

	if body := fake.requestBody(1); strings.Contains(body, "NextToken") {
		t.Errorf("request after a reset = %s, want no next token", body)
	}
}

func TestCloudTrailDescribeTrailsSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cloudTrailResponse(`{"trailList":[{"Name":"security"},{"Name":"audit"}]}`)
	})

	var trails, err = NewCloudTrailApi(fake.clients, testLogger()).DescribeTrails(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(trails) != 2 || aws.ToString(trails[0].Name) != "audit" {
		t.Errorf("trails = %v, want [audit security]", trails)
	}
}

func TestCloudTrailGetTrailStatusAsksForTheTrailByArn(t *testing.T) {
	const trailArn = "arn:aws:cloudtrail:us-east-1:123456789012:trail/audit"
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cloudTrailResponse(`{"IsLogging":true}`)
	})

	var status, err = NewCloudTrailApi(fake.clients, testLogger()).GetTrailStatus(context.Background(), trailArn)
	if err != nil {
		t.Fatal(err)
	}
	if !aws.ToBool(status.IsLogging) {
		t.Errorf("is logging = false, want true")
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"Name":"`+trailArn+`"`) {
		t.Errorf("request = %s, want the trail ARN", body)
	}
}
