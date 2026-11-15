package core

import (
	"slices"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/safetest"

	"github.com/rivo/tview"
)

// fakeTimers stands in for time.AfterFunc, so that a test decides when each
// timer fires. Timers are started on the UI goroutine and fired by the test.
type fakeTimers struct {
	mtx    sync.Mutex
	timers []*fakeTimer
}

type fakeTimer struct {
	fire    func()
	stopped bool
}

func (inst *fakeTimers) afterFunc(_ time.Duration, f func()) func() bool {
	inst.mtx.Lock()
	defer inst.mtx.Unlock()
	var timer = &fakeTimer{fire: f}
	inst.timers = append(inst.timers, timer)
	return func() bool {
		inst.mtx.Lock()
		defer inst.mtx.Unlock()
		var wasRunning = !timer.stopped
		timer.stopped = true
		return wasRunning
	}
}

func (inst *fakeTimers) count() int {
	inst.mtx.Lock()
	defer inst.mtx.Unlock()
	return len(inst.timers)
}

// fireLatest fires the last timer started, as when its delay passes
func (inst *fakeTimers) fireLatest(t *testing.T) {
	t.Helper()
	inst.mtx.Lock()
	if len(inst.timers) == 0 {
		inst.mtx.Unlock()
		t.Fatal("no timer was started")
	}
	var timer = inst.timers[len(inst.timers)-1]
	timer.stopped = true
	inst.mtx.Unlock()
	timer.fire()
}

// fireStale fires the first timer even though it was stopped, as a timer
// does whose callback had already started when it was stopped
func (inst *fakeTimers) fireStale(t *testing.T) {
	t.Helper()
	inst.mtx.Lock()
	var timer = inst.timers[0]
	inst.mtx.Unlock()
	timer.fire()
}

func newDebouncerTest(t *testing.T) (*Debouncer, *fakeTimers, func(func())) {
	t.Helper()
	var appCtx, _, _ = newLoaderTest(t)
	var timers = &fakeTimers{}
	var debouncer = newDebouncer(appCtx.App, time.Second, timers.afterFunc)
	return debouncer, timers, onUIOf(t, appCtx.App)
}

// onUIOf runs a func on the UI goroutine of the app and waits for it
func onUIOf(t *testing.T, app *tview.Application) func(func()) {
	return func(fn func()) {
		t.Helper()
		var done = make(chan struct{})
		app.QueueUpdate(func() {
			fn()
			close(done)
		})
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for the UI goroutine")
		}
	}
}

func TestDebouncer__FirstCallRunsStraightAway(t *testing.T) {
	var debouncer, _, onUI = newDebouncerTest(t)
	var calls safetest.Recorder[string]

	onUI(func() { debouncer.Run(func() { calls.Record("orders") }) })

	if got := calls.Calls(); !slices.Equal(got, []string{"orders"}) {
		t.Errorf("calls = %v, want [orders]", got)
	}
}

func TestDebouncer__BurstRunsOnlyTheLatestCallOnceTheCallsStop(t *testing.T) {
	var debouncer, timers, onUI = newDebouncerTest(t)
	var calls safetest.Recorder[string]
	onUI(func() {
		for _, name := range []string{"orders", "payments", "invoices", "refunds"} {
			debouncer.Run(func() { calls.Record(name) })
		}
	})
	if got := calls.Calls(); !slices.Equal(got, []string{"orders"}) {
		t.Fatalf("calls before the delay = %v, want [orders]", got)
	}

	timers.fireLatest(t)
	// The fired timer runs its call on the UI goroutine, ahead of this
	onUI(func() {})

	if got := calls.Calls(); !slices.Equal(got, []string{"orders", "refunds"}) {
		t.Errorf("calls = %v, want [orders refunds]", got)
	}
}

func TestDebouncer__SingleCallIsNotRunAgainWhenTheDelayPasses(t *testing.T) {
	var debouncer, timers, onUI = newDebouncerTest(t)
	var calls safetest.Recorder[string]
	onUI(func() { debouncer.Run(func() { calls.Record("orders") }) })

	timers.fireLatest(t)
	onUI(func() {})

	if got := calls.Count(); got != 1 {
		t.Errorf("ran %d times, want 1", got)
	}
}

func TestDebouncer__CallAfterTheBurstEndsRunsStraightAway(t *testing.T) {
	var debouncer, timers, onUI = newDebouncerTest(t)
	var calls safetest.Recorder[string]
	onUI(func() {
		debouncer.Run(func() { calls.Record("orders") })
		debouncer.Run(func() { calls.Record("payments") })
	})
	timers.fireLatest(t)
	onUI(func() {})

	onUI(func() { debouncer.Run(func() { calls.Record("invoices") }) })

	if got := calls.Calls(); !slices.Equal(got, []string{"orders", "payments", "invoices"}) {
		t.Errorf("calls = %v, want [orders payments invoices]", got)
	}
}

func TestDebouncer__ReplacedTimerThatStillFiresDoesNothing(t *testing.T) {
	var debouncer, timers, onUI = newDebouncerTest(t)
	var calls safetest.Recorder[string]
	onUI(func() {
		debouncer.Run(func() { calls.Record("orders") })
		debouncer.Run(func() { calls.Record("payments") })
	})
	if got := timers.count(); got != 2 {
		t.Fatalf("started %d timers, want 2", got)
	}

	timers.fireStale(t)
	onUI(func() {})

	if got := calls.Calls(); !slices.Equal(got, []string{"orders"}) {
		t.Errorf("calls = %v, want [orders]", got)
	}
}

func TestDebouncer__CancelDropsTheWaitingCall(t *testing.T) {
	var debouncer, timers, onUI = newDebouncerTest(t)
	var calls safetest.Recorder[string]
	onUI(func() {
		debouncer.Run(func() { calls.Record("orders") })
		debouncer.Run(func() { calls.Record("payments") })
		debouncer.Cancel()
	})

	timers.fireLatest(t)
	onUI(func() {})

	if got := calls.Calls(); !slices.Equal(got, []string{"orders"}) {
		t.Errorf("calls = %v, want [orders]", got)
	}
}
