package cloudwatch

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"aws-tui/internal/settings"
	"aws-tui/internal/ui/core"
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

func newViewWithQueriesFile(t *testing.T) (*InsightsQueryInputView, string) {
	var appCtx = uitest.NewAppContext()
	appCtx.QueriesPath = filepath.Join(t.TempDir(), "queries.json")
	return NewInsightsQueryInputView(appCtx), appCtx.QueriesPath
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

func TestInsightsQueryView__SavedQueryIsKeptByNameInOrderAndReplacedBySameName(t *testing.T) {
	var view, path = newViewWithQueriesFile(t)

	var save = func(name string, query string) {
		view.nameInput.SetText(name)
		view.queryTextArea.SetText(query, false)
		view.SaveQuery()
	}
	save("slow", "fields @duration")
	save("errors", "fields @message")
	save("slow", "fields @duration | sort @duration desc")

	uitest.AssertTableRows(t, view.savedTable, []core.TableRow{{"Name"}, {"errors"}, {"slow"}})
	var kept, err = settings.LoadQueries(path)
	if err != nil {
		t.Fatal(err)
	}
	var want = []settings.SavedQuery{
		{Name: "errors", Query: "fields @message"},
		{Name: "slow", Query: "fields @duration | sort @duration desc"},
	}
	if !slices.Equal(kept.Saved, want) {
		t.Errorf("kept = %+v, want %+v", kept.Saved, want)
	}
}

func TestInsightsQueryView__SavingNeedsANameAndAQuery(t *testing.T) {
	var view, path = newViewWithQueriesFile(t)
	var messages []string
	view.ErrorMessageCallback = func(text string, a ...any) { messages = append(messages, text) }

	view.nameInput.SetText("  ")
	view.SaveQuery()
	view.nameInput.SetText("errors")
	view.queryTextArea.SetText(" ", false)
	view.SaveQuery()

	if want := []string{"Enter a name to save the query as", "The query is empty"}; !slices.Equal(messages, want) {
		t.Errorf("messages = %v, want %v", messages, want)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("a file was written for queries that were not saved")
	}
}

func TestInsightsQueryView__SavedQueriesAndTheLastQueryComeBackNextTime(t *testing.T) {
	var view, path = newViewWithQueriesFile(t)
	view.nameInput.SetText("errors")
	view.queryTextArea.SetText("fields @message", false)
	view.SaveQuery()
	view.RememberQuery(InsightsQuery{Query: "stats count(*) by bin(5m)"})

	var appCtx = uitest.NewAppContext()
	appCtx.QueriesPath = path
	var next = NewInsightsQueryInputView(appCtx)

	uitest.AssertTableRows(t, next.savedTable, []core.TableRow{{"Name"}, {"errors"}})
	if got := next.queryTextArea.GetText(); got != "stats count(*) by bin(5m)" {
		t.Errorf("query = %q, want the one run last", got)
	}
}

func TestInsightsQueryView__SelectingASavedQueryPutsItInTheForm(t *testing.T) {
	var view, _ = newViewWithQueriesFile(t)
	view.nameInput.SetText("errors")
	view.queryTextArea.SetText("fields @message", false)
	view.SaveQuery()
	view.nameInput.SetText("")
	view.queryTextArea.SetText("something else", false)

	uitest.PressEnter(view.savedTable)

	if got := view.queryTextArea.GetText(); got != "fields @message" {
		t.Errorf("query = %q, want the saved one", got)
	}
	if got := view.nameInput.GetText(); got != "errors" {
		t.Errorf("name = %q, want the saved name so it can be replaced", got)
	}
}

func TestInsightsQueryView__DeletingASavedQueryForgetsIt(t *testing.T) {
	var view, path = newViewWithQueriesFile(t)
	for _, name := range []string{"a", "b"} {
		view.nameInput.SetText(name)
		view.queryTextArea.SetText("fields "+name, false)
		view.SaveQuery()
	}
	view.savedTable.Select(1, 0)

	uitest.PressTableRune(view.savedTable, core.APP_KEY_BINDINGS.RemoveItem)

	uitest.AssertTableRows(t, view.savedTable, []core.TableRow{{"Name"}, {"b"}})
	var kept, _ = settings.LoadQueries(path)
	if !slices.Equal(kept.Saved, []settings.SavedQuery{{Name: "b", Query: "fields b"}}) {
		t.Errorf("kept = %+v", kept.Saved)
	}
}

func TestInsightsQueryView__WithoutAFileQueriesAreKeptOnlyForTheSession(t *testing.T) {
	var view = NewInsightsQueryInputView(uitest.NewAppContext())
	view.nameInput.SetText("errors")
	view.queryTextArea.SetText("fields @message", false)

	view.SaveQuery()

	uitest.AssertTableRows(t, view.savedTable, []core.TableRow{{"Name"}, {"errors"}})
}
