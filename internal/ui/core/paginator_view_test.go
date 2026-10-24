package core

import (
	"runtime"
	"testing"
	"time"
)

func TestPaginatorView__StopEndsThePollingGoroutine(t *testing.T) {
	var before = runtime.NumGoroutine()
	var view = CreatePaginatorView("service", newTestAppContext())
	if runtime.NumGoroutine() <= before {
		t.Fatal("expected the paginator to start a polling goroutine")
	}

	view.Stop()
	view.Stop()

	var deadline = time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > before {
		t.Errorf("goroutines = %d, want %d or fewer once stopped", got, before)
	}
}

func TestServiceRootView__CloseStopsThePaginatorMoreThanOnce(t *testing.T) {
	var view = NewServiceRootView("service", newTestAppContext())

	view.Close()
	view.Close()
}
