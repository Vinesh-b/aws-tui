package core

import (
	"testing"
	"time"
)

func TestFormatTime(t *testing.T) {
	var plusTwo = time.FixedZone("UTC+2", 2*60*60)
	var local = time.Date(2024, 1, 2, 5, 4, 5, 0, plusTwo)
	var tests = []struct {
		name string
		time *time.Time
		want string
	}{
		{"unset", nil, ""},
		{"zero", &time.Time{}, ""},
		{"converted to utc", &local, "2024-01-02 03:04:05"},
	}

	for _, tt := range tests {
		if got := FormatTime(tt.time); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestFormatUnixMilli(t *testing.T) {
	var ms = time.Date(2024, 1, 2, 3, 4, 5, 678e6, time.UTC).UnixMilli()

	if got := FormatUnixMilli(&ms); got != "2024-01-02 03:04:05" {
		t.Errorf("got %q", got)
	}
	if got := FormatUnixMilli(nil); got != "" {
		t.Errorf("unset = %q, want empty", got)
	}
}

func TestFormatUnixSeconds(t *testing.T) {
	var seconds = "1704164645"

	if got := FormatUnixSeconds(seconds); got != "2024-01-02 03:04:05" {
		t.Errorf("got %q", got)
	}
	for _, text := range []string{"", "soon"} {
		if got := FormatUnixSeconds(text); got != "" {
			t.Errorf("%q = %q, want empty", text, got)
		}
	}
}

type testArch string

func TestFormatList(t *testing.T) {
	if got := FormatList([]testArch{"x86_64", "arm64"}); got != "x86_64, arm64" {
		t.Errorf("got %q", got)
	}
	if got := FormatList([]string(nil)); got != "" {
		t.Errorf("empty = %q, want empty", got)
	}
}
