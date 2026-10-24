package sfn

import (
	"strings"
	"testing"
	"time"

	"aws-tui/internal/ui/uitest"
)

func TestExecutionsQueryView__GeneratesQueryFromInputs(t *testing.T) {
	var view = NewSfnExecutionsQueryInputView(uitest.NewAppContext())
	view.SetDefaultTimes(baseTime, baseTime.Add(time.Hour))
	view.executionArnInput.SetText("  run-1 ")

	var query, err = view.GenerateQuery()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.executionArn != "run-1" {
		t.Errorf("execution arn = %q, want %q", query.executionArn, "run-1")
	}
	if !query.startTime.Equal(baseTime) || !query.endTime.Equal(baseTime.Add(time.Hour)) {
		t.Errorf("times = %v - %v, want %v - %v", query.startTime, query.endTime, baseTime, baseTime.Add(time.Hour))
	}
}

func TestExecutionsQueryView__InvalidTimesAreRejected(t *testing.T) {
	var tests = []struct {
		name  string
		setup func(view *SfnExecutionsQueryInputView)
	}{
		{"start time", func(view *SfnExecutionsQueryInputView) { view.startDateInput.SetText("yesterday") }},
		{"end time", func(view *SfnExecutionsQueryInputView) { view.endDateInput.SetText("tomorrow") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var view = NewSfnExecutionsQueryInputView(uitest.NewAppContext())
			tt.setup(view)

			var query, err = view.GenerateQuery()

			if err == nil {
				t.Fatal("expected an error")
			}
			if query != (SfnExecutionsQuery{}) {
				t.Errorf("query = %+v, want empty", query)
			}
		})
	}
}

func TestExecutionsQueryView__DefaultTimesSpanTheLastThreeHours(t *testing.T) {
	var view = NewSfnExecutionsQueryInputView(uitest.NewAppContext())

	var query, err = view.GenerateQuery()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if span := query.endTime.Sub(query.startTime); span < 3*time.Hour-time.Minute || span > 3*time.Hour+time.Minute {
		t.Errorf("span = %v, want about 3h", span)
	}
	if strings.TrimSpace(query.executionArn) != "" {
		t.Errorf("execution arn = %q, want empty", query.executionArn)
	}
}
