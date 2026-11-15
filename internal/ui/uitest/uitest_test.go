package uitest

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newKeyCountingApp runs an app whose only view counts the keys it is given
func newKeyCountingApp(t *testing.T) (*tview.Application, *atomic.Int32) {
	t.Helper()
	var app = tview.NewApplication()
	var handled = new(atomic.Int32)
	var box = tview.NewBox()
	box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		handled.Add(1)
		return nil
	})
	RunApp(t, app)
	OnUI(t, app, func() {
		app.SetRoot(box, true)
		app.SetFocus(box)
	})
	return app, handled
}

// The app takes whichever of a waiting event and a waiting update it likes. So
// when it is busy while a key is pressed, as it often is with pages loading,
// waiting for an update queued after the key is not enough.
func TestPressKey__ReturnsOnceTheKeyIsHandled(t *testing.T) {
	RunInBubble(t, func(t *testing.T) {
		var app = tview.NewApplication()
		// returned is closed once the press being made returns, and
		// handledAfterReturn counts the keys handled after their press
		// returned. Both are only used on the UI goroutine.
		var returned chan struct{}
		var handledAfterReturn int
		var box = tview.NewBox()
		box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			select {
			case <-returned:
				handledAfterReturn++
			case <-time.After(time.Second):
				// The press is still waiting for its key, as it should be.
				// Shorter than the time the press waits.
			}
			return nil
		})
		RunApp(t, app)
		OnUI(t, app, func() {
			app.SetRoot(box, true)
			app.SetFocus(box)
		})

		for press := 1; press <= 50; press++ {
			var pressReturned = make(chan struct{})
			OnUI(t, app, func() { returned = pressReturned })
			// The app is kept busy until the press has queued all it queues,
			// so that it then has both an event and an update waiting
			var release = make(chan struct{})
			go app.QueueUpdate(func() { <-release })
			synctest.Wait()
			go func() {
				PressKey(t, app, tcell.KeyTab)
				close(pressReturned)
			}()
			synctest.Wait()
			close(release)
			<-pressReturned
			synctest.Wait()

			var late int
			OnUI(t, app, func() { late = handledAfterReturn })
			if late != 0 {
				t.Fatalf("press %d returned before its key was handled", press)
			}
		}
	})
}

func TestPressKeyEvent__KeepsTheInputCaptureOfTheApp(t *testing.T) {
	var app, handled = newKeyCountingApp(t)
	var captured []rune
	OnUI(t, app, func() {
		app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			captured = append(captured, event.Rune())
			return event
		})
	})

	PressKeyEvent(t, app, tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	PressKeyEvent(t, app, tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone))

	var got string
	OnUI(t, app, func() { got = string(captured) })
	if got != "ab" || handled.Load() != 2 {
		t.Errorf("captured %q and handled %d keys, want ab and 2", got, handled.Load())
	}
}
