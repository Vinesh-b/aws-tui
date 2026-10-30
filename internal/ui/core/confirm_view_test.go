package core

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func pressEnterOn(button *Button) {
	button.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
}

func TestConfirmView__ButtonsCallTheirHandlers(t *testing.T) {
	var view = NewConfirmView(newTestAppContext(), "Delete")
	var confirmed, cancelled int
	view.SetOnConfirmFunc(func() { confirmed++ })
	view.SetOnCancelFunc(func() { cancelled++ })

	pressEnterOn(view.confirmButton)
	pressEnterOn(view.cancelButton)
	pressEnterOn(view.cancelButton)

	if confirmed != 1 || cancelled != 2 {
		t.Errorf("confirmed, cancelled = %d, %d, want 1, 2", confirmed, cancelled)
	}
}

func TestConfirmView__FocusStartsOnCancelEachTimeItIsShown(t *testing.T) {
	var view = NewConfirmView(newTestAppContext(), "Delete")
	if view.GetLastFocusedView() != view.cancelButton {
		t.Fatal("expected the cancel button to be focused first")
	}

	view.navigator.NavigateWithin(APP_KEY_BINDINGS.ViewFocusRight)
	if view.GetLastFocusedView() != view.confirmButton {
		t.Fatal("expected the confirm button after moving focus")
	}
	view.OnShow()

	if view.GetLastFocusedView() != view.cancelButton {
		t.Error("expected the cancel button to be focused again when shown")
	}
}

func TestConfirmView__OnShowFillsInTheMessage(t *testing.T) {
	var view = NewConfirmView(newTestAppContext(), "Delete")
	view.SetOnShowFunc(func() { view.SetMessage("Delete the thing?") })

	view.OnShow()

	if got := view.GetMessage(); got != "Delete the thing?" {
		t.Errorf("message = %q", got)
	}
}

func TestFloatingConfirmView__OnShowIsForwarded(t *testing.T) {
	var floating = NewFloatingConfirmView(newTestAppContext(), "Confirm", "OK")
	var shown bool
	floating.Input.SetOnShowFunc(func() { shown = true })

	floating.OnShow()

	if !shown {
		t.Error("expected the show function to run")
	}
}
