package awsapi

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// fakeHttp serves canned responses to the SDK clients so the api methods can be
// tested end to end without touching AWS.
type fakeHttp struct {
	mtx      sync.Mutex
	requests []string // request bodies in call order
	handler  func(call int, req *http.Request, body string) *http.Response
}

func (f *fakeHttp) Do(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}

	f.mtx.Lock()
	f.requests = append(f.requests, string(body))
	var call = len(f.requests)
	f.mtx.Unlock()

	var resp = f.handler(call, req, string(body))
	resp.Request = req
	return resp, nil
}

func (f *fakeHttp) calls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return len(f.requests)
}

func response(status int, contentType string, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func jsonResponse(body string) *http.Response {
	return response(200, "application/x-amz-json-1.0", body)
}

// useFakeAws points the package client singleton at a fake HTTP backend.
func useFakeAws(t *testing.T, handler func(call int, req *http.Request, body string) *http.Response) *fakeHttp {
	t.Helper()

	var fake = &fakeHttp{handler: handler}
	ResetAwsApiClients(aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("id", "secret", ""),
		HTTPClient:       fake,
		RetryMaxAttempts: 1,
	}, "test")

	t.Cleanup(func() { ResetAwsApiClients(aws.Config{}, "") })
	return fake
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestS3DownloadFile(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/octet-stream", "file contents")
	})

	var target = filepath.Join(t.TempDir(), "out.txt")
	if err := NewS3BucketsApi(testLogger()).DownloadFile("bucket", "key", target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got, err = os.ReadFile(target)
	if err != nil || string(got) != "file contents" {
		t.Fatalf("unexpected file contents %q, err=%v", got, err)
	}
}

func TestDynamoDBListTagsFollowsNextToken(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return jsonResponse(`{"Tags":[{"Key":"a","Value":"1"}],"NextToken":"n1"}`)
		}
		return jsonResponse(`{"Tags":[{"Key":"b","Value":"2"}]}`)
	})

	var tags, err = NewDynamoDBApi(testLogger()).ListTags(true, "arn:aws:dynamodb:::table/t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.calls() != 2 || !strings.Contains(fake.requests[1], `"NextToken":"n1"`) {
		t.Fatalf("second request did not send the next token: %v", fake.requests)
	}
	if len(tags) != 2 || aws.ToString(tags[0].Key) != "a" || aws.ToString(tags[1].Key) != "b" {
		t.Fatalf("unexpected tags %v", tags)
	}
}

func TestLambdaListLambdasSortedAcrossPages(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return response(200, "application/json",
				`{"Functions":[{"FunctionName":"zeta"}],"NextMarker":"m"}`)
		}
		return response(200, "application/json", `{"Functions":[{"FunctionName":"alpha"}]}`)
	})

	var funcs, err = NewLambdaApi(testLogger()).ListLambdas(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(funcs) != 2 || aws.ToString(funcs[0].FunctionName) != "alpha" {
		t.Fatalf("expected [alpha zeta], got %v", funcs)
	}
}

// Missing arguments must be rejected before any request is made.
func TestMissingArgumentsAreRejected(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse("{}")
	})

	var logger = testLogger()
	var cwl = NewCloudWatchLogsApi(logger)
	var sfnApi = NewStateMachineApi(logger)
	var lambdaApi = NewLambdaApi(logger)
	var s3Api = NewS3BucketsApi(logger)

	var tests = []struct {
		name string
		call func() error
		want string
	}{
		{"cloudformation stack events", func() error {
			var _, err = NewCloudFormationApi(logger).DescribeStackEvents(context.Background(), "", true)
			return err
		}, "stack name not set"},
		{"alarm history", func() error {
			var _, err = NewCloudWatchAlarmsApi(logger).ListAlarmHistory(context.Background(), "", true)
			return err
		}, "alarm name not set"},
		{"log streams", func() error {
			var _, err = cwl.ListLogStreams("", "", true)
			return err
		}, "log group not set"},
		{"log events without stream", func() error {
			var _, err = cwl.ListLogEvents("group", "", true)
			return err
		}, "log stream not set"},
		{"stop insights query", func() error {
			var _, err = cwl.StopInsightsQuery("")
			return err
		}, "query ID not set"},
		{"insights results", func() error {
			var _, _, err = cwl.GetInsightsQueryResults("")
			return err
		}, "query ID not set"},
		{"insights log record", func() error {
			var _, err = cwl.GetInsightsLogRecord("")
			return err
		}, "record pointer not set"},
		{"describe table", func() error {
			var _, err = NewDynamoDBApi(logger).DescribeTable("")
			return err
		}, "table name not set"},
		{"invoke lambda", func() error {
			var _, err = lambdaApi.InvokeLambda(context.Background(), "", nil)
			return err
		}, "lambda name not set"},
		{"lambda policy", func() error {
			var _, err = lambdaApi.GetPolicy(context.Background(), "")
			return err
		}, "lambda ARN not set"},
		{"lambda tags", func() error {
			var _, err = lambdaApi.ListTags(context.Background(), "")
			return err
		}, "lambda ARN not set"},
		{"ssm path", func() error {
			var _, err = NewSystemsManagerApi(logger).GetParametersByPath("", true)
			return err
		}, "parameter path not set"},
		{"ssm history", func() error {
			var _, err = NewSystemsManagerApi(logger).GetParameterHistory("", true)
			return err
		}, "parameter name not set"},
		{"describe state machine", func() error {
			var _, err = sfnApi.DescribeStateMachine("")
			return err
		}, "state machine ARN not set"},
		{"list executions", func() error {
			var _, err = sfnApi.ListExecutions("", testTime, testTime, true)
			return err
		}, "state machine ARN not set"},
		{"describe execution", func() error {
			var _, err = sfnApi.DescribeExecution("")
			return err
		}, "execution ARN not set"},
		{"execution history", func() error {
			var _, err = sfnApi.GetExecutionHistory("")
			return err
		}, "execution ARN not set"},
		{"s3 list objects", func() error {
			var _, _, err = s3Api.ListObjects("", "", true)
			return err
		}, "bucket name not set"},
		{"s3 download without key", func() error {
			return s3Api.DownloadFile("bucket", "", "file")
		}, "object key not set"},
		{"s3 download without file", func() error {
			return s3Api.DownloadFile("bucket", "key", "")
		}, "file name not set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err = tt.call()
			if err == nil || err.Error() != tt.want {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}

	if fake.calls() != 0 {
		t.Fatalf("expected no requests to be made, got %d", fake.calls())
	}
}

var testTime = time.Unix(0, 0)
