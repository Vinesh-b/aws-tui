package core

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func lexKinds(line string) map[string]jsonTokenKind {
	var runes = []rune(line)
	var kinds = map[string]jsonTokenKind{}
	for _, span := range lexJsonLine(runes) {
		kinds[string(runes[span.start:span.end])] = span.kind
	}
	return kinds
}

func TestLexJsonLine__ClassifiesKeysStringsNumbersAndLiterals(t *testing.T) {
	var tests = []struct {
		line string
		want map[string]jsonTokenKind
	}{
		{`  "name": "orders",`, map[string]jsonTokenKind{`"name"`: jsonKey, `"orders"`: jsonString}},
		{`  "count": -12.5e3,`, map[string]jsonTokenKind{`"count"`: jsonKey, `-12.5e3`: jsonNumber}},
		{`  "enabled": true`, map[string]jsonTokenKind{`"enabled"`: jsonKey, `true`: jsonLiteral}},
		{`  "none" : null`, map[string]jsonTokenKind{`"none"`: jsonKey, `null`: jsonLiteral}},
		{`    "a",`, map[string]jsonTokenKind{`"a"`: jsonString}},
		{`  "quote": "say \"hi\"",`, map[string]jsonTokenKind{`"quote"`: jsonKey, `"say \"hi\""`: jsonString}},
	}

	for _, test := range tests {
		var got = lexKinds(test.line)
		for text, kind := range test.want {
			if gotKind, ok := got[text]; !ok || gotKind != kind {
				t.Errorf("%s: token %s = %v (found %v), want %v", test.line, text, gotKind, ok, kind)
			}
		}
		if len(got) != len(test.want) {
			t.Errorf("%s: got %d tokens %v, want %d", test.line, len(got), got, len(test.want))
		}
	}
}

func TestLexJsonLine__PunctuationAndUnterminatedStringDoNotPanic(t *testing.T) {
	for _, line := range []string{`{`, `},`, `[`, `"unterminated`, `"ends with slash\`, ``} {
		lexJsonLine([]rune(line))
	}
}

func drawTextView(t *testing.T, view *SearchableTextView, width, height int) tcell.SimulationScreen {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	view.syntaxArea.SetRect(0, 0, width, height)
	view.syntaxArea.Draw(screen)
	return screen
}

func foregroundAt(screen tcell.SimulationScreen, x, y int) tcell.Color {
	var _, style, _ = screen.Get(x, y)
	var fg, _, _ = style.Decompose()
	return fg
}

func newJsonTextView(text string) *SearchableTextView {
	var view = NewSearchableTextView("json", newTestAppContext())
	view.SetText(text, false)
	return view
}

const sampleJson = "{\n  \"name\": \"orders\",\n  \"count\": 3,\n  \"on\": true\n}"

func TestSearchableTextView__JsonHighlightingColoursVisibleTokens(t *testing.T) {
	var view = newJsonTextView(sampleJson)
	view.SetJsonHighlighting(true)

	var screen = drawTextView(t, view, 30, 6)

	var tests = []struct {
		name string
		x, y int
		want tcell.Color
	}{
		{"key", 3, 1, jsonTokenColours[jsonKey]},
		{"string", 11, 1, jsonTokenColours[jsonString]},
		{"number", 11, 2, jsonTokenColours[jsonNumber]},
		{"literal", 8, 3, jsonTokenColours[jsonLiteral]},
	}
	for _, test := range tests {
		if got := foregroundAt(screen, test.x, test.y); got != test.want {
			t.Errorf("%s colour at (%d,%d) = %v, want %v", test.name, test.x, test.y, got, test.want)
		}
	}
	for _, colour := range jsonTokenColours {
		if got := foregroundAt(screen, 0, 0); got == colour {
			t.Errorf("punctuation was coloured %v", got)
		}
	}
}

func TestSearchableTextView__DisabledHighlightingLeavesStyleUntouched(t *testing.T) {
	var view = newJsonTextView(sampleJson)
	view.SetJsonHighlighting(true)
	view.SetJsonHighlighting(false)

	var screen = drawTextView(t, view, 30, 6)

	for _, colour := range jsonTokenColours {
		if got := foregroundAt(screen, 3, 1); got == colour {
			t.Errorf("key was coloured %v with highlighting disabled", got)
		}
	}
}

func TestSearchableTextView__HighlightingFollowsHorizontalScroll(t *testing.T) {
	var view = newJsonTextView("{\n  \"key\": \"value\"\n}")
	view.SetJsonHighlighting(true)
	view.syntaxArea.SetOffset(0, 4)

	var screen = drawTextView(t, view, 30, 4)

	// `"key"` now starts at column -2 and the value starts at column 5
	if got := foregroundAt(screen, 0, 1); got != jsonTokenColours[jsonKey] {
		t.Errorf("scrolled key colour = %v, want %v", got, jsonTokenColours[jsonKey])
	}
	if got := foregroundAt(screen, 5, 1); got != jsonTokenColours[jsonString] {
		t.Errorf("scrolled string colour = %v, want %v", got, jsonTokenColours[jsonString])
	}
}

func TestSearchableTextView__HighlightingIsDisabledByDefault(t *testing.T) {
	if newJsonTextView(sampleJson).JsonHighlighting() {
		t.Error("plain text views should not highlight")
	}
}

func TestJsonTextView__HighlightsByDefaultAndCanBeDisabled(t *testing.T) {
	var newView = func(disable bool) *JsonTextView[string] {
		return &JsonTextView[string]{
			TextView:            NewSearchableTextView("json", newTestAppContext()),
			ExtractTextFunc:     func(data string) string { return data },
			DisableHighlighting: disable,
		}
	}

	var enabled = newView(false)
	enabled.SetText(`{"a":1}`)
	var disabled = newView(true)
	disabled.SetText(`{"a":1}`)

	if !enabled.TextView.JsonHighlighting() {
		t.Error("json views should highlight by default")
	}
	if disabled.TextView.JsonHighlighting() {
		t.Error("DisableHighlighting did not turn highlighting off")
	}
}

func drawTextArea(t *testing.T, area *TextArea, width, height int) tcell.SimulationScreen {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	area.SetBorder(false)
	area.SetRect(0, 0, width, height)
	area.Draw(screen)
	return screen
}

func TestTextArea__JsonHighlightingColoursTheText(t *testing.T) {
	var area = NewTextArea("json", newTestAppContext().Theme)
	area.SetJsonHighlighting(true)
	area.SetText(sampleJson, false)

	var screen = drawTextArea(t, area, 30, 6)

	if got := foregroundAt(screen, 3, 1); got != jsonTokenColours[jsonKey] {
		t.Errorf("key colour = %v, want %v", got, jsonTokenColours[jsonKey])
	}
	if got := foregroundAt(screen, 11, 2); got != jsonTokenColours[jsonNumber] {
		t.Errorf("number colour = %v, want %v", got, jsonTokenColours[jsonNumber])
	}
}

func TestTextArea__HighlightingFollowsTypedText(t *testing.T) {
	var area = NewTextArea("json", newTestAppContext().Theme)
	area.SetJsonHighlighting(true)
	area.SetText("", false)

	// Typing goes through the input handler, which does not call SetText
	var handler = area.InputHandler()
	for _, r := range `"a": 1` {
		handler(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(p tview.Primitive) {})
	}
	var screen = drawTextArea(t, area, 20, 3)

	if got := foregroundAt(screen, 1, 0); got != jsonTokenColours[jsonKey] {
		t.Errorf("key colour = %v, want %v", got, jsonTokenColours[jsonKey])
	}
	if got := foregroundAt(screen, 5, 0); got != jsonTokenColours[jsonNumber] {
		t.Errorf("number colour = %v, want %v", got, jsonTokenColours[jsonNumber])
	}
}

func TestTextArea__NoHighlightingByDefaultOrWhenDisabled(t *testing.T) {
	var area = NewTextArea("json", newTestAppContext().Theme)
	area.SetText(sampleJson, false)

	for _, enabled := range []bool{false, true} {
		area.SetJsonHighlighting(enabled)
		if enabled {
			area.SetJsonHighlighting(false)
		}
		var screen = drawTextArea(t, area, 30, 6)
		for _, colour := range jsonTokenColours {
			if got := foregroundAt(screen, 3, 1); got == colour {
				t.Errorf("key was coloured %v with highlighting off", got)
			}
		}
	}
}
