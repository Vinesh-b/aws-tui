package cloudwatch

import (
	"testing"
	"time"

	"aws-tui/internal/ui/uitest"
)

func TestInsightsQueryView__DefaultQuery(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())

	var query, err = view.GenerateQuery()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var want = "fields @timestamp, @message, @log\n| sort @timestamp desc\n| limit 1000"
	if query.Query != want {
		t.Errorf("query = %q, want %q", query.Query, want)
	}
}

func TestInsightsQueryView__DefaultTimesSpanTheLastThreeHours(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())

	var query, err = view.GenerateQuery()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var span = query.EndTime.Sub(query.StartTime)
	if span < 3*time.Hour-time.Minute || span > 3*time.Hour+time.Minute {
		t.Errorf("span = %v, want about 3h", span)
	}
}

func TestInsightsQueryView__UsesEnteredQueryAndTimes(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())
	view.queryTextArea.SetText("  fields @message  \n", false)
	view.startDateInput.SetTextTime(baseTime)
	view.endDateInput.SetTextTime(baseTime.Add(time.Hour))

	var query, err = view.GenerateQuery()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Query != "fields @message" {
		t.Errorf("query = %q, want %q", query.Query, "fields @message")
	}
	if !query.StartTime.Equal(baseTime.Truncate(time.Second)) ||
		!query.EndTime.Equal(baseTime.Add(time.Hour).Truncate(time.Second)) {
		t.Errorf("times = %v - %v", query.StartTime, query.EndTime)
	}
}

func TestInsightsQueryView__InvalidTimesAreRejected(t *testing.T) {
	var tests = []struct {
		name  string
		setup func(view *InsightsQueryInputView)
	}{
		{"start time", func(view *InsightsQueryInputView) { view.startDateInput.SetText("yesterday") }},
		{"end time", func(view *InsightsQueryInputView) { view.endDateInput.SetText("tomorrow") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var view = NewInsightsQueryInputView(uitest.NewAppContext())
			tt.setup(view)

			var query, err = view.GenerateQuery()

			if err == nil {
				t.Fatal("expected an error")
			}
			if query != (InsightsQuery{}) {
				t.Errorf("query = %+v, want empty", query)
			}
		})
	}
}

func TestInsightsQueryView__EmptyQueryIsRejected(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())
	view.queryTextArea.SetText(" \n ", false)

	var query, err = view.GenerateQuery()

	if err == nil || err.Error() != "The query is empty" {
		t.Errorf("error = %v, want the query is empty", err)
	}
	if query != (InsightsQuery{}) {
		t.Errorf("query = %+v, want empty", query)
	}
}

func TestInsightsQueryView__EndBeforeStartIsRejected(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())
	view.startDateInput.SetTextTime(baseTime.Add(time.Hour))
	view.endDateInput.SetTextTime(baseTime)

	var query, err = view.GenerateQuery()

	if err == nil || err.Error() != "The end time is before the start time" {
		t.Errorf("error = %v, want the end time is before the start time", err)
	}
	if query != (InsightsQuery{}) {
		t.Errorf("query = %+v, want empty", query)
	}
}
