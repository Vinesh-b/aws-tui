package core

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// cellColours finds the text on the screen and returns the colours of its first cell.
func cellColours(t *testing.T, screen tcell.SimulationScreen, text string) (fg, bg tcell.Color) {
	t.Helper()
	var cells, width, height = screen.GetContents()
	for row := 0; row < height; row++ {
		var line strings.Builder
		for col := 0; col < width; col++ {
			line.WriteString(string(cells[row*width+col].Runes))
		}
		if idx := strings.Index(line.String(), text); idx >= 0 {
			var fg, bg, _ = cells[row*width+idx].Style.Decompose()
			return fg, bg
		}
	}
	t.Fatalf("%q is not on the screen", text)
	return 0, 0
}

func drawFilterableList(t *testing.T, list *FilterableList) tcell.SimulationScreen {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(40, 10)
	list.SetRect(0, 0, 40, 10)
	list.Focus(func(tview.Primitive) {})
	list.Draw(screen)
	screen.Show()
	return screen
}

func newThemedFilterableList(theme *AppTheme) *FilterableList {
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, theme)
	var list = NewFilterableList(tview.NewList().SetHighlightFullLine(true), appCtx)
	list.AddItem("first", "about first", 0, func() {})
	list.AddItem("second", "about second", 'x', func() {})
	return list
}

func assertListColours(t *testing.T, screen tcell.SimulationScreen, theme AppTheme) {
	t.Helper()
	if fg, bg := cellColours(t, screen, "first"); fg != theme.SecondaryTextColour || bg != theme.MoreContrastBackgroundColor {
		t.Errorf("%s: selected row = fg %x bg %x, want fg %x bg %x",
			theme.Name, fg.Hex(), bg.Hex(), theme.SecondaryTextColour.Hex(), theme.MoreContrastBackgroundColor.Hex())
	}
	// The background of text is part of its style, not of the list
	for _, want := range []struct {
		text string
		fg   tcell.Color
	}{
		{"second", theme.PrimaryTextColour},
		{"about second", theme.PlaceholderTextColour},
		{"about first", theme.PlaceholderTextColour},
		{"(x)", theme.SecondaryTextColour},
	} {
		var fg, bg = cellColours(t, screen, want.text)
		if fg != want.fg || bg != theme.BackgroundColour {
			t.Errorf("%s: %q = fg %x bg %x, want fg %x bg %x",
				theme.Name, want.text, fg.Hex(), bg.Hex(), want.fg.Hex(), theme.BackgroundColour.Hex())
		}
	}
}

func TestFilterableList__IsColouredFromTheTheme(t *testing.T) {
	keepGlobalStyle(t)
	for _, preset := range ThemePresets() {
		var theme = preset
		var list = newThemedFilterableList(&theme)

		assertListColours(t, drawFilterableList(t, list), preset)
	}
}

func TestRestyleChrome__GivesAFilterableListTheNewTheme(t *testing.T) {
	keepGlobalStyle(t)
	var theme = DefaultTheme()
	var list = newThemedFilterableList(&theme)
	var nord = ThemePresets()[2]

	RestyleChrome(list, &nord)

	assertListColours(t, drawFilterableList(t, list), nord)
}

func TestRestyleChrome__ColoursInputFields(t *testing.T) {
	keepGlobalStyle(t)
	var input = tview.NewInputField().SetLabel("Search ").SetPlaceholder("type here")
	var nord = ThemePresets()[2]

	RestyleChrome(input, &nord)

	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(30, 3)
	input.SetRect(0, 0, 30, 3)
	input.Draw(screen)
	screen.Show()

	if fg, bg := cellColours(t, screen, "Search"); fg != nord.SecondaryTextColour || bg != nord.BackgroundColour {
		t.Errorf("label = fg %x bg %x, want fg %x bg %x",
			fg.Hex(), bg.Hex(), nord.SecondaryTextColour.Hex(), nord.BackgroundColour.Hex())
	}
	if fg, bg := cellColours(t, screen, "type here"); fg != nord.PlaceholderTextColour || bg != nord.ContrastBackgroundColor {
		t.Errorf("placeholder = fg %x bg %x, want fg %x bg %x",
			fg.Hex(), bg.Hex(), nord.PlaceholderTextColour.Hex(), nord.ContrastBackgroundColor.Hex())
	}
}
