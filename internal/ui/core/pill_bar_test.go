package core

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

func newTestScreen(t *testing.T, width int, height int) tcell.SimulationScreen {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	return screen
}

func TestPillBar__WidthCountsPillsAndGaps(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills([]string{"ab", "日本"})

	// Each pill has two edge cells, and the wide runes take two cells each
	if got := bar.Width(); got != 4+6 {
		t.Errorf("width = %d, want 10", got)
	}
	if got := bar.SetGap(2).Width(); got != 4+2+6 {
		t.Errorf("width with a gap = %d, want 12", got)
	}
	if got := NewPillBar(&theme).Width(); got != 0 {
		t.Errorf("empty bar width = %d, want 0", got)
	}
}

func TestPillBar__PillAtFindsPillsAndSkipsGaps(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills([]string{"ab", "cd"}).SetGap(1)

	// Layout: [0..3] "ab" pill, [4] gap, [5..8] "cd" pill
	for _, test := range []struct {
		offset int
		idx    int
		ok     bool
	}{
		{-1, 0, false},
		{0, 0, true},
		{3, 0, true},
		{4, 0, false},
		{5, 1, true},
		{8, 1, true},
		{9, 0, false},
	} {
		if idx, ok := bar.PillAt(test.offset); idx != test.idx || ok != test.ok {
			t.Errorf("PillAt(%d) = %d, %v, want %d, %v", test.offset, idx, ok, test.idx, test.ok)
		}
	}
}

func TestPillBar__DrawsCurrentPillHighlighted(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills([]string{"first", "second"}).SetCurrent(1).SetGap(1)
	var screen = newTestScreen(t, 40, 1)

	if end := bar.Draw(screen, 2, 0, 40); end != 2+bar.Width() {
		t.Errorf("end = %d, want %d", end, 2+bar.Width())
	}
	screen.Show()

	if line := screenLine(screen); !strings.HasPrefix(line, "  🭃first🭎 🭃second🭎") {
		t.Errorf("line = %q, want the pills from x 2", line)
	}
	for _, want := range []struct {
		text   string
		fg, bg tcell.Color
	}{
		{"first", theme.PrimaryTextColour, theme.ContrastBackgroundColor},
		{"second", theme.SecondaryTextColour, theme.MoreContrastBackgroundColor},
		// The edges are the colour of the pill on the background around it
		{"🭃first", theme.ContrastBackgroundColor, theme.BackgroundColour},
		{"🭃second", theme.MoreContrastBackgroundColor, theme.BackgroundColour},
	} {
		if fg, bg := lineColours(t, screen, want.text); fg != want.fg || bg != want.bg {
			t.Errorf("%s = fg %x bg %x, want fg %x bg %x",
				want.text, fg.Hex(), bg.Hex(), want.fg.Hex(), want.bg.Hex())
		}
	}
}

func TestPillBar__OnePillWiderThanTheBarIsCutOff(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills([]string{"first", "second"})
	var screen = newTestScreen(t, 20, 1)

	bar.Draw(screen, 0, 0, 7)
	screen.Show()

	// Without room for even one pill and the mark, the current pill is cut off
	if line := screenLine(screen); !strings.HasPrefix(line, "🭃firs") {
		t.Errorf("line = %q, want the current pill cut off", line)
	}
}

// scrollingBarPills take 38 cells, which don't fit in 30
var scrollingBarPills = []string{"Details", "Logs", "Tags", "Invoke", "Metrics"}

func drawScrollingBar(t *testing.T, current int) (*PillBar, string, int) {
	t.Helper()
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills(scrollingBarPills).SetCurrent(current)
	var screen = newTestScreen(t, 30, 1)

	var end = bar.Draw(screen, 0, 0, 30)
	screen.Show()

	return bar, screenLine(screen), end
}

func TestPillBar__BarThatFitsIsNotScrolled(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills(scrollingBarPills).SetCurrent(4)
	var screen = newTestScreen(t, 40, 1)

	bar.Draw(screen, 0, 0, 40)
	screen.Show()

	if line := screenLine(screen); line != "🭃Details🭎🭃Logs🭎🭃Tags🭎🭃Invoke🭎🭃Metrics🭎  " {
		t.Errorf("line = %q, want every pill without marks", line)
	}
}

func TestPillBar__ScrolledBarShowsTheStartWithMoreToTheRight(t *testing.T) {
	var _, line, end = drawScrollingBar(t, 0)

	if line != "🭃Details🭎🭃Logs🭎🭃Tags🭎 ›       " {
		t.Errorf("line = %q, want the first pills and a mark for the rest", line)
	}
	if end != 23 {
		t.Errorf("end = %d, want the x after the mark", end)
	}
}

func TestPillBar__ScrolledBarKeepsTheCurrentPillInView(t *testing.T) {
	var _, line, _ = drawScrollingBar(t, 4)

	if line != "‹ 🭃Tags🭎🭃Invoke🭎🭃Metrics🭎     " {
		t.Errorf("line = %q, want the last pills and a mark for those before", line)
	}
}

func TestPillBar__ScrolledBarShowsMarksOnBothSides(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills(scrollingBarPills).SetCurrent(4)
	var screen = newTestScreen(t, 20, 1)
	bar.Draw(screen, 0, 0, 20)

	// Moving back keeps the pills after it out of view
	bar.SetCurrent(2)
	bar.Draw(screen, 0, 0, 20)
	screen.Show()

	if line := screenLine(screen); line != "‹ 🭃Tags🭎🭃Invoke🭎 ›  " {
		t.Errorf("line = %q, want marks on both sides of the current pill", line)
	}
}

func TestPillBar__ScrollingBackShowsEarlierPills(t *testing.T) {
	var bar, _, _ = drawScrollingBar(t, 4)
	var screen = newTestScreen(t, 30, 1)

	bar.SetCurrent(0)
	bar.Draw(screen, 0, 0, 30)
	screen.Show()

	if line := screenLine(screen); !strings.HasPrefix(line, "🭃Details🭎") {
		t.Errorf("line = %q, want the bar scrolled back to the start", line)
	}
}

func TestPillBar__ClicksFindThePillsWhereTheyWereDrawn(t *testing.T) {
	var bar, _, _ = drawScrollingBar(t, 4)

	var tests = []struct {
		offset int
		idx    int
		ok     bool
	}{
		{0, 0, false}, // the scroll mark
		{2, 2, true},  // Tags, the first pill drawn
		{8, 3, true},  // Invoke
		{24, 4, true}, // the end of Metrics
		{25, 0, false},
	}
	for _, test := range tests {
		if idx, ok := bar.PillAt(test.offset); idx != test.idx || ok != test.ok {
			t.Errorf("PillAt(%d) = %d, %v, want %d, %v", test.offset, idx, ok, test.idx, test.ok)
		}
	}
}

func TestPillBar__NewPillsResetTheScroll(t *testing.T) {
	var bar, _, _ = drawScrollingBar(t, 4)

	bar.SetPills([]string{"a", "b"})

	if bar.first != 0 {
		t.Errorf("first = %d, want 0 for new pills", bar.first)
	}
	if idx, ok := bar.PillAt(4); idx != 1 || !ok {
		t.Errorf("PillAt(4) = %d, %v, want the second pill before the bar is drawn again", idx, ok)
	}
}

func TestPillBar__WideningTheBarShowsEveryPill(t *testing.T) {
	var bar, _, _ = drawScrollingBar(t, 4)
	var screen = newTestScreen(t, 40, 1)

	bar.Draw(screen, 0, 0, 40)
	screen.Show()

	if line := screenLine(screen); !strings.HasPrefix(line, "🭃Details🭎") || strings.Contains(line, "‹") {
		t.Errorf("line = %q, want every pill once there is room", line)
	}
}

func newTestTabView(t *testing.T) (*TabViewHorizontal, *AppTheme) {
	t.Helper()
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)
	var tabView = NewTabViewHorizontal(appCtx).
		AddAndSwitchToTab("Details", tview.NewBox(), 0, 1, true).
		AddTab("Tags", tview.NewBox(), 0, 1, true)
	tabView.SetRect(0, 0, 40, 5)
	return tabView, &theme
}

func TestTabViewHorizontal__DrawsTabsAsPills(t *testing.T) {
	var tabView, theme = newTestTabView(t)
	var screen = newTestScreen(t, 40, 5)
	tabView.Draw(screen)
	screen.Show()

	if line := screenLine(screen); !strings.HasPrefix(line, " 🭃¹Details🭎🭃²Tags🭎") {
		t.Errorf("line = %q, want the tabs as pills from x 1", line)
	}
	if fg, bg := lineColours(t, screen, "Details"); fg != theme.SecondaryTextColour || bg != theme.MoreContrastBackgroundColor {
		t.Errorf("current tab = fg %x bg %x, want the active pill colours", fg.Hex(), bg.Hex())
	}
}

func TestTabViewHorizontal__ClickingATabSwitchesToIt(t *testing.T) {
	var tabView, _ = newTestTabView(t)
	var click = func(x int, y int) {
		var event = tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone)
		tabView.MouseHandler()(tview.MouseLeftClick, event, func(tview.Primitive) {})
	}

	// "Tags" starts after the offset and the "Details" pill
	var tagsX = 1 + pillWidth("¹Details")
	click(tagsX, 1)
	if name, _ := tabView.GetCurrentTab(); name != "Details" {
		t.Errorf("a click below the bar switched to %q", name)
	}

	click(tagsX, 0)
	if name, _ := tabView.GetCurrentTab(); name != "Tags" {
		t.Errorf("current tab = %q, want Tags", name)
	}

	click(0, 0)
	if name, _ := tabView.GetCurrentTab(); name != "Tags" {
		t.Errorf("a click on the offset switched to %q", name)
	}
}

func TestTabViewHorizontal__ScrollsToTheCurrentTabAndClicksFollow(t *testing.T) {
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)
	var tabView = NewTabViewHorizontal(appCtx)
	for idx, name := range scrollingBarPills {
		if idx == 0 {
			tabView.AddAndSwitchToTab(name, tview.NewBox(), 0, 1, true)
		} else {
			tabView.AddTab(name, tview.NewBox(), 0, 1, true)
		}
	}
	tabView.SetRect(0, 0, 31, 5)
	var screen = newTestScreen(t, 31, 5)

	tabView.SwitchToTab("Metrics")
	tabView.Draw(screen)
	screen.Show()

	if line := screenLine(screen); line != " ‹ 🭃³Tags🭎🭃⁴Invoke🭎🭃⁵Metrics🭎  " {
		t.Errorf("line = %q, want the bar scrolled to Metrics", line)
	}

	// "Invoke" is drawn after the offset, the mark and the "Tags" pill
	var invokeX = 1 + uniseg.StringWidth(scrolledLeftMark) + pillWidth("³Tags")
	var event = tcell.NewEventMouse(invokeX, 0, tcell.Button1, tcell.ModNone)
	tabView.MouseHandler()(tview.MouseLeftClick, event, func(tview.Primitive) {})

	if name, _ := tabView.GetCurrentTab(); name != "Invoke" {
		t.Errorf("current tab = %q, want the clicked Invoke", name)
	}
}

func TestTabLabels__NumberTheFirstNineTabs(t *testing.T) {
	var names = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	var labels = tabLabels(names)

	if labels[0] != "¹a" || labels[8] != "⁹i" || labels[9] != "j" {
		t.Errorf("labels = %v, want the first nine numbered", labels)
	}
	if got := uniseg.StringWidth(labels[0]); got != 2 {
		t.Errorf("label width = %d, want the number to take one cell", got)
	}
}

func pressTabRune(tabView *TabViewHorizontal, r rune, mod tcell.ModMask) {
	tabView.InputHandler()(tcell.NewEventKey(tcell.KeyRune, r, mod), func(p tview.Primitive) {})
}

func newNumberedTabView(t *testing.T, focused tview.Primitive) (*TabViewHorizontal, *AppContext) {
	t.Helper()
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)
	var tabView = NewTabViewHorizontal(appCtx).
		AddAndSwitchToTab("Details", tview.NewBox(), 0, 1, true).
		AddTab("Triggers", tview.NewBox(), 0, 1, true).
		AddTab("Tags", focused, 0, 1, true)
	appCtx.App.SetFocus(focused)
	return tabView, appCtx
}

func TestTabViewHorizontal__NumberKeysSwitchTabs(t *testing.T) {
	var tabView, _ = newNumberedTabView(t, tview.NewBox())
	var changed []string
	tabView.SetOnTabChangeFunc(func(name string, index int) { changed = append(changed, name) })

	pressTabRune(tabView, '3', tcell.ModNone)
	if name, _ := tabView.GetCurrentTab(); name != "Tags" {
		t.Errorf("after 3 the tab = %q, want Tags", name)
	}

	pressTabRune(tabView, '9', tcell.ModNone)
	if name, _ := tabView.GetCurrentTab(); name != "Tags" {
		t.Errorf("a number past the last tab switched to %q", name)
	}

	pressTabRune(tabView, '1', tcell.ModAlt)
	if name, _ := tabView.GetCurrentTab(); name != "Tags" {
		t.Errorf("alt+1 switched to %q, want it left for other keys", name)
	}

	pressTabRune(tabView, '1', tcell.ModNone)
	if name, _ := tabView.GetCurrentTab(); name != "Details" {
		t.Errorf("after 1 the tab = %q, want Details", name)
	}
	if len(changed) != 2 || changed[0] != "Tags" || changed[1] != "Details" {
		t.Errorf("tab changes = %v, want Tags then Details", changed)
	}
}

func TestTabViewHorizontal__NumberKeysAreTypedIntoTextEntries(t *testing.T) {
	var theme = DefaultTheme()
	var tests = []struct {
		name    string
		focused tview.Primitive
		typed   bool
	}{
		{"input field", NewInputField(&theme), true},
		{"tview input field", tview.NewInputField(), true},
		{"date input", NewDateTimeInputField(&theme), true},
		{"text area", NewTextArea("Payload", newTestAppContext()), true},
		{"tview text area", tview.NewTextArea(), true},
		{"disabled text area", tview.NewTextArea().SetDisabled(true).(*tview.TextArea), false},
		{"read only text", NewSearchableTextView("Policy", &AppContext{Theme: &theme, App: tview.NewApplication()}).syntaxArea, false},
		{"table", tview.NewTable(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tabView, _ = newNumberedTabView(t, tt.focused)
			tabView.SwitchToTab("Tags")

			pressTabRune(tabView, '1', tcell.ModNone)

			var name, _ = tabView.GetCurrentTab()
			if switched := name != "Tags"; switched == tt.typed {
				t.Errorf("switched = %v, want the key typed = %v", switched, tt.typed)
			}
		})
	}
}
