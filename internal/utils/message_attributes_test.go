package utils

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMessageAttributes__ReadsNamesTypesAndValuesInOrder(t *testing.T) {
	var text = "Source=tui\n\n  Count:Number = 3.5 \nPriority:Number.int=2\nBlob:Binary=aGVsbG8=\nLabel:String.custom=a=b\n"

	var got, err = ParseMessageAttributes(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want = []MessageAttribute{
		{Name: "Source", DataType: "String", StringValue: "tui"},
		{Name: "Count", DataType: "Number", StringValue: "3.5"},
		{Name: "Priority", DataType: "Number.int", StringValue: "2"},
		{Name: "Blob", DataType: "Binary", BinaryValue: []byte("hello")},
		{Name: "Label", DataType: "String.custom", StringValue: "a=b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseMessageAttributes__EmptyTextHasNoAttributes(t *testing.T) {
	var got, err = ParseMessageAttributes(" \n\n")

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
			var _, err = ParseMessageAttributes(tc.text)

			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestParseMessageAttributes__ErrorNamesTheLine(t *testing.T) {
	var _, err = ParseMessageAttributes("a=1\n\nb:Number=x")

	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("error = %v, want one naming line 3", err)
	}
}
