package core

import (
	"slices"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SequenceTimeout is how long the next key of a sequence is waited for, like
// the timeoutlen of Vim
const SequenceTimeout = time.Second

// keySequence is the keys of a sequence binding and the key the views know
// that it does
type keySequence struct {
	keys []rune
	does *tcell.EventKey
}

// keySequences are the sequences as they are bound now
func keySequences() []keySequence {
	var keys = APP_KEY_BINDINGS
	return []keySequence{
		{[]rune(keys.MoveTopSequence), tcell.NewEventKey(tcell.KeyRune, keys.MovePageTopRune, tcell.ModNone)},
		{[]rune(keys.PageForwardSequence), tcell.NewEventKey(tcell.KeyRune, keys.PageForward, keys.PageChangeModKey)},
		{[]rune(keys.PageBackSequence), tcell.NewEventKey(tcell.KeyRune, keys.PageBack, keys.PageChangeModKey)},
		{[]rune(keys.OpenLinkSequence), tcell.NewEventKey(tcell.KeyRune, keys.OpenLink, tcell.ModNone)},
	}
}

// KeySequencer turns the key sequences of the bindings, such as gg, into the
// keys the views handle, so the views need not know about sequences. The keys
// of a sequence are held back until it is complete. A key that ends no
// sequence drops the keys before it, as in Vim, and is then taken on its own.
// Text typed into an input is never held back. It is only used on the UI
// goroutine.
type KeySequencer struct {
	pending   []rune
	pendingAt time.Time
	now       func() time.Time
}

func NewKeySequencer() *KeySequencer {
	return &KeySequencer{now: time.Now}
}

// HandleEvent returns the event to pass on to the focused view, which is nil
// while a sequence is being typed
func (inst *KeySequencer) HandleEvent(event *tcell.EventKey, focused tview.Primitive) *tcell.EventKey {
	var now = inst.now()
	if len(inst.pending) > 0 && now.Sub(inst.pendingAt) > SequenceTimeout {
		inst.pending = nil
	}

	if isTextEntry(focused) || event.Key() != tcell.KeyRune || event.Modifiers() != tcell.ModNone {
		// Keys such as Esc stop the sequence and still do what they do
		inst.pending = nil
		return event
	}

	var typed = append(slices.Clone(inst.pending), event.Rune())
	var started = false
	for _, sequence := range keySequences() {
		if len(sequence.keys) == 0 {
			continue
		}
		if slices.Equal(sequence.keys, typed) {
			inst.pending = nil
			return sequence.does
		}
		if len(sequence.keys) > len(typed) && slices.Equal(sequence.keys[:len(typed)], typed) {
			started = true
		}
	}

	if started {
		inst.pending = typed
		inst.pendingAt = now
		return nil
	}
	if len(inst.pending) > 0 {
		inst.pending = nil
		return inst.HandleEvent(event, focused)
	}
	return event
}
