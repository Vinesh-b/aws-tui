package core

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// FormatKey renders a special key the way it appears in help views, for
// example "Esc", "PgUp" or "Ctrl+X".
func FormatKey(key tcell.Key) string {
	switch key {
	case tcell.KeyESC:
		return "Esc"
	case tcell.KeyEnter:
		return "Enter"
	case tcell.KeyTab:
		return "Tab"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "Backspace"
	case tcell.KeyPgUp:
		return "PgUp"
	case tcell.KeyPgDn:
		return "PgDn"
	case tcell.KeyUp:
		return "Up"
	case tcell.KeyDown:
		return "Down"
	case tcell.KeyLeft:
		return "Left"
	case tcell.KeyRight:
		return "Right"
	case tcell.KeyHome:
		return "Home"
	case tcell.KeyEnd:
		return "End"
	case tcell.KeyCtrlSpace:
		return "Ctrl+Space"
	}

	if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
		return "Ctrl+" + string(rune('A'+key-tcell.KeyCtrlA))
	}

	return tcell.KeyNames[key]
}

// FormatRune renders a rune binding for help views. Space is spelled out
// because a bare " " is invisible in a table cell.
func FormatRune(r rune) string {
	if r == ' ' {
		return "Space"
	}
	return string(r)
}

// FormatModRune renders a rune pressed with a modifier, for example "Alt+]".
func FormatModRune(mod tcell.ModMask, r rune) string {
	return strings.Join(append(modifierNames(mod), FormatRune(r)), "+")
}

// FormatModKey renders a special key pressed with a modifier, for example
// "Alt+Left".
func FormatModKey(mod tcell.ModMask, key tcell.Key) string {
	return strings.Join(append(modifierNames(mod), FormatKey(key)), "+")
}

func modifierNames(mod tcell.ModMask) []string {
	var names []string
	if mod&tcell.ModCtrl != 0 {
		names = append(names, "Ctrl")
	}
	if mod&tcell.ModAlt != 0 {
		names = append(names, "Alt")
	}
	if mod&tcell.ModShift != 0 {
		names = append(names, "Shift")
	}
	return names
}
