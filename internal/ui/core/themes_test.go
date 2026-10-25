package core

import (
	"context"
	"testing"

	"github.com/rivo/tview"
)

// keepGlobalStyle restores the tview styles, which a theme change sets for the
// whole process.
func keepGlobalStyle(t *testing.T) {
	t.Helper()
	var styles, borders = tview.Styles, tview.Borders
	t.Cleanup(func() { tview.Styles, tview.Borders = styles, borders })
}

func TestThemePresets__StartWithTheDefaultAndHaveUniqueNames(t *testing.T) {
	var presets = ThemePresets()

	if presets[0].Name != DefaultThemeName || presets[0] != DefaultTheme() {
		t.Errorf("first preset = %q, want the default theme", presets[0].Name)
	}
	var seen = map[string]bool{}
	for _, preset := range presets {
		if preset.Name == "" || seen[preset.Name] {
			t.Errorf("preset name %q is empty or repeated", preset.Name)
		}
		seen[preset.Name] = true
		if preset.PrimaryTextColour == preset.BackgroundColour {
			t.Errorf("%s: text and background are the same colour", preset.Name)
		}
	}
}

func TestSetTheme__ChangesTheSharedThemeAndTheGlobalStyle(t *testing.T) {
	keepGlobalStyle(t)
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)
	var nord = ThemePresets()[2]

	appCtx.SetTheme(nord)

	if appCtx.Theme != &theme || theme.Name != nord.Name || theme.BorderColour != nord.BorderColour {
		t.Errorf("shared theme = %+v, want it changed in place to Nord", theme)
	}
	if tview.Styles.BorderColor != nord.BorderColour || tview.Styles.PrimitiveBackgroundColor != nord.BackgroundColour {
		t.Error("the global style was not set from the new theme")
	}
}

func TestSetTheme__TellsListenersAfterTheThemeChanged(t *testing.T) {
	keepGlobalStyle(t)
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)
	var seen []string
	appCtx.OnThemeChanged(func() { seen = append(seen, theme.Name) })
	appCtx.OnThemeChanged(func() { seen = append(seen, "second") })

	appCtx.SetTheme(ThemePresets()[1])

	if len(seen) != 2 || seen[0] != "High Contrast" || seen[1] != "second" {
		t.Errorf("listeners saw %v, want the new theme name then the second listener", seen)
	}
}

func TestRestyleChrome__ColoursNestedViewsAndSkipsGaps(t *testing.T) {
	keepGlobalStyle(t)
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)
	var text = tview.NewTextView()
	var list = tview.NewList()
	var filterable = NewFilterableList(tview.NewList(), appCtx)
	var pages = tview.NewPages().AddPage("list", list, true, true)
	var inner = tview.NewFlex().AddItem(nil, 0, 1, false).AddItem(text, 0, 1, false)
	var root = tview.NewFlex().AddItem(inner, 0, 1, false).AddItem(pages, 0, 1, false).AddItem(filterable, 0, 1, false)
	root.SetBorder(true)

	var nord = ThemePresets()[2]
	RestyleChrome(root, &nord)

	if got := root.GetBorderColor(); got != nord.BorderColour {
		t.Errorf("root border = %v, want %v", got, nord.BorderColour)
	}
	if got := text.GetBackgroundColor(); got != nord.BackgroundColour {
		t.Errorf("nested text view background = %v, want %v", got, nord.BackgroundColour)
	}
	if got := pages.GetBackgroundColor(); got != nord.BackgroundColour {
		t.Errorf("pages background = %v, want %v", got, nord.BackgroundColour)
	}
	if got := list.GetBackgroundColor(); got != nord.BackgroundColour {
		t.Errorf("list in pages background = %v, want %v", got, nord.BackgroundColour)
	}
	if got := filterable.GetBackgroundColor(); got != nord.BackgroundColour {
		t.Errorf("filterable list background = %v, want %v", got, nord.BackgroundColour)
	}
}

func TestRestyleChrome__IgnoresNil(t *testing.T) {
	var theme = DefaultTheme()

	RestyleChrome(nil, &theme)
}

func TestFindThemePreset__FindsPresetsByName(t *testing.T) {
	for _, preset := range ThemePresets() {
		if found, ok := FindThemePreset(preset.Name); !ok || found != preset {
			t.Errorf("FindThemePreset(%q) = %+v, %v, want the preset", preset.Name, found, ok)
		}
	}
	if _, ok := FindThemePreset("Does Not Exist"); ok {
		t.Error("an unknown name should not be found")
	}
}
