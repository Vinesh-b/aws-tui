package app

import (
	"context"
	"testing"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/logging"
)

func TestCreateApplication__BuildsRootViewWithoutAwsProfiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	var tuiApp = CreateApplication(context.Background(), aws.Config{Region: "us-east-1"}, "test-version")

	if tuiApp == nil {
		t.Fatal("application was not created")
	}
}

func TestCreateApplication__ApiClientsLogToTheInAppLogger(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	CreateApplication(context.Background(), aws.Config{Region: "us-east-1"}, "test-version")

	var logger = awsapi.GetAwsApiClients().Config.Logger
	if _, ok := logger.(logging.StandardLogger); !ok {
		t.Errorf("api client logger = %T, want the in-app logger so SDK output stays off the terminal", logger)
	}
}
