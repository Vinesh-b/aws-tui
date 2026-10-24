package app

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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
