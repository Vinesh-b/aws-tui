package awsapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// useReadOnlyFakeAws is a fake backend with read-only clients that reach it
func useReadOnlyFakeAws(t *testing.T, body string) *fakeHttp {
	t.Helper()
	var fake = &fakeHttp{handler: func(int, *http.Request, string) *http.Response {
		return response(200, "application/x-amz-json-1.1", body)
	}}
	fake.clients = NewAwsApiClients(WithReadOnly(aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("id", "secret", ""),
		HTTPClient:       fake,
		RetryMaxAttempts: 1,
	}), "test")
	return fake
}

func TestIsReadOperation(t *testing.T) {
	for operation, want := range map[string]bool{
		"ListTopics":          true,
		"GetParameter":        true,
		"DescribeInstances":   true,
		"HeadObject":          true,
		"Query":               true,
		"Scan":                true,
		"FilterLogEvents":     true,
		"StartQuery":          true,
		"StopQuery":           true,
		"GetCallerIdentity":   true,
		"PutParameter":        false,
		"DeleteTopic":         false,
		"Publish":             false,
		"Invoke":              false,
		"TestInvokeMethod":    false,
		"StartExecution":      false,
		"SendMessage":         false,
		"UpdateItem":          false,
		"CreateSchedule":      false,
		"Subscribe":           false,
		"StartQueryExecution": false,
		"":                    false,
	} {
		if got := IsReadOperation(operation); got != want {
			t.Errorf("IsReadOperation(%q) = %v, want %v", operation, got, want)
		}
	}
}

func TestReadOnly__ReadsAreSent(t *testing.T) {
	var fake = useReadOnlyFakeAws(t, `{"Parameter":{"Name":"/app/db/host","Value":"db.internal"}}`)

	var value, err = NewSystemsManagerApi(fake.clients, testLogger()).GetParameterValue(context.Background(), "/app/db/host", 0)

	if err != nil || value != "db.internal" || fake.calls() != 1 {
		t.Errorf("value = %q, err = %v, calls = %d, want the value from one call", value, err, fake.calls())
	}
}

func TestReadOnly__ChangesAreRefusedBeforeTheyAreSent(t *testing.T) {
	var fake = useReadOnlyFakeAws(t, `{}`)
	var ssmApi = NewSystemsManagerApi(fake.clients, testLogger())
	var changes = map[string]func() error{
		"PutParameter": func() error {
			return ssmApi.PutParameter(context.Background(), "/app/db/host", "db.external", ssmTypes.ParameterTypeString)
		},
		"DeleteParameter": func() error { return ssmApi.DeleteParameter(context.Background(), "/app/db/host") },
		"DeleteTopic": func() error {
			return NewSnsApi(fake.clients, testLogger()).DeleteTopic(context.Background(), "arn:aws:sns:us-east-1:1:orders")
		},
		"SendMessage": func() error {
			var _, err = NewSqsApi(fake.clients, testLogger()).SendMessage(
				context.Background(), "https://sqs.us-east-1.amazonaws.com/1/orders", "body", "", "", nil,
			)
			return err
		},
		"Invoke": func() error {
			var _, err = NewLambdaApi(fake.clients, testLogger()).InvokeLambda(context.Background(), "orders", map[string]any{})
			return err
		},
	}

	for operation, change := range changes {
		var err = change()

		var refused *ReadOnlyError
		if !errors.As(err, &refused) || refused.Operation != operation {
			t.Errorf("%s: err = %v, want it refused as read-only", operation, err)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("%d requests were sent, want none", fake.calls())
	}
}

func TestWithReadOnly__LeavesTheConfigItWasGivenAsItWas(t *testing.T) {
	var cfg = aws.Config{Region: "us-east-1"}

	var readOnly = WithReadOnly(cfg)

	if len(cfg.APIOptions) != 0 || len(readOnly.APIOptions) != 1 || readOnly.Region != "us-east-1" {
		t.Errorf("options = %d before and %d after, want 0 and 1 with the region kept", len(cfg.APIOptions), len(readOnly.APIOptions))
	}
}
