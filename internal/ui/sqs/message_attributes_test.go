package sqs

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestParseMessageAttributes__BuildsSqsAttributeValues(t *testing.T) {
	var got, err = parseMessageAttributes("Source=tui\nCount:Number=3\nBlob:Binary=aGVsbG8=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("got %d attributes, want 3: %v", len(got), got)
	}
	if v := got["Source"]; aws.ToString(v.DataType) != "String" || aws.ToString(v.StringValue) != "tui" || v.BinaryValue != nil {
		t.Errorf("Source = %v", v)
	}
	if v := got["Count"]; aws.ToString(v.DataType) != "Number" || aws.ToString(v.StringValue) != "3" {
		t.Errorf("Count = %v", v)
	}
	if v := got["Blob"]; aws.ToString(v.DataType) != "Binary" || string(v.BinaryValue) != "hello" || v.StringValue != nil {
		t.Errorf("Blob = %v", v)
	}
}

func TestParseMessageAttributes__NoAttributesIsNil(t *testing.T) {
	if got, err := parseMessageAttributes(""); got != nil || err != nil {
		t.Errorf("got %v, %v, want nil and no error", got, err)
	}
}

func TestParseMessageAttributes__ReportsInvalidAttributes(t *testing.T) {
	if _, err := parseMessageAttributes("Count:Number=abc"); err == nil {
		t.Error("expected an error for a bad number")
	}
}
