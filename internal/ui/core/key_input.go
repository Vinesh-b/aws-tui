package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NormalizeKeyEvent gives the event of a key the way the bindings name it,
// whatever the keyboard protocol of the terminal. Terminals with the newer
// protocols send Ctrl+Space as a space with Ctrl, and the others as
// KeyCtrlSpace.
func NormalizeKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune && event.Rune() == ' ' && event.Modifiers()&tcell.ModCtrl != 0 {
		return tcell.NewEventKey(tcell.KeyCtrlSpace, 0, event.Modifiers())
	}
	return event
}

// paneKey is the key of the event for moving between panes. Terminals without
// the newer keyboard protocols send Ctrl+H as Backspace, so Backspace moves
// left too while Ctrl+H does, except in the views text is typed into.
func paneKey(event *tcell.EventKey, keyLeft tcell.Key, focused tview.Primitive) tcell.Key {
	if event.Key() == tcell.KeyBackspace && keyLeft == tcell.KeyCtrlH && !isTextEntry(focused) {
		return tcell.KeyCtrlH
	}
	return event.Key()
}
