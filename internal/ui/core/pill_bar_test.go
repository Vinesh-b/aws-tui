package core

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

func TestPillBar__DrawIsCutOffAtMaxX(t *testing.T) {
	var theme = DefaultTheme()
	var bar = NewPillBar(&theme).SetPills([]string{"first", "second"})
	var screen = newTestScreen(t, 20, 1)

	if end := bar.Draw(screen, 0, 0, 10); end != 10 {
		t.Errorf("end = %d, want 10", end)
	}
	screen.Show()

	if line := screenLine(screen); line != "🭃first🭎🭃se"+strings.Repeat(" ", 10) {
		t.Errorf("line = %q, want the bar cut off after 10 cells", line)
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

	if line := screenLine(screen); !strings.HasPrefix(line, " 🭃Details🭎🭃Tags🭎") {
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
	var tagsX = 1 + pillWidth("Details")
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
