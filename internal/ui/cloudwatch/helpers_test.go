package cloudwatch

import (
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

var baseTime = time.Date(2024, 1, 2, 3, 4, 5, 678000000, time.UTC)

func TestMain(m *testing.M) {
	time.Local = time.UTC
	os.Exit(m.Run())
}

func millisAt(seconds int) *int64 {
	return aws.Int64(baseTime.Add(time.Duration(seconds) * time.Second).UnixMilli())
}
