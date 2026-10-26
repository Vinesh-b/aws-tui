package cloudwatch

import (
	"slices"
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

func TestInsightsQueryView__PickingARangeSetsTimesEndingNow(t *testing.T) {
	var tests = []struct {
		label string
		span  time.Duration
	}{
		{"Last 15 minutes", 15 * time.Minute},
		{"Last hour", time.Hour},
		{"Last 3 hours", 3 * time.Hour},
		{"Last 24 hours", 24 * time.Hour},
		{"Last 7 days", 7 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			var view = NewInsightsQueryInputView(uitest.NewAppContext())
			var index = slices.IndexFunc(insightsRanges, func(r insightsRange) bool { return r.label == tt.label })

			view.rangeDropDown.SetCurrentOption(index)
			var query, err = view.GenerateQuery()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if span := query.EndTime.Sub(query.StartTime); span != tt.span {
				t.Errorf("span = %v, want %v", span, tt.span)
			}
			if age := time.Since(query.EndTime); age < 0 || age > time.Minute {
				t.Errorf("the range ends %v ago, want now", age)
			}
		})
	}
}

func TestInsightsQueryView__RangeStartsOnThreeHours(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())

	if index := view.rangeIndex(); insightsRanges[index].label != "Last 3 hours" {
		t.Errorf("range = %q, want Last 3 hours", insightsRanges[index].label)
	}
}

func TestInsightsQueryView__EnteringATimeMovesToCustomAndKeepsIt(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())
	view.rangeDropDown.SetCurrentOption(1)

	view.startDateInput.SetTextTime(baseTime)
	view.endDateInput.SetTextTime(baseTime.Add(2 * time.Hour))
	var query, err = view.GenerateQuery()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if index := view.rangeIndex(); index != customInsightsRange {
		t.Errorf("range = %q, want Custom", insightsRanges[index].label)
	}
	if query.EndTime.Sub(query.StartTime) != 2*time.Hour || !query.StartTime.Equal(baseTime.Truncate(time.Second)) {
		t.Errorf("times = %v - %v, want those entered", query.StartTime, query.EndTime)
	}
}

func TestInsightsQueryView__PickingARangeIsNotEnteringATime(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())

	view.rangeDropDown.SetCurrentOption(0)
	view.rangeDropDown.SetCurrentOption(3)

	if index := view.rangeIndex(); insightsRanges[index].label != "Last 24 hours" {
		t.Errorf("range = %q, want the one picked", insightsRanges[index].label)
	}
}
