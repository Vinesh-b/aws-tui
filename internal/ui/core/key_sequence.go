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

// keySequence is the keys of a sequence binding and what it does on the
// focused view. That returns the event to pass on to the view, for sequences
// that do what a key of the views does, or nil.
type keySequence struct {
	keys []rune
	does func(focused tview.Primitive) *tcell.EventKey
}

// passRune passes the key of the views on, for a sequence that does what it does
func passRune(r rune) func(tview.Primitive) *tcell.EventKey {
	return func(tview.Primitive) *tcell.EventKey {
		return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
	}
}

// keySequences are the sequences as they are bound now
func (inst *KeySequencer) keySequences() []keySequence {
	var keys = APP_KEY_BINDINGS
	var movePage = func(offset int) func(tview.Primitive) *tcell.EventKey {
		return func(tview.Primitive) *tcell.EventKey {
			inst.movePage(offset)
			return nil
		}
	}
	return []keySequence{
		{[]rune(keys.MoveTopSequence), passRune(keys.MovePageTopRune)},
		{[]rune(keys.RemoveItemSequence), passRune(keys.RemoveItem)},
		{[]rune(keys.PageForwardSequence), movePage(1)},
		{[]rune(keys.PageBackSequence), movePage(-1)},
		{[]rune(keys.OpenLinkSequence), func(focused tview.Primitive) *tcell.EventKey {
			inst.appCtx.OpenLink(focused)
			return nil
		}},
	}
}

// KeySequencer runs the key sequences of the bindings, such as gg. The keys of
// a sequence are held back until it is complete. A key that ends no
// sequence drops the keys before it, as in Vim, and is then taken on its own.
// Text typed into an input is never held back. It is only used on the UI
// goroutine.
type KeySequencer struct {
	appCtx    *AppContext
	movePage  func(offset int)
	pending   []rune
	pendingAt time.Time
	now       func() time.Time
}

func NewKeySequencer(appCtx *AppContext) *KeySequencer {
	return &KeySequencer{appCtx: appCtx, movePage: func(int) {}, now: time.Now}
}

// SetMovePageFunc sets what moves the service in front to the page offset
// from the one shown, such as 1 for the next
func (inst *KeySequencer) SetMovePageFunc(movePage func(offset int)) {
	inst.movePage = movePage
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
	for _, sequence := range inst.keySequences() {
		if len(sequence.keys) == 0 {
			continue
		}
		if slices.Equal(sequence.keys, typed) {
			inst.pending = nil
			return sequence.does(focused)
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
