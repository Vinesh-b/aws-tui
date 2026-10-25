package core

import "github.com/gdamore/tcell/v2"

const DefaultThemeName = "Default"

// DefaultTheme is the theme the app starts with.
func DefaultTheme() AppTheme {
	return AppTheme{
		Name:                        DefaultThemeName,
		PrimaryTextColour:           tcell.NewHexColor(0xBFBFBF),
		SecondaryTextColour:         tcell.NewHexColor(0xFFFFFF),
		TertiaryTextColour:          tcell.NewHexColor(0xCC8B00),
		TitleColour:                 tcell.NewHexColor(0x43B143),
		BorderColour:                tcell.NewHexColor(0x404040),
		InverseTextColour:           tcell.NewHexColor(0x404040),
		BackgroundColour:            tcell.ColorDefault,
		ContrastBackgroundColor:     tcell.NewHexColor(0x303030),
		MoreContrastBackgroundColor: tcell.NewHexColor(0x404040),
		PlaceholderTextColour:       tcell.NewHexColor(0x717171),
	}
}

// FindThemePreset returns the preset with the name, if there is one.
func FindThemePreset(name string) (AppTheme, bool) {
	for _, preset := range ThemePresets() {
		if preset.Name == name {
			return preset, true
		}
	}

	return AppTheme{}, false
}

// ThemePresets are the themes that can be chosen in the settings. They are all
// for a dark background, since the colour each service uses to mark the
// selected row is picked to be read on one.
func ThemePresets() []AppTheme {
	return []AppTheme{
		DefaultTheme(),
		{
			Name:                        "High Contrast",
			PrimaryTextColour:           tcell.NewHexColor(0xFFFFFF),
			SecondaryTextColour:         tcell.NewHexColor(0xFFFFFF),
			TertiaryTextColour:          tcell.NewHexColor(0xFFD700),
			TitleColour:                 tcell.NewHexColor(0x00FF7F),
			BorderColour:                tcell.NewHexColor(0x808080),
			InverseTextColour:           tcell.NewHexColor(0x000000),
			BackgroundColour:            tcell.ColorDefault,
			ContrastBackgroundColor:     tcell.NewHexColor(0x1C1C1C),
			MoreContrastBackgroundColor: tcell.NewHexColor(0x505050),
			PlaceholderTextColour:       tcell.NewHexColor(0x9E9E9E),
		},
		{
			Name:                        "Nord",
			PrimaryTextColour:           tcell.NewHexColor(0xD8DEE9),
			SecondaryTextColour:         tcell.NewHexColor(0xECEFF4),
			TertiaryTextColour:          tcell.NewHexColor(0xEBCB8B),
			TitleColour:                 tcell.NewHexColor(0xA3BE8C),
			BorderColour:                tcell.NewHexColor(0x4C566A),
			InverseTextColour:           tcell.NewHexColor(0x2E3440),
			BackgroundColour:            tcell.NewHexColor(0x2E3440),
			ContrastBackgroundColor:     tcell.NewHexColor(0x3B4252),
			MoreContrastBackgroundColor: tcell.NewHexColor(0x434C5E),
			PlaceholderTextColour:       tcell.NewHexColor(0x7B88A1),
		},
		{
			Name:                        "Solarized Dark",
			PrimaryTextColour:           tcell.NewHexColor(0x93A1A1),
			SecondaryTextColour:         tcell.NewHexColor(0xEEE8D5),
			TertiaryTextColour:          tcell.NewHexColor(0xB58900),
			TitleColour:                 tcell.NewHexColor(0x859900),
			BorderColour:                tcell.NewHexColor(0x586E75),
			InverseTextColour:           tcell.NewHexColor(0x002B36),
			BackgroundColour:            tcell.NewHexColor(0x002B36),
			ContrastBackgroundColor:     tcell.NewHexColor(0x073642),
			MoreContrastBackgroundColor: tcell.NewHexColor(0x124A58),
			PlaceholderTextColour:       tcell.NewHexColor(0x586E75),
		},
	}
}
