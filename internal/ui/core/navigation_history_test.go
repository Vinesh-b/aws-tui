package core

import (
	"testing"
)

const historyLimit = 100

var (
	lambdaList   = Location{Service: "Lambda", Page: "Functions"}
	logsGroups   = Location{Service: "Logs", Page: "Groups"}
	logsStreams  = Location{Service: "Logs", Page: "Streams"}
	settingsPage = Location{Service: "Settings"}
)

func visitAll(history *NavigationHistory, locations ...Location) {
	for _, location := range locations {
		history.Visit(location)
	}
}

func assertCurrent(t *testing.T, history *NavigationHistory, want Location) {
	t.Helper()
	if got, ok := history.Current(); !ok || got != want {
		t.Errorf("current = %+v (%t), want %+v", got, ok, want)
	}
}

func TestNavigationHistory__StartsWithNothingToGoBackOrForwardTo(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)

	if _, ok := history.Current(); ok {
		t.Error("a new history has a current location")
	}
	if _, ok := history.Back(); ok || history.CanGoBack() {
		t.Error("a new history goes back")
	}
	if _, ok := history.Forward(); ok || history.CanGoForward() {
		t.Error("a new history goes forward")
	}
}

func TestNavigationHistory__BackGoesThroughTheVisitsInReverse(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups, logsStreams)

	for _, want := range []Location{logsGroups, lambdaList} {
		if got, ok := history.Back(); !ok || got != want {
			t.Fatalf("back = %+v (%t), want %+v", got, ok, want)
		}
	}
	if _, ok := history.Back(); ok {
		t.Error("went back past the first visit")
	}
	assertCurrent(t, history, lambdaList)
}

func TestNavigationHistory__ForwardGoesBackToWhereBackCameFrom(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups, logsStreams)
	history.Back()
	history.Back()

	for _, want := range []Location{logsGroups, logsStreams} {
		if got, ok := history.Forward(); !ok || got != want {
			t.Fatalf("forward = %+v (%t), want %+v", got, ok, want)
		}
	}
	if history.CanGoForward() {
		t.Error("can go forward past the last visit")
	}
}

func TestNavigationHistory__VisitingAfterGoingBackDropsTheForwardLocations(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups, logsStreams)
	history.Back()

	history.Visit(settingsPage)

	if history.CanGoForward() {
		t.Error("the locations gone back from were kept")
	}
	if got, _ := history.Back(); got != logsGroups {
		t.Errorf("back = %+v, want %+v", got, logsGroups)
	}
}

func TestNavigationHistory__VisitingTheCurrentLocationAgainChangesNothing(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups, logsGroups)

	if got, _ := history.Back(); got != lambdaList {
		t.Errorf("back = %+v, want %+v", got, lambdaList)
	}
	if history.CanGoBack() {
		t.Error("the repeated visit was kept")
	}
}

func TestNavigationHistory__KeepsOnlyTheNewestLocationsToGoBackTo(t *testing.T) {
	var history = NewNavigationHistory(2)
	visitAll(history, lambdaList, logsGroups, logsStreams, settingsPage)

	var got []Location
	for history.CanGoBack() {
		var location, _ = history.Back()
		got = append(got, location)
	}

	if len(got) != 2 || got[0] != logsStreams || got[1] != logsGroups {
		t.Errorf("back went to %+v, want the newest two", got)
	}
}

func TestNavigationHistory__VisitsWithoutRecordingAreNotKept(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups)
	var back, _ = history.Back()

	history.WithoutRecording(func() {
		history.Visit(logsStreams)
		history.Visit(back)
	})

	assertCurrent(t, history, lambdaList)
	if got, _ := history.Forward(); got != logsGroups {
		t.Errorf("forward = %+v, want %+v", got, logsGroups)
	}

	history.Visit(settingsPage)
	if got, _ := history.Back(); got != logsGroups {
		t.Errorf("visits after WithoutRecording are not kept, back = %+v", got)
	}
}

func TestNavigationHistory__RestartKeepsOnlyTheLocationGiven(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups, logsStreams)
	history.Back()

	history.Restart(settingsPage)

	assertCurrent(t, history, settingsPage)
	if history.CanGoBack() || history.CanGoForward() {
		t.Error("restart kept locations to go back or forward to")
	}
}

func TestNavigationHistory__ClearDropsTheCurrentLocation(t *testing.T) {
	var history = NewNavigationHistory(historyLimit)
	visitAll(history, lambdaList, logsGroups)

	history.Clear()

	if got, ok := history.Current(); ok {
		t.Errorf("current = %+v, want none", got)
	}
	if history.CanGoBack() {
		t.Error("clear kept locations to go back to")
	}
	history.Visit(logsStreams)
	if history.CanGoBack() {
		t.Error("the first visit after clear has a location to go back to")
	}
}
