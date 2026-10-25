package core

import (
	"testing"
	"time"
)

func TestDateTimeInputField__ReadsBackTheInstantItWasGiven(t *testing.T) {
	var saved = time.Local
	time.Local = time.FixedZone("UTC+5", 5*60*60)
	defer func() { time.Local = saved }()

	var field = NewDateTimeInputField(&AppTheme{})
	var instant = time.Date(2026, 7, 1, 12, 30, 15, 0, time.UTC)

	field.SetTextTime(instant)
	var got, err = field.ValidateInput()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(instant) {
		t.Errorf("read %v, want the same instant as %v", got, instant)
	}
	if field.GetText() != "2026-07-01 17:30:15" {
		t.Errorf("text = %q, want the local wall clock time", field.GetText())
	}
}

func TestDateTimeInputField__RejectsTextThatIsNotATime(t *testing.T) {
	var field = NewDateTimeInputField(&AppTheme{})
	field.SetText("2026-13-45")

	if _, err := field.ValidateInput(); err == nil {
		t.Error("expected an error for an impossible date")
	}
}
