package sqs

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestParseMessageAttributes__ReadsNamesTypesAndValues(t *testing.T) {
	var text = "Source=tui\n\n  Count:Number = 3.5 \nPriority:Number.int=2\nBlob:Binary=aGVsbG8=\nLabel:String.custom=a=b\n"

	var got, err = parseMessageAttributes(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("got %d attributes, want 5: %v", len(got), got)
	}
	if v := got["Source"]; aws.ToString(v.DataType) != "String" || aws.ToString(v.StringValue) != "tui" {
		t.Errorf("Source = %v", v)
	}
	if v := got["Count"]; aws.ToString(v.DataType) != "Number" || aws.ToString(v.StringValue) != "3.5" {
		t.Errorf("Count = %v", v)
	}
	if v := got["Priority"]; aws.ToString(v.DataType) != "Number.int" || aws.ToString(v.StringValue) != "2" {
		t.Errorf("Priority = %v", v)
	}
	if v := got["Blob"]; aws.ToString(v.DataType) != "Binary" || string(v.BinaryValue) != "hello" || v.StringValue != nil {
		t.Errorf("Blob = %v", v)
	}
	if v := got["Label"]; aws.ToString(v.DataType) != "String.custom" || aws.ToString(v.StringValue) != "a=b" {
		t.Errorf("Label = %v", v)
	}
}

func TestParseMessageAttributes__EmptyTextHasNoAttributes(t *testing.T) {
	var got, err = parseMessageAttributes(" \n\n")

	if err != nil || got != nil {
		t.Errorf("got %v, %v, want nil and no error", got, err)
	}
}

func TestParseMessageAttributes__RejectsInvalidLines(t *testing.T) {
	var tooMany = strings.Repeat("a=1\n", 1) + "b=1\nc=1\nd=1\ne=1\nf=1\ng=1\nh=1\ni=1\nj=1\nk=1"

	for name, tc := range map[string]struct{ text, want string }{
		"no equals":       {"Source", "expected Name=Value"},
		"empty name":      {"=x", "invalid name"},
		"bad name":        {"my attr=x", "invalid name"},
		"reserved prefix": {"AWS.thing=x", "can't start with"},
		"empty string":    {"a=", "is empty"},
		"bad number":      {"a:Number=abc", "not a number"},
		"infinite number": {"a:Number=Inf", "not a number"},
		"bad base64":      {"a:Binary=***", "not base64"},
		"unknown type":    {"a:Bool=true", "unknown type"},
		"duplicate":       {"a=1\na=2", "more than once"},
		"too many":        {tooMany, "at most 10"},
	} {
		t.Run(name, func(t *testing.T) {
			var _, err = parseMessageAttributes(tc.text)

			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestParseMessageAttributes__ErrorNamesTheLine(t *testing.T) {
	var _, err = parseMessageAttributes("a=1\n\nb:Number=x")

	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("error = %v, want one naming line 3", err)
	}
}
