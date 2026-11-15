package core

import (
	"time"

	"github.com/rivo/tview"
)

// SelectionLoadDelay is how long the selection of a list has to stay on a row
// before the details of that row are loaded, while moving through the list
const SelectionLoadDelay = 200 * time.Millisecond

// Debouncer runs the last of a burst of calls once the calls stop, so that
// moving quickly through a list does not load every row passed on the way.
// The first call after a quiet spell is run straight away, so a single move
// shows its data with no delay.
//
// Run is called on the UI goroutine, and the calls it is given are run there
// too, so they can update views directly.
type Debouncer struct {
	app       *tview.Application
	delay     time.Duration
	afterFunc func(time.Duration, func()) (stop func() bool)

	// The state below is only used on the UI goroutine.

	// pending is the latest call that is waiting for the calls to stop
	pending func()
	// stopTimer stops the timer of the current burst, which is nil when no
	// burst is under way
	stopTimer func() bool
	// burst tells the timers apart, so that one which fires after it was
	// replaced does nothing
	burst uint64
}

func NewDebouncer(appCtx *AppContext, delay time.Duration) *Debouncer {
	return newDebouncer(appCtx.App, delay, func(d time.Duration, f func()) func() bool {
		return time.AfterFunc(d, f).Stop
	})
}

func newDebouncer(
	app *tview.Application,
	delay time.Duration,
	afterFunc func(time.Duration, func()) (stop func() bool),
) *Debouncer {
	return &Debouncer{app: app, delay: delay, afterFunc: afterFunc}
}

// Run runs call straight away if no other call was made within the delay.
// Otherwise it waits until no call has been made for the delay and then runs
// only the latest call. It must be called on the UI goroutine.
func (inst *Debouncer) Run(call func()) {
	if inst.stopTimer == nil {
		call()
	} else {
		inst.stopTimer()
		inst.pending = call
	}
	inst.startTimer()
}

// Cancel drops the call that is waiting, if there is one. It must be called on
// the UI goroutine.
func (inst *Debouncer) Cancel() {
	if inst.stopTimer != nil {
		inst.stopTimer()
	}
	inst.burst++
	inst.stopTimer = nil
	inst.pending = nil
}

func (inst *Debouncer) startTimer() {
	inst.burst++
	var burst = inst.burst
	inst.stopTimer = inst.afterFunc(inst.delay, func() {
		inst.app.QueueUpdateDraw(func() { inst.fire(burst) })
	})
}

// fire ends the burst when the delay passes without another call
func (inst *Debouncer) fire(burst uint64) {
	// Stopping a timer doesn't stop a callback that has already started, so a
	// timer that has since been replaced can still get here
	if burst != inst.burst {
		return
	}
	var call = inst.pending
	inst.pending = nil
	inst.stopTimer = nil
	if call != nil {
		call()
	}
}
