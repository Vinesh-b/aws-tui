package awsapi

import (
	"context"
	"encoding/json"
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
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	snsTypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
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
	if err := NewS3BucketsApi(testLogger()).DownloadFile(context.Background(), "bucket", "key", target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got, err = os.ReadFile(target)
	if err != nil || string(got) != "file contents" {
		t.Fatalf("unexpected file contents %q, err=%v", got, err)
	}
}

func TestS3GetObjectContentReturnsBody(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/plain", "file contents")
	})

	var got, err = NewS3BucketsApi(testLogger()).GetObjectContent(context.Background(), "bucket", "key", 100)

	if err != nil || string(got) != "file contents" {
		t.Fatalf("unexpected content %q, err=%v", got, err)
	}
}

func TestS3GetObjectContentAcceptsObjectExactlyAtTheLimit(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/plain", "12345")
	})

	var got, err = NewS3BucketsApi(testLogger()).GetObjectContent(context.Background(), "bucket", "key", 5)

	if err != nil || string(got) != "12345" {
		t.Fatalf("unexpected content %q, err=%v", got, err)
	}
}

func TestS3GetObjectContentRejectsObjectOverTheLimit(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/plain", "123456")
	})

	var got, err = NewS3BucketsApi(testLogger()).GetObjectContent(context.Background(), "bucket", "key", 5)

	if err == nil || got != nil {
		t.Fatalf("expected an error and no content, got %q, err=%v", got, err)
	}
}

func TestS3GetObjectContentReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "application/xml", "<Error><Code>NoSuchKey</Code></Error>")
	})

	var _, err = NewS3BucketsApi(testLogger()).GetObjectContent(context.Background(), "bucket", "key", 5)

	if err == nil {
		t.Fatal("expected an error for a missing object")
	}
}

func TestDynamoDBListTagsFollowsNextToken(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return jsonResponse(`{"Tags":[{"Key":"a","Value":"1"}],"NextToken":"n1"}`)
		}
		return jsonResponse(`{"Tags":[{"Key":"b","Value":"2"}]}`)
	})

	var tags, err = NewDynamoDBApi(testLogger()).ListTags(context.Background(), true, "arn:aws:dynamodb:::table/t")
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
			var _, err = cwl.ListLogStreams(context.Background(), "", "", true)
			return err
		}, "log group not set"},
		{"log events without stream", func() error {
			var _, err = cwl.ListLogEvents(context.Background(), "group", "", true)
			return err
		}, "log stream not set"},
		{"stop insights query", func() error {
			var _, err = cwl.StopInsightsQuery(context.Background(), "")
			return err
		}, "query ID not set"},
		{"insights results", func() error {
			var _, err = cwl.GetInsightsQueryResults(context.Background(), "")
			return err
		}, "query ID not set"},
		{"insights log record", func() error {
			var _, err = cwl.GetInsightsLogRecord(context.Background(), "")
			return err
		}, "record pointer not set"},
		{"describe table", func() error {
			var _, err = NewDynamoDBApi(logger).DescribeTable(context.Background(), "")
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
		{"secret value", func() error {
			var _, err = NewSecretsManagerApi(logger).GetSecretValue(context.Background(), "")
			return err
		}, "secret ID not set"},
		{"put secret id", func() error {
			return NewSecretsManagerApi(logger).PutSecretValue(context.Background(), "", "value")
		}, "secret ID not set"},
		{"put secret value", func() error {
			return NewSecretsManagerApi(logger).PutSecretValue(context.Background(), "id", "")
		}, "secret value not set"},
		{"ssm path", func() error {
			var _, err = NewSystemsManagerApi(logger).GetParametersByPath(context.Background(), "", true)
			return err
		}, "parameter path not set"},
		{"ssm history", func() error {
			var _, err = NewSystemsManagerApi(logger).GetParameterHistory(context.Background(), "", true)
			return err
		}, "parameter name not set"},
		{"describe state machine", func() error {
			var _, err = sfnApi.DescribeStateMachine(context.Background(), "")
			return err
		}, "state machine ARN not set"},
		{"list executions", func() error {
			var _, err = sfnApi.ListExecutions(context.Background(), "", testTime, testTime, true)
			return err
		}, "state machine ARN not set"},
		{"describe execution", func() error {
			var _, err = sfnApi.DescribeExecution(context.Background(), "")
			return err
		}, "execution ARN not set"},
		{"execution history", func() error {
			var _, err = sfnApi.GetExecutionHistory(context.Background(), "")
			return err
		}, "execution ARN not set"},
		{"start execution", func() error {
			var _, err = sfnApi.StartExecution(context.Background(), "", "", "{}")
			return err
		}, "state machine ARN not set"},
		{"sqs send without queue", func() error {
			var _, err = NewSqsApi(logger).SendMessage(context.Background(), "", "body", "", "", nil)
			return err
		}, "queueUrl not set"},
		{"sqs send without body", func() error {
			var _, err = NewSqsApi(logger).SendMessage(context.Background(), "https://q", "", "", "", nil)
			return err
		}, "message body not set"},
		{"put event without bus", func() error {
			var _, err = NewEventBridgeApi(logger).PutEvent(context.Background(), "", "src", "type", "{}", nil)
			return err
		}, "event bus not set"},
		{"put event without source", func() error {
			var _, err = NewEventBridgeApi(logger).PutEvent(context.Background(), "bus", "", "type", "{}", nil)
			return err
		}, "event source not set"},
		{"put event without detail type", func() error {
			var _, err = NewEventBridgeApi(logger).PutEvent(context.Background(), "bus", "src", "", "{}", nil)
			return err
		}, "event detail type not set"},
		{"put event without detail", func() error {
			var _, err = NewEventBridgeApi(logger).PutEvent(context.Background(), "bus", "src", "type", "", nil)
			return err
		}, "event detail not set"},
		{"sns topic attributes", func() error {
			var _, err = NewSnsApi(logger).GetTopicAttributes(context.Background(), "")
			return err
		}, "topicArn not set"},
		{"sns topic tags", func() error {
			var _, err = NewSnsApi(logger).ListTopicTags(context.Background(), "")
			return err
		}, "topicArn not set"},
		{"sns subscriptions", func() error {
			var _, err = NewSnsApi(logger).ListTopicSubscriptions(context.Background(), "")
			return err
		}, "topicArn not set"},
		{"sns subscribe without topic", func() error {
			var _, err = NewSnsApi(logger).Subscribe(context.Background(), "", "email", "a@example.com")
			return err
		}, "topicArn not set"},
		{"sns subscribe without protocol", func() error {
			var _, err = NewSnsApi(logger).Subscribe(context.Background(), "arn", "", "a@example.com")
			return err
		}, "protocol not set"},
		{"sns subscribe without endpoint", func() error {
			var _, err = NewSnsApi(logger).Subscribe(context.Background(), "arn", "email", "")
			return err
		}, "endpoint not set"},
		{"sns unsubscribe without arn", func() error {
			return NewSnsApi(logger).Unsubscribe(context.Background(), "")
		}, "subscriptionArn not set"},
		{"sns unsubscribe pending", func() error {
			return NewSnsApi(logger).Unsubscribe(context.Background(), "PendingConfirmation")
		}, "a subscription that is pending confirmation can't be deleted"},
		{"sns delete topic without arn", func() error {
			return NewSnsApi(logger).DeleteTopic(context.Background(), "")
		}, "topicArn not set"},
		{"sns create topic without name", func() error {
			var _, err = NewSnsApi(logger).CreateTopic(context.Background(), "", false, false)
			return err
		}, "topic name not set"},
		{"sns publish without topic", func() error {
			var _, err = NewSnsApi(logger).Publish(context.Background(), "", "hi", "", "", "", nil)
			return err
		}, "topicArn not set"},
		{"sns publish without message", func() error {
			var _, err = NewSnsApi(logger).Publish(context.Background(), "arn", "", "", "", "", nil)
			return err
		}, "message not set"},
		{"s3 list objects", func() error {
			var _, _, err = s3Api.ListObjects(context.Background(), "", "", true)
			return err
		}, "bucket name not set"},
		{"s3 download without key", func() error {
			return s3Api.DownloadFile(context.Background(), "bucket", "", "file")
		}, "object key not set"},
		{"metric data without metric name", func() error {
			var _, err = NewCloudWatchMetricsApi(logger).GetMetricData(
				context.Background(), cwTypes.Metric{}, "Average", 60, testTime, testTime,
			)
			return err
		}, "metric name not set"},
		{"s3 content without bucket", func() error {
			var _, err = s3Api.GetObjectContent(context.Background(), "", "key", 1)
			return err
		}, "bucket name not set"},
		{"s3 content without key", func() error {
			var _, err = s3Api.GetObjectContent(context.Background(), "bucket", "", 1)
			return err
		}, "object key not set"},
		{"s3 download without file", func() error {
			return s3Api.DownloadFile(context.Background(), "bucket", "key", "")
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

func TestDynamoDBPutItemSendsTheItemToTheTable(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{}`)
	})
	var item = map[string]any{"id": "a1", "count": json.Number("12345678901234567890"), "tags": []any{"x"}}

	var err = NewDynamoDBApi(testLogger()).PutItem(context.Background(), "orders", item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{`"TableName":"orders"`, `"id":{"S":"a1"}`, `"count":{"N":"12345678901234567890"}`, `"tags":{"L":[{"S":"x"}]}`} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestDynamoDBPutItemRejectsMissingArguments(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{}`)
	})
	var api = NewDynamoDBApi(testLogger())

	if err := api.PutItem(context.Background(), "", map[string]any{"id": "a"}); err == nil {
		t.Error("expected an error without a table name")
	}
	if err := api.PutItem(context.Background(), "orders", nil); err == nil {
		t.Error("expected an error without attributes")
	}
	if fake.calls() != 0 {
		t.Errorf("expected no requests but got %d", fake.calls())
	}
}

func TestDynamoDBPutItemReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "application/x-amz-json-1.0", `{"__type":"ResourceNotFoundException","message":"missing"}`)
	})

	var err = NewDynamoDBApi(testLogger()).PutItem(context.Background(), "orders", map[string]any{"id": "a"})

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSfnStartExecutionSendsInputAndName(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{"executionArn":"arn:exec","startDate":1700000000}`)
	})

	var output, err = NewStateMachineApi(testLogger()).
		StartExecution(context.Background(), "arn:sm", "run-1", `{"a":1}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if aws.ToString(output.ExecutionArn) != "arn:exec" {
		t.Errorf("unexpected execution ARN %q", aws.ToString(output.ExecutionArn))
	}
	var body = fake.requests[0]
	for _, want := range []string{`"stateMachineArn":"arn:sm"`, `"name":"run-1"`, `"input":"{\"a\":1}"`} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSfnStartExecutionDefaultsToEmptyObjectWithoutName(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{"executionArn":"arn:exec","startDate":1700000000}`)
	})

	var _, err = NewStateMachineApi(testLogger()).StartExecution(context.Background(), "arn:sm", "", "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	if !strings.Contains(body, `"input":"{}"`) || strings.Contains(body, `"name"`) {
		t.Errorf("unexpected request %s", body)
	}
}

func TestSfnStartExecutionReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "application/x-amz-json-1.0", `{"__type":"InvalidArn","message":"bad"}`)
	})

	var _, err = NewStateMachineApi(testLogger()).StartExecution(context.Background(), "arn:sm", "", "{}")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSqsSendMessageSendsBodyGroupAndDeduplicationId(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/x-amz-json-1.0", `{"MessageId":"m-1","SequenceNumber":"42"}`)
	})

	var output, err = NewSqsApi(testLogger()).
		SendMessage(context.Background(), "https://sqs.us-east-1.amazonaws.com/1/q.fifo", `{"a":1}`, "g1", "d1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if aws.ToString(output.MessageId) != "m-1" {
		t.Errorf("unexpected message ID %q", aws.ToString(output.MessageId))
	}
	var body = fake.requests[0]
	for _, want := range []string{`"MessageBody":"{\"a\":1}"`, `"MessageGroupId":"g1"`, `"MessageDeduplicationId":"d1"`, `q.fifo`} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSqsSendMessageOmitsEmptyGroupAndDeduplicationId(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/x-amz-json-1.0", `{"MessageId":"m-1"}`)
	})

	var _, err = NewSqsApi(testLogger()).SendMessage(context.Background(), "https://q", "hello", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body := fake.requests[0]; strings.Contains(body, "MessageGroupId") || strings.Contains(body, "MessageDeduplicationId") {
		t.Errorf("unexpected request %s", body)
	}
}

func TestSqsSendMessageReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "application/x-amz-json-1.0", `{"__type":"QueueDoesNotExist","message":"gone"}`)
	})

	var _, err = NewSqsApi(testLogger()).SendMessage(context.Background(), "https://q", "hello", "", "", nil)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSqsSendMessageSendsMessageAttributes(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/x-amz-json-1.0", `{"MessageId":"m-1"}`)
	})
	var attributes = map[string]sqsTypes.MessageAttributeValue{
		"Source": {DataType: aws.String("String"), StringValue: aws.String("tui")},
		"Count":  {DataType: aws.String("Number"), StringValue: aws.String("3")},
	}

	var _, err = NewSqsApi(testLogger()).SendMessage(context.Background(), "https://q", "hello", "", "", attributes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{`"Source":{"DataType":"String","StringValue":"tui"}`, `"Count":{"DataType":"Number","StringValue":"3"}`} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestEventBridgePutEventSendsTheEventToTheBus(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{"FailedEntryCount":0,"Entries":[{"EventId":"e-1"}]}`)
	})

	var id, err = NewEventBridgeApi(testLogger()).PutEvent(
		context.Background(), "arn:aws:events:us-east-1:1:event-bus/orders", "my.app", "OrderPlaced",
		`{"id":1}`, []string{"arn:aws:s3:::b"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id != "e-1" {
		t.Errorf("event ID = %q, want e-1", id)
	}
	var body = fake.requests[0]
	for _, want := range []string{
		`"EventBusName":"arn:aws:events:us-east-1:1:event-bus/orders"`, `"Source":"my.app"`,
		`"DetailType":"OrderPlaced"`, `"Detail":"{\"id\":1}"`, `"Resources":["arn:aws:s3:::b"]`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestEventBridgePutEventReportsARejectedEvent(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{"FailedEntryCount":1,"Entries":[{"ErrorCode":"InvalidArgument","ErrorMessage":"bad detail"}]}`)
	})

	var _, err = NewEventBridgeApi(testLogger()).PutEvent(context.Background(), "bus", "src", "type", "{}", nil)

	if err == nil || !strings.Contains(err.Error(), "InvalidArgument") || !strings.Contains(err.Error(), "bad detail") {
		t.Fatalf("error = %v, want one with the rejection code and message", err)
	}
}

func TestEventBridgePutEventReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "application/x-amz-json-1.1", `{"__type":"ResourceNotFoundException","message":"gone"}`)
	})

	var _, err = NewEventBridgeApi(testLogger()).PutEvent(context.Background(), "bus", "src", "type", "{}", nil)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSnsListTopicsSortedAcrossPages(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return response(200, "text/xml", `<ListTopicsResponse><ListTopicsResult><Topics><member><TopicArn>arn:aws:sns:us-east-1:1:zeta</TopicArn></member></Topics><NextToken>t</NextToken></ListTopicsResult></ListTopicsResponse>`)
		}
		return response(200, "text/xml", `<ListTopicsResponse><ListTopicsResult><Topics><member><TopicArn>arn:aws:sns:us-east-1:1:alpha</TopicArn></member></Topics></ListTopicsResult></ListTopicsResponse>`)
	})

	var topics, err = NewSnsApi(testLogger()).ListTopics(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(topics) != 2 || !strings.HasSuffix(topics[0], ":alpha") || !strings.HasSuffix(topics[1], ":zeta") {
		t.Fatalf("unexpected topics %v", topics)
	}
}

func TestSnsGetTopicAttributes(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", `<GetTopicAttributesResponse><GetTopicAttributesResult><Attributes><entry><key>DisplayName</key><value>Orders</value></entry></Attributes></GetTopicAttributesResult></GetTopicAttributesResponse>`)
	})

	var attributes, err = NewSnsApi(testLogger()).GetTopicAttributes(context.Background(), "arn:aws:sns:us-east-1:1:orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attributes["DisplayName"] != "Orders" {
		t.Fatalf("unexpected attributes %v", attributes)
	}
}

func TestSnsListTopicTags(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", `<ListTagsForResourceResponse><ListTagsForResourceResult><Tags><member><Key>env</Key><Value>prod</Value></member></Tags></ListTagsForResourceResult></ListTagsForResourceResponse>`)
	})

	var tags, err = NewSnsApi(testLogger()).ListTopicTags(context.Background(), "arn:aws:sns:us-east-1:1:orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tags) != 1 || aws.ToString(tags[0].Key) != "env" || aws.ToString(tags[0].Value) != "prod" {
		t.Fatalf("unexpected tags %v", tags)
	}
}

func TestSnsGetTopicAttributesReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>NotFound</Code><Message>gone</Message></Error></ErrorResponse>`)
	})

	var _, err = NewSnsApi(testLogger()).GetTopicAttributes(context.Background(), "arn:aws:sns:us-east-1:1:orders")

	if err == nil {
		t.Fatal("expected an error")
	}
}

const snsPublishResponse = `<PublishResponse><PublishResult><MessageId>m-1</MessageId><SequenceNumber>42</SequenceNumber></PublishResult></PublishResponse>`

func TestSnsPublishSendsMessageAndOptions(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", snsPublishResponse)
	})
	var attributes = map[string]snsTypes.MessageAttributeValue{
		"Source": {DataType: aws.String("String"), StringValue: aws.String("tui")},
	}

	var output, err = NewSnsApi(testLogger()).Publish(
		context.Background(), "arn:aws:sns:us-east-1:1:orders.fifo", "hello", "Greeting", "g1", "d1", attributes,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if aws.ToString(output.MessageId) != "m-1" || aws.ToString(output.SequenceNumber) != "42" {
		t.Errorf("unexpected output %v", output)
	}
	var body = fake.requests[0]
	for _, want := range []string{
		"TopicArn=arn%3Aaws%3Asns%3Aus-east-1%3A1%3Aorders.fifo", "Message=hello", "Subject=Greeting",
		"MessageGroupId=g1", "MessageDeduplicationId=d1",
		"MessageAttributes.entry.1.Name=Source", "MessageAttributes.entry.1.Value.DataType=String",
		"MessageAttributes.entry.1.Value.StringValue=tui",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSnsPublishOmitsEmptyOptions(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", snsPublishResponse)
	})

	var _, err = NewSnsApi(testLogger()).Publish(context.Background(), "arn", "hello", "", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, unwanted := range []string{"Subject", "MessageGroupId", "MessageDeduplicationId", "MessageAttributes"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("request %s contains %s", body, unwanted)
		}
	}
}

func TestSnsPublishReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>NotFound</Code><Message>gone</Message></Error></ErrorResponse>`)
	})

	var _, err = NewSnsApi(testLogger()).Publish(context.Background(), "arn", "hello", "", "", "", nil)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSnsListTopicSubscriptionsFollowsNextToken(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return response(200, "text/xml", `<ListSubscriptionsByTopicResponse><ListSubscriptionsByTopicResult><Subscriptions><member><SubscriptionArn>arn:sub:1</SubscriptionArn><Protocol>email</Protocol><Endpoint>a@example.com</Endpoint></member></Subscriptions><NextToken>t</NextToken></ListSubscriptionsByTopicResult></ListSubscriptionsByTopicResponse>`)
		}
		return response(200, "text/xml", `<ListSubscriptionsByTopicResponse><ListSubscriptionsByTopicResult><Subscriptions><member><SubscriptionArn>PendingConfirmation</SubscriptionArn><Protocol>sqs</Protocol><Endpoint>arn:aws:sqs:q</Endpoint></member></Subscriptions></ListSubscriptionsByTopicResult></ListSubscriptionsByTopicResponse>`)
	})

	var subs, err = NewSnsApi(testLogger()).ListTopicSubscriptions(context.Background(), "arn:aws:sns:us-east-1:1:orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.calls() != 2 || !strings.Contains(fake.requests[1], "NextToken=t") {
		t.Fatalf("second request did not send the next token: %v", fake.requests)
	}
	if len(subs) != 2 || aws.ToString(subs[0].Endpoint) != "a@example.com" || aws.ToString(subs[1].Protocol) != "sqs" {
		t.Fatalf("unexpected subscriptions %v", subs)
	}
}

func TestSnsListTopicSubscriptionsReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>NotFound</Code><Message>gone</Message></Error></ErrorResponse>`)
	})

	var _, err = NewSnsApi(testLogger()).ListTopicSubscriptions(context.Background(), "arn")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSnsSubscribeSendsTheSubscriptionAndReturnsItsArn(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", `<SubscribeResponse><SubscribeResult><SubscriptionArn>arn:aws:sns:us-east-1:1:orders:abc</SubscriptionArn></SubscribeResult></SubscribeResponse>`)
	})

	var arn, err = NewSnsApi(testLogger()).Subscribe(
		context.Background(), "arn:aws:sns:us-east-1:1:orders", "email", "a@example.com",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if arn != "arn:aws:sns:us-east-1:1:orders:abc" {
		t.Errorf("subscription ARN = %q", arn)
	}
	var body = fake.requests[0]
	for _, want := range []string{
		"Action=Subscribe", "Protocol=email", "Endpoint=a%40example.com", "ReturnSubscriptionArn=true",
		"TopicArn=arn%3Aaws%3Asns%3Aus-east-1%3A1%3Aorders",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSnsSubscribeReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>InvalidParameter</Code><Message>bad endpoint</Message></Error></ErrorResponse>`)
	})

	var _, err = NewSnsApi(testLogger()).Subscribe(context.Background(), "arn", "email", "nope")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSnsUnsubscribeDeletesTheSubscription(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", `<UnsubscribeResponse><ResponseMetadata><RequestId>r</RequestId></ResponseMetadata></UnsubscribeResponse>`)
	})

	var err = NewSnsApi(testLogger()).Unsubscribe(context.Background(), "arn:aws:sns:us-east-1:1:orders:abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{"Action=Unsubscribe", "SubscriptionArn=arn%3Aaws%3Asns%3Aus-east-1%3A1%3Aorders%3Aabc"} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSnsUnsubscribeReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>NotFound</Code><Message>gone</Message></Error></ErrorResponse>`)
	})

	var err = NewSnsApi(testLogger()).Unsubscribe(context.Background(), "arn:aws:sns:us-east-1:1:orders:abc")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSnsDeleteTopicDeletesTheTopic(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", `<DeleteTopicResponse><ResponseMetadata><RequestId>r</RequestId></ResponseMetadata></DeleteTopicResponse>`)
	})

	var err = NewSnsApi(testLogger()).DeleteTopic(context.Background(), "arn:aws:sns:us-east-1:1:orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{"Action=DeleteTopic", "TopicArn=arn%3Aaws%3Asns%3Aus-east-1%3A1%3Aorders"} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSnsDeleteTopicReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(403, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>AuthorizationError</Code><Message>denied</Message></Error></ErrorResponse>`)
	})

	var err = NewSnsApi(testLogger()).DeleteTopic(context.Background(), "arn:aws:sns:us-east-1:1:orders")

	if err == nil {
		t.Fatal("expected an error")
	}
}

const snsCreateTopicResponse = `<CreateTopicResponse><CreateTopicResult><TopicArn>arn:aws:sns:us-east-1:1:orders</TopicArn></CreateTopicResult></CreateTopicResponse>`

func TestSnsCreateTopicCreatesAStandardTopic(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", snsCreateTopicResponse)
	})

	var arn, err = NewSnsApi(testLogger()).CreateTopic(context.Background(), "orders", false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if arn != "arn:aws:sns:us-east-1:1:orders" {
		t.Errorf("topic ARN = %q", arn)
	}
	var body = fake.requests[0]
	if !strings.Contains(body, "Name=orders") {
		t.Errorf("request %s does not contain the name", body)
	}
	if strings.Contains(body, "FifoTopic") || strings.Contains(body, "ContentBasedDeduplication") {
		t.Errorf("request %s has FIFO settings for a standard topic", body)
	}
}

func TestSnsCreateTopicCreatesAFifoTopic(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", snsCreateTopicResponse)
	})

	var _, err = NewSnsApi(testLogger()).CreateTopic(context.Background(), "orders.fifo", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{
		"Name=orders.fifo", "Attributes.entry.1.key=ContentBasedDeduplication", "Attributes.entry.2.key=FifoTopic",
		"Attributes.entry.1.value=true", "Attributes.entry.2.value=true",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestSnsCreateTopicOmitsContentBasedDeduplicationUnlessAsked(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "text/xml", snsCreateTopicResponse)
	})

	var _, err = NewSnsApi(testLogger()).CreateTopic(context.Background(), "orders.fifo", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	if !strings.Contains(body, "FifoTopic") || strings.Contains(body, "ContentBasedDeduplication") {
		t.Errorf("unexpected request %s", body)
	}
}

func TestSnsCreateTopicReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>InvalidParameter</Code><Message>bad name</Message></Error></ErrorResponse>`)
	})

	var _, err = NewSnsApi(testLogger()).CreateTopic(context.Background(), "bad name", false, false)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDynamoDBUpdateItemOnlyChangesTheGivenAttributes(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{}`)
	})

	var err = NewDynamoDBApi(testLogger()).UpdateItem(
		context.Background(), "orders",
		map[string]any{"id": "a1"},
		map[string]any{"status": "shipped", "count": json.Number("12345678901234567890")},
		[]string{"note"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{
		`"TableName":"orders"`, `"Key":{"id":{"S":"a1"}}`, `"UpdateExpression":"REMOVE #`, `\nSET #`,
		`"ConditionExpression":"attribute_exists (#`, `{"S":"shipped"}`, `{"N":"12345678901234567890"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestDynamoDBUpdateItemRejectsInvalidArguments(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{}`)
	})
	var api = NewDynamoDBApi(testLogger())
	var key = map[string]any{"id": "a1"}
	var set = map[string]any{"status": "shipped"}

	for name, err := range map[string]error{
		"no table":     api.UpdateItem(context.Background(), "", key, set, nil),
		"no key":       api.UpdateItem(context.Background(), "orders", nil, set, nil),
		"no changes":   api.UpdateItem(context.Background(), "orders", key, nil, nil),
		"dotted name":  api.UpdateItem(context.Background(), "orders", key, map[string]any{"a.b": 1}, nil),
		"bracket name": api.UpdateItem(context.Background(), "orders", key, nil, []string{"a[0]"}),
	} {
		if err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("expected no requests but got %d", fake.calls())
	}
}

func TestDynamoDBUpdateItemReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "application/x-amz-json-1.0", `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"The conditional request failed"}`)
	})

	var err = NewDynamoDBApi(testLogger()).UpdateItem(
		context.Background(), "orders", map[string]any{"id": "a1"}, map[string]any{"status": "x"}, nil,
	)

	if err == nil || !strings.Contains(err.Error(), "ConditionalCheckFailed") {
		t.Fatalf("error = %v, want the conditional check failure", err)
	}
}

func TestDynamoDBDeleteItemDeletesTheItemWithTheKey(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{}`)
	})

	var err = NewDynamoDBApi(testLogger()).DeleteItem(
		context.Background(), "orders", map[string]any{"id": "a1", "ts": json.Number("5")},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body = fake.requests[0]
	for _, want := range []string{
		`"TableName":"orders"`, `"id":{"S":"a1"}`, `"ts":{"N":"5"}`,
		`"ConditionExpression":"attribute_exists (#0)"`, `"ExpressionAttributeNames":{"#0":"id"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request %s does not contain %s", body, want)
		}
	}
}

func TestDynamoDBDeleteItemRejectsInvalidArguments(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return jsonResponse(`{}`)
	})
	var api = NewDynamoDBApi(testLogger())

	for name, err := range map[string]error{
		"no table":    api.DeleteItem(context.Background(), "", map[string]any{"id": "a1"}),
		"no key":      api.DeleteItem(context.Background(), "orders", nil),
		"dotted name": api.DeleteItem(context.Background(), "orders", map[string]any{"a.b": "x"}),
	} {
		if err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("expected no requests but got %d", fake.calls())
	}
}

func TestDynamoDBDeleteItemReportsAMissingItem(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(400, "application/x-amz-json-1.0", `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"The conditional request failed"}`)
	})

	var err = NewDynamoDBApi(testLogger()).DeleteItem(context.Background(), "orders", map[string]any{"id": "a1"})

	if err == nil || !strings.Contains(err.Error(), "ConditionalCheckFailed") {
		t.Fatalf("error = %v, want the conditional check failure", err)
	}
}

// iamAction is the action of an IAM request, which is sent as a form
func iamAction(body string) string {
	for _, field := range strings.Split(body, "&") {
		if action, found := strings.CutPrefix(field, "Action="); found {
			return action
		}
	}
	return ""
}

func iamResponse(action string, result string) *http.Response {
	return response(200, "text/xml",
		"<"+action+"Response><"+action+"Result>"+result+"</"+action+"Result></"+action+"Response>")
}

func TestIamListRolesSortedAcrossPages(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return iamResponse("ListRoles", `<IsTruncated>true</IsTruncated><Marker>m</Marker><Roles><member><RoleName>zeta</RoleName></member></Roles>`)
		}
		return iamResponse("ListRoles", `<IsTruncated>false</IsTruncated><Roles><member><RoleName>alpha</RoleName></member></Roles>`)
	})

	var roles, err = NewIamApi(testLogger()).ListRoles(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(roles) != 2 || aws.ToString(roles[0].RoleName) != "alpha" || aws.ToString(roles[1].RoleName) != "zeta" {
		t.Fatalf("unexpected roles %v", roles)
	}
}

func TestIamGetRoleDecodesTrustPolicy(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return iamResponse("GetRole", `<Role><RoleName>orders</RoleName><AssumeRolePolicyDocument>%7B%22Version%22%3A%222012-10-17%22%7D</AssumeRolePolicyDocument></Role>`)
	})

	var role, err = NewIamApi(testLogger()).GetRole(context.Background(), "orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := aws.ToString(role.AssumeRolePolicyDocument); got != `{"Version":"2012-10-17"}` {
		t.Fatalf("trust policy = %q", got)
	}
}

func TestIamListRolePoliciesGivesAttachedThenInline(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch iamAction(body) {
		case "ListAttachedRolePolicies":
			return iamResponse("ListAttachedRolePolicies", `<IsTruncated>false</IsTruncated><AttachedPolicies>`+
				`<member><PolicyName>ReadOnly</PolicyName><PolicyArn>arn:aws:iam::aws:policy/ReadOnly</PolicyArn></member>`+
				`<member><PolicyName>Logs</PolicyName><PolicyArn>arn:aws:iam::1:policy/Logs</PolicyArn></member>`+
				`</AttachedPolicies>`)
		case "ListRolePolicies":
			return iamResponse("ListRolePolicies", `<IsTruncated>false</IsTruncated><PolicyNames><member>queue-access</member></PolicyNames>`)
		}
		return response(400, "text/xml", `<ErrorResponse><Error><Code>Unexpected</Code></Error></ErrorResponse>`)
	})

	var policies, err = NewIamApi(testLogger()).ListRolePolicies(context.Background(), "orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want = []PrincipalPolicy{
		{Name: "Logs", Arn: "arn:aws:iam::1:policy/Logs"},
		{Name: "ReadOnly", Arn: "arn:aws:iam::aws:policy/ReadOnly"},
		{Name: "queue-access"},
	}
	if len(policies) != len(want) {
		t.Fatalf("policies = %v, want %v", policies, want)
	}
	for i := range want {
		if policies[i] != want[i] {
			t.Errorf("policy %d = %v, want %v", i, policies[i], want[i])
		}
	}
}

func TestIamGetRolePolicyDocumentOfInlinePolicy(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return iamResponse("GetRolePolicy", `<PolicyDocument>%7B%22Statement%22%3A%5B%5D%7D</PolicyDocument>`)
	})

	var document, err = NewIamApi(testLogger()).GetRolePolicyDocument(
		context.Background(), "orders", PrincipalPolicy{Name: "queue-access"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if document != `{"Statement":[]}` {
		t.Errorf("document = %q", document)
	}
	if got := iamAction(fake.requests[0]); got != "GetRolePolicy" {
		t.Errorf("action = %q, want GetRolePolicy", got)
	}
}

func TestIamGetRolePolicyDocumentOfManagedPolicyUsesDefaultVersion(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch iamAction(body) {
		case "GetPolicy":
			return iamResponse("GetPolicy", `<Policy><DefaultVersionId>v3</DefaultVersionId></Policy>`)
		case "GetPolicyVersion":
			return iamResponse("GetPolicyVersion", `<PolicyVersion><Document>%7B%22Version%22%3A%222012-10-17%22%7D</Document></PolicyVersion>`)
		}
		return response(400, "text/xml", `<ErrorResponse><Error><Code>Unexpected</Code></Error></ErrorResponse>`)
	})

	var document, err = NewIamApi(testLogger()).GetRolePolicyDocument(
		context.Background(), "orders", PrincipalPolicy{Name: "Logs", Arn: "arn:aws:iam::1:policy/Logs"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if document != `{"Version":"2012-10-17"}` {
		t.Errorf("document = %q", document)
	}
	if fake.calls() != 2 || !strings.Contains(fake.requests[1], "VersionId=v3") {
		t.Errorf("requests = %v, want the default version asked for", fake.requests)
	}
}

func TestIamGetRoleReportsServiceErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>NoSuchEntity</Code><Message>gone</Message></Error></ErrorResponse>`)
	})

	var _, err = NewIamApi(testLogger()).GetRole(context.Background(), "orders")

	if err == nil || !strings.Contains(err.Error(), "NoSuchEntity") {
		t.Fatalf("error = %v, want the service error", err)
	}
}

func TestIamMissingArgumentsAreRejected(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return iamResponse("GetRole", "")
	})
	var api = NewIamApi(testLogger())
	var ctx = context.Background()

	var tests = []struct {
		name string
		call func() error
		want string
	}{
		{"get role", func() error { var _, err = api.GetRole(ctx, ""); return err }, "role name not set"},
		{"role policies", func() error { var _, err = api.ListRolePolicies(ctx, ""); return err }, "role name not set"},
		{"role tags", func() error { var _, err = api.ListRoleTags(ctx, ""); return err }, "role name not set"},
		{"policy document without role", func() error {
			var _, err = api.GetRolePolicyDocument(ctx, "", PrincipalPolicy{Name: "p"})
			return err
		}, "role name not set"},
		{"policy document without policy", func() error {
			var _, err = api.GetRolePolicyDocument(ctx, "orders", PrincipalPolicy{})
			return err
		}, "policy name not set"},
	}

	for _, tt := range tests {
		if err := tt.call(); err == nil || err.Error() != tt.want {
			t.Errorf("%s: error = %v, want %q", tt.name, err, tt.want)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("calls = %d, want none for missing arguments", fake.calls())
	}
}

func TestIamListUserPoliciesGivesAttachedThenInline(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch iamAction(body) {
		case "ListAttachedUserPolicies":
			return iamResponse("ListAttachedUserPolicies", `<IsTruncated>false</IsTruncated><AttachedPolicies>`+
				`<member><PolicyName>ReadOnly</PolicyName><PolicyArn>arn:aws:iam::aws:policy/ReadOnly</PolicyArn></member>`+
				`</AttachedPolicies>`)
		case "ListUserPolicies":
			return iamResponse("ListUserPolicies", `<IsTruncated>false</IsTruncated><PolicyNames><member>deploy</member><member>build</member></PolicyNames>`)
		}
		return response(400, "text/xml", `<ErrorResponse><Error><Code>Unexpected</Code></Error></ErrorResponse>`)
	})

	var policies, err = NewIamApi(testLogger()).ListUserPolicies(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want = []PrincipalPolicy{
		{Name: "ReadOnly", Arn: "arn:aws:iam::aws:policy/ReadOnly"},
		{Name: "build"},
		{Name: "deploy"},
	}
	if len(policies) != len(want) {
		t.Fatalf("policies = %v, want %v", policies, want)
	}
	for i := range want {
		if policies[i] != want[i] {
			t.Errorf("policy %d = %v, want %v", i, policies[i], want[i])
		}
	}
}

func TestIamGetUserPolicyDocumentOfInlinePolicy(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return iamResponse("GetUserPolicy", `<PolicyDocument>%7B%22Statement%22%3A%5B%5D%7D</PolicyDocument>`)
	})

	var document, err = NewIamApi(testLogger()).GetUserPolicyDocument(
		context.Background(), "alice", PrincipalPolicy{Name: "deploy"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if document != `{"Statement":[]}` {
		t.Errorf("document = %q", document)
	}
	if got := iamAction(fake.requests[0]); got != "GetUserPolicy" {
		t.Errorf("action = %q, want GetUserPolicy", got)
	}
}

func TestIamListUsersGroupsAndAccessKeys(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch iamAction(body) {
		case "ListUsers":
			return iamResponse("ListUsers", `<IsTruncated>false</IsTruncated><Users><member><UserName>bob</UserName></member><member><UserName>alice</UserName></member></Users>`)
		case "ListGroupsForUser":
			return iamResponse("ListGroupsForUser", `<IsTruncated>false</IsTruncated><Groups><member><GroupName>ops</GroupName></member><member><GroupName>dev</GroupName></member></Groups>`)
		case "ListAccessKeys":
			return iamResponse("ListAccessKeys", `<IsTruncated>false</IsTruncated><AccessKeyMetadata><member><AccessKeyId>AKIA1</AccessKeyId><Status>Active</Status></member></AccessKeyMetadata>`)
		}
		return response(400, "text/xml", `<ErrorResponse><Error><Code>Unexpected</Code></Error></ErrorResponse>`)
	})
	var api = NewIamApi(testLogger())
	var ctx = context.Background()

	var users, err = api.ListUsers(ctx, true)
	if err != nil || len(users) != 2 || aws.ToString(users[0].UserName) != "alice" {
		t.Errorf("users = %v, %v, want alice then bob", users, err)
	}

	var groups, groupsErr = api.ListUserGroups(ctx, "alice")
	if groupsErr != nil || len(groups) != 2 || aws.ToString(groups[0].GroupName) != "dev" {
		t.Errorf("groups = %v, %v, want dev then ops", groups, groupsErr)
	}

	var keys, keysErr = api.ListUserAccessKeys(ctx, "alice")
	if keysErr != nil || len(keys) != 1 || aws.ToString(keys[0].AccessKeyId) != "AKIA1" || keys[0].Status != "Active" {
		t.Errorf("access keys = %v, %v", keys, keysErr)
	}
}

func TestIamListPoliciesGivesCustomerThenAttachedAwsPolicies(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if strings.Contains(body, "Scope=Local") {
			return iamResponse("ListPolicies", `<IsTruncated>false</IsTruncated><Policies><member><PolicyName>orders</PolicyName></member></Policies>`)
		}
		return iamResponse("ListPolicies", `<IsTruncated>false</IsTruncated><Policies><member><PolicyName>ReadOnly</PolicyName></member><member><PolicyName>Admin</PolicyName></member></Policies>`)
	})

	var policies, err = NewIamApi(testLogger()).ListPolicies(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var names []string
	for _, policy := range policies {
		names = append(names, aws.ToString(policy.PolicyName))
	}
	if strings.Join(names, ",") != "orders,Admin,ReadOnly" {
		t.Errorf("policies = %v, want the customer policy then the AWS ones sorted", names)
	}
	if fake.calls() != 2 || !strings.Contains(fake.requests[1], "Scope=AWS") || !strings.Contains(fake.requests[1], "OnlyAttached=true") {
		t.Errorf("requests = %v, want only the attached AWS policies asked for", fake.requests)
	}
}

func TestIamGetPolicyDocumentUsesDefaultVersion(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch iamAction(body) {
		case "GetPolicy":
			return iamResponse("GetPolicy", `<Policy><DefaultVersionId>v2</DefaultVersionId></Policy>`)
		case "GetPolicyVersion":
			if !strings.Contains(body, "VersionId=v2") {
				break
			}
			return iamResponse("GetPolicyVersion", `<PolicyVersion><Document>%7B%7D</Document></PolicyVersion>`)
		}
		return response(400, "text/xml", `<ErrorResponse><Error><Code>Unexpected</Code></Error></ErrorResponse>`)
	})

	var document, err = NewIamApi(testLogger()).GetPolicyDocument(context.Background(), "arn:aws:iam::1:policy/orders")

	if err != nil || document != "{}" {
		t.Errorf("document = %q, %v", document, err)
	}
}

func TestIamListPolicyEntitiesGivesRolesUsersThenGroups(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return iamResponse("ListEntitiesForPolicy", `<IsTruncated>false</IsTruncated>`+
			`<PolicyGroups><member><GroupName>dev</GroupName></member></PolicyGroups>`+
			`<PolicyUsers><member><UserName>bob</UserName></member><member><UserName>alice</UserName></member></PolicyUsers>`+
			`<PolicyRoles><member><RoleName>orders</RoleName></member></PolicyRoles>`)
	})

	var entities, err = NewIamApi(testLogger()).ListPolicyEntities(context.Background(), "arn:aws:iam::1:policy/orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want = []PolicyEntity{
		{Name: "orders", Type: "Role"},
		{Name: "alice", Type: "User"},
		{Name: "bob", Type: "User"},
		{Name: "dev", Type: "Group"},
	}
	if len(entities) != len(want) {
		t.Fatalf("entities = %v, want %v", entities, want)
	}
	for i := range want {
		if entities[i] != want[i] {
			t.Errorf("entity %d = %v, want %v", i, entities[i], want[i])
		}
	}
}

func TestIamUserAndPolicyMissingArgumentsAreRejected(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return iamResponse("GetUser", "")
	})
	var api = NewIamApi(testLogger())
	var ctx = context.Background()

	var tests = []struct {
		name string
		call func() error
		want string
	}{
		{"get user", func() error { var _, err = api.GetUser(ctx, ""); return err }, "user name not set"},
		{"user policies", func() error { var _, err = api.ListUserPolicies(ctx, ""); return err }, "user name not set"},
		{"user groups", func() error { var _, err = api.ListUserGroups(ctx, ""); return err }, "user name not set"},
		{"user access keys", func() error { var _, err = api.ListUserAccessKeys(ctx, ""); return err }, "user name not set"},
		{"user tags", func() error { var _, err = api.ListUserTags(ctx, ""); return err }, "user name not set"},
		{"user policy document", func() error {
			var _, err = api.GetUserPolicyDocument(ctx, "alice", PrincipalPolicy{})
			return err
		}, "policy name not set"},
		{"get policy", func() error { var _, err = api.GetPolicy(ctx, ""); return err }, "policy ARN not set"},
		{"policy document", func() error { var _, err = api.GetPolicyDocument(ctx, ""); return err }, "policy ARN not set"},
		{"policy entities", func() error { var _, err = api.ListPolicyEntities(ctx, ""); return err }, "policy ARN not set"},
		{"policy tags", func() error { var _, err = api.ListPolicyTags(ctx, ""); return err }, "policy ARN not set"},
	}

	for _, tt := range tests {
		if err := tt.call(); err == nil || err.Error() != tt.want {
			t.Errorf("%s: error = %v, want %q", tt.name, err, tt.want)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("calls = %d, want none for missing arguments", fake.calls())
	}
}

func TestLambdaListEventSourceMappingsSortedAcrossPages(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return response(200, "application/json",
				`{"EventSourceMappings":[{"EventSourceArn":"arn:aws:sqs:us-east-1:1:zeta"}],"NextMarker":"m"}`)
		}
		return response(200, "application/json",
			`{"EventSourceMappings":[{"EventSourceArn":"arn:aws:sqs:us-east-1:1:alpha"}]}`)
	})
	var requests []*http.Request
	var original = fake.handler
	fake.handler = func(call int, req *http.Request, body string) *http.Response {
		requests = append(requests, req)
		return original(call, req, body)
	}

	var mappings, err = NewLambdaApi(testLogger()).ListEventSourceMappings(
		context.Background(), "arn:aws:lambda:us-east-1:1:function:orders",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mappings) != 2 || !strings.HasSuffix(aws.ToString(mappings[0].EventSourceArn), ":alpha") {
		t.Fatalf("mappings = %v, want alpha then zeta", mappings)
	}
	if got := requests[0].URL.Query().Get("FunctionName"); got != "arn:aws:lambda:us-east-1:1:function:orders" {
		t.Errorf("FunctionName = %q, want the lambda", got)
	}
}

func TestLambdaGetPolicy(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/json", `{"Policy":"{\"Version\":\"2012-10-17\"}","RevisionId":"r1"}`)
	})

	var policy, err = NewLambdaApi(testLogger()).GetPolicy(context.Background(), "arn:aws:lambda:us-east-1:1:function:orders")

	if err != nil || policy != `{"Version":"2012-10-17"}` {
		t.Errorf("policy = %q, %v", policy, err)
	}
}

func lambdaErrorResponse(status int, errorType string) *http.Response {
	var resp = response(status, "application/json", `{"Type":"User","Message":"`+errorType+`"}`)
	resp.Header.Set("X-Amzn-ErrorType", errorType)
	return resp
}

func TestLambdaGetPolicyOfLambdaWithoutOneIsEmpty(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return lambdaErrorResponse(404, "ResourceNotFoundException")
	})

	var policy, err = NewLambdaApi(testLogger()).GetPolicy(context.Background(), "arn:aws:lambda:us-east-1:1:function:orders")

	if err != nil || policy != "" {
		t.Errorf("policy = %q, %v, want empty without an error", policy, err)
	}
}

func TestLambdaGetPolicyReportsOtherErrors(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return lambdaErrorResponse(403, "AccessDeniedException")
	})

	var _, err = NewLambdaApi(testLogger()).GetPolicy(context.Background(), "arn:aws:lambda:us-east-1:1:function:orders")

	if err == nil || !strings.Contains(err.Error(), "AccessDeniedException") {
		t.Errorf("error = %v, want the service error", err)
	}
}

func TestLambdaListEventSourceMappingsRejectsMissingLambda(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/json", `{}`)
	})

	var _, err = NewLambdaApi(testLogger()).ListEventSourceMappings(context.Background(), "")

	if err == nil || err.Error() != "lambda ARN not set" || fake.calls() != 0 {
		t.Errorf("error = %v, calls = %d", err, fake.calls())
	}
}

func TestLambdaListVersionsGivesLatestThenNewestFirst(t *testing.T) {
	useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return response(200, "application/json",
				`{"Versions":[{"Version":"$LATEST"},{"Version":"1"},{"Version":"9"}],"NextMarker":"m"}`)
		}
		return response(200, "application/json", `{"Versions":[{"Version":"10"},{"Version":"2"}]}`)
	})

	var versions, err = NewLambdaApi(testLogger()).ListVersions(context.Background(), "arn:aws:lambda:us-east-1:1:function:orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []string
	for _, version := range versions {
		got = append(got, aws.ToString(version.Version))
	}
	if strings.Join(got, ",") != "$LATEST,10,9,2,1" {
		t.Errorf("versions = %v, want $LATEST then newest first", got)
	}
}

func TestLambdaListAliasesSortedByName(t *testing.T) {
	useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/json", `{"Aliases":[{"Name":"staging"},{"Name":"prod"}]}`)
	})

	var aliases, err = NewLambdaApi(testLogger()).ListAliases(context.Background(), "arn:aws:lambda:us-east-1:1:function:orders")

	if err != nil || len(aliases) != 2 || aws.ToString(aliases[0].Name) != "prod" {
		t.Errorf("aliases = %v, %v, want prod then staging", aliases, err)
	}
}

func TestLambdaVersionsAndAliasesRejectMissingLambda(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/json", `{}`)
	})
	var api = NewLambdaApi(testLogger())

	var _, versionsErr = api.ListVersions(context.Background(), "")
	var _, aliasesErr = api.ListAliases(context.Background(), "")

	for _, err := range []error{versionsErr, aliasesErr} {
		if err == nil || err.Error() != "lambda ARN not set" {
			t.Errorf("error = %v, want lambda ARN not set", err)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("calls = %d, want none", fake.calls())
	}
}
