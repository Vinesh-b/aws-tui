package core

import (
	"slices"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type sequencerTest struct {
	t         *testing.T
	appCtx    *AppContext
	sequencer *KeySequencer
	clock     time.Time
	focused   tview.Primitive
	moves     []int
}

func newSequencerTest(t *testing.T) *sequencerTest {
	var test = &sequencerTest{t: t, appCtx: newTestAppContext(), clock: time.Now(), focused: tview.NewTable()}
	test.sequencer = NewKeySequencer(test.appCtx)
	test.sequencer.now = func() time.Time { return test.clock }
	test.sequencer.SetMovePageFunc(func(offset int) { test.moves = append(test.moves, offset) })
	return test
}

// press types the rune and returns the event passed on to the focused view
func (test *sequencerTest) press(r rune) *tcell.EventKey {
	return test.sequencer.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), test.focused)
}

func (test *sequencerTest) assertHeld(r rune) {
	test.t.Helper()
	if got := test.press(r); got != nil {
		test.t.Fatalf("%q was passed on as %v, want it held for the rest of the sequence", r, got.Name())
	}
}

func (test *sequencerTest) assertPassed(r rune, want rune, mod tcell.ModMask) {
	test.t.Helper()
	var got = test.press(r)
	if got == nil {
		test.t.Fatalf("%q was held, want %q passed on", r, want)
	}
	if got.Key() != tcell.KeyRune || got.Rune() != want || got.Modifiers() != mod {
		test.t.Errorf("%q was passed on as %v, want %q with %v", r, got.Name(), want, mod)
	}
}

func keepBindings(t *testing.T) {
	var original = APP_KEY_BINDINGS
	t.Cleanup(func() { APP_KEY_BINDINGS = original })
}

func TestKeySequencer__SomeSequencesDoTheKeyTheViewsKnow(t *testing.T) {
	var keys = APP_KEY_BINDINGS
	var tests = map[string]rune{
		"gg": keys.MovePageTopRune,
		"dd": keys.RemoveItem,
	}
	for sequence, does := range tests {
		t.Run(sequence, func(t *testing.T) {
			var test = newSequencerTest(t)
			var typed = []rune(sequence)

			test.assertHeld(typed[0])
			test.assertPassed(typed[1], does, tcell.ModNone)
		})
	}
}

func TestKeySequencer__GtAndGTMoveThePage(t *testing.T) {
	var test = newSequencerTest(t)

	test.assertHeld('g')
	test.assertHeld('t')
	test.assertHeld('g')
	test.assertHeld('T')

	if !slices.Equal(test.moves, []int{1, -1}) {
		t.Errorf("moves = %v, want next then previous", test.moves)
	}
}

func TestKeySequencer__GxOpensTheLinkOfTheFocusedDetails(t *testing.T) {
	var test = newSequencerTest(t)
	var opened []string
	test.appCtx.SetResourceOpener(func(resource arn.ARN) error {
		opened = append(opened, resource.String())
		return nil
	}, func(arn.ARN) bool { return true })
	var details = NewDetailsTable("details", test.appCtx)
	var link = "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"
	details.SetData([]TableRow{DetailsLink("Log Group", "/aws/lambda/orders", link)})
	details.Select(0, 0)
	test.appCtx.App.SetFocus(details.table)
	test.focused = details.table

	test.assertHeld('g')
	test.assertHeld('x')

	if !slices.Equal(opened, []string{link}) {
		t.Errorf("opened = %v, want the link of the row", opened)
	}
}

func TestKeySequencer__KeysThatStartNoSequenceArePassedOn(t *testing.T) {
	var test = newSequencerTest(t)

	test.assertPassed('j', 'j', tcell.ModNone)
	test.assertPassed('G', 'G', tcell.ModNone)
}

func TestKeySequencer__AKeyThatEndsNoSequenceDropsTheKeysBeforeIt(t *testing.T) {
	var test = newSequencerTest(t)
	test.assertHeld('g')

	test.assertPassed('j', 'j', tcell.ModNone)
	// The g was dropped, so this g starts a new sequence
	test.assertHeld('g')
}

func TestKeySequencer__TheRestOfASequenceIsOnlyWaitedForAWhile(t *testing.T) {
	var test = newSequencerTest(t)
	test.assertHeld('g')

	test.clock = test.clock.Add(SequenceTimeout + time.Millisecond)

	test.assertHeld('g')
	test.assertHeld('t')
	if !slices.Equal(test.moves, []int{1}) {
		t.Errorf("moves = %v, want the gt typed after the wait", test.moves)
	}
}

func TestKeySequencer__OtherKeysStopTheSequenceAndStillWork(t *testing.T) {
	var test = newSequencerTest(t)
	test.assertHeld('g')

	var esc = tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
	if got := test.sequencer.HandleEvent(esc, test.focused); got != esc {
		t.Errorf("Esc was passed on as %v", got)
	}

	test.assertPassed('t', 't', tcell.ModNone)
}

func TestKeySequencer__TextTypedIntoAnInputIsNeverHeld(t *testing.T) {
	var test = newSequencerTest(t)
	test.focused = tview.NewInputField()

	test.assertPassed('g', 'g', tcell.ModNone)
	test.assertPassed('g', 'g', tcell.ModNone)
}

func TestKeySequencer__FollowsTheSequenceBindings(t *testing.T) {
	keepBindings(t)
	APP_KEY_BINDINGS.MoveTopSequence = "tt"
	APP_KEY_BINDINGS.OpenLinkSequence = ""
	var test = newSequencerTest(t)

	test.assertHeld('t')
	test.assertPassed('t', APP_KEY_BINDINGS.MovePageTopRune, tcell.ModNone)

	// gx is off, so the x after the g of gt and gT is taken on its own
	test.assertHeld('g')
	test.assertPassed('x', 'x', tcell.ModNone)
}

func TestKeySequencer__GgGoesToTheTopOfATable(t *testing.T) {
	var test = newSequencerTest(t)
	var table = tview.NewTable().SetSelectable(true, false)
	for row := range 5 {
		table.SetCellSimple(row, 0, "row")
	}
	table.Select(4, 0)
	test.focused = table
	var press = func(r rune) {
		if event := test.press(r); event != nil {
			table.InputHandler()(event, func(tview.Primitive) {})
		}
	}

	press('g')
	if row, _ := table.GetSelection(); row != 4 {
		t.Errorf("row after g = %d, want 4 until the sequence is done", row)
	}
	press('g')
	if row, _ := table.GetSelection(); row != 0 {
		t.Errorf("row after gg = %d, want 0", row)
	}
}
