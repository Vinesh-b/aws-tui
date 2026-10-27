package core

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

// keyBinding points at one field of KeyBindings, so a name from the settings
// file can change it. Exactly one of the pointers is set.
type keyBinding struct {
	rune *rune
	key  *tcell.Key
	mod  *tcell.ModMask
}

// keyBindingConflictGroups lists the bindings that are active in the same view
// and so must not share a key. Bindings that are meant to share a key, such as
// LoadMoreData and NextSearch, or Quit and TableQuery, are in no common group.
var keyBindingConflictGroups = [][]string{
	{
		"help", "reset", "loadMoreData", "saveTable", "find", "textCopy",
		"moveUp", "moveDown", "moveTop", "moveBottom",
	},
	{
		"help", "find", "nextSearch", "prevSearch", "moveUp", "moveDown",
		"moveLeft", "moveRight", "moveLineStart", "moveLineEnd", "moveTop",
		"moveBottom", "textCopy", "textViewWordRight", "textViewWordLeft",
		"textViewUndo",
	},
	{"viewFocusUp", "viewFocusDown", "viewFocusLeft", "viewFocusRight"},
	{"historyBack", "historyForward"},
}

func keyBindingFields(bindings *KeyBindings) map[string]keyBinding {
	return map[string]keyBinding{
		"help":               {rune: &bindings.Help},
		"quit":               {rune: &bindings.Quit},
		"escape":             {key: &bindings.Escape},
		"toggleServicesMenu": {key: &bindings.ToggleServicesMenu},
		"toggleServicePages": {key: &bindings.ToggleServicePages},
		"reset":              {rune: &bindings.Reset},
		"loadMoreData":       {rune: &bindings.LoadMoreData},
		"clearTable":         {key: &bindings.ClearTable},
		"saveTable":          {rune: &bindings.SaveTable},
		"done":               {key: &bindings.Done},
		"find":               {rune: &bindings.Find},
		"pageChangeModKey":   {mod: &bindings.PageChangeModKey},
		"pageForward":        {rune: &bindings.PageForward},
		"pageBack":           {rune: &bindings.PageBack},
		"historyModKey":      {mod: &bindings.HistoryModKey},
		"historyBack":        {key: &bindings.HistoryBack},
		"historyForward":     {key: &bindings.HistoryForward},
		"viewFocusUp":        {key: &bindings.ViewFocusUp},
		"viewFocusDown":      {key: &bindings.ViewFocusDown},
		"viewFocusLeft":      {key: &bindings.ViewFocusLeft},
		"viewFocusRight":     {key: &bindings.ViewFocusRight},
		"viewResizeModKey":   {mod: &bindings.ViewResizeModKey},
		"viewResizeReset":    {rune: &bindings.ViewResizeReset},
		"moveUp":             {rune: &bindings.MoveUpRune},
		"moveDown":           {rune: &bindings.MoveDownRune},
		"moveLeft":           {rune: &bindings.MoveLeftRune},
		"moveRight":          {rune: &bindings.MoveRightRune},
		"moveLineStart":      {rune: &bindings.MoveLineStartRune},
		"moveLineEnd":        {rune: &bindings.MoveLineEndRune},
		"moveTop":            {rune: &bindings.MovePageTopRune},
		"moveBottom":         {rune: &bindings.MovePageBottomRune},
		"nextSearch":         {rune: &bindings.NextSearch},
		"prevSearch":         {rune: &bindings.PrevSearch},
		"formFocusNext":      {key: &bindings.FormFocusNext},
		"formFocusPrev":      {key: &bindings.FormFocusPrev},
		"tableScan":          {rune: &bindings.TableScan},
		"tableQuery":         {rune: &bindings.TableQuery},
		"textCopy":           {rune: &bindings.TextCopy},
		"toggleRawView":      {rune: &bindings.ToggleRawView},
		"openLink":           {rune: &bindings.OpenLink},
		"textViewPageUp":     {key: &bindings.TextViewPageUp},
		"textViewPageDown":   {key: &bindings.TextViewPageDown},
		"textViewWordRight":  {rune: &bindings.TextViewWordRight},
		"textViewWordLeft":   {rune: &bindings.TextViewWordLeft},
		"textViewUndo":       {rune: &bindings.TextViewUndo},
		"textViewRedo":       {key: &bindings.TextViewRedo},
	}
}

// ApplyKeyBindingOverrides changes APP_KEY_BINDINGS to what the user chose in
// the settings file, keyed by binding name such as "reset" or "viewFocusUp". It
// must run before any view is built, as views list their keys in their help
// when they are created.
//
// An override that names no binding, cannot be read, or would give two
// bindings of one view the same key is skipped and returned as an error, so
// the caller can report it and the app still starts.
func ApplyKeyBindingOverrides(overrides map[string]string) []error {
	return applyKeyBindingOverrides(&APP_KEY_BINDINGS, overrides)
}

func applyKeyBindingOverrides(bindings *KeyBindings, overrides map[string]string) []error {
	var (
		fields = keyBindingFields(bindings)
		names  = make([]string, 0, len(overrides))
		errs   []error
	)
	for name := range overrides {
		names = append(names, name)
	}
	sort.Strings(names)

	// All are applied before any is checked, so two keys can be swapped.
	var undos = map[string]func(){}
	for _, name := range names {
		var field, ok = fields[name]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown key binding %q in the settings file", name))
			continue
		}

		var undo, err = setKeyBinding(field, overrides[name])
		if err != nil {
			errs = append(errs, fmt.Errorf("key binding %q in the settings file: %w", name, err))
			continue
		}
		undos[name] = undo
	}

	for _, name := range names {
		var undo, applied = undos[name]
		if !applied {
			continue
		}
		if other := conflictingBinding(fields, name); other != "" {
			undo()
			errs = append(errs, fmt.Errorf(
				"key binding %q in the settings file: %q is already used by %q", name, overrides[name], other,
			))
		}
	}

	return errs
}

// setKeyBinding parses the text for the kind of binding, sets it, and returns
// a function that puts the old value back.
func setKeyBinding(field keyBinding, text string) (func(), error) {
	switch {
	case field.rune != nil:
		var parsed, err = ParseRune(text)
		if err != nil {
			return nil, err
		}
		var old = *field.rune
		*field.rune = parsed
		return func() { *field.rune = old }, nil
	case field.key != nil:
		var parsed, err = ParseKey(text)
		if err != nil {
			return nil, err
		}
		var old = *field.key
		*field.key = parsed
		return func() { *field.key = old }, nil
	default:
		var parsed, err = ParseModKey(text)
		if err != nil {
			return nil, err
		}
		var old = *field.mod
		*field.mod = parsed
		return func() { *field.mod = old }, nil
	}
}

// conflictingBinding returns a binding that shares its key with the named one
// in a view where both are active, or "" if there is none.
func conflictingBinding(fields map[string]keyBinding, name string) string {
	var field = fields[name]
	for _, group := range keyBindingConflictGroups {
		if !slices.Contains(group, name) {
			continue
		}
		for _, other := range group {
			if other == name {
				continue
			}
			var o = fields[other]
			if (field.rune != nil && o.rune != nil && *field.rune == *o.rune) ||
				(field.key != nil && o.key != nil && *field.key == *o.key) {
				return other
			}
		}
	}
	return ""
}

// ParseRune reads a binding of one character, or "space".
func ParseRune(text string) (rune, error) {
	if strings.EqualFold(text, "space") {
		return ' ', nil
	}
	var r, size = utf8.DecodeRuneInString(text)
	if text == "" || size != len(text) || r == utf8.RuneError || unicode.IsControl(r) {
		return 0, fmt.Errorf("%q is not a single character", text)
	}
	return r, nil
}

// ParseKey reads a special key as FormatKey writes it, such as "Esc", "PgUp"
// or "Ctrl+X", ignoring case.
func ParseKey(text string) (tcell.Key, error) {
	var lower = strings.ToLower(strings.TrimSpace(text))

	if letter, ok := strings.CutPrefix(lower, "ctrl+"); ok {
		if len(letter) == 1 && letter[0] >= 'a' && letter[0] <= 'z' {
			return tcell.KeyCtrlA + tcell.Key(letter[0]-'a'), nil
		}
		return 0, fmt.Errorf("%q is not a key, Ctrl needs a letter such as Ctrl+X", text)
	}

	var named = map[string]tcell.Key{
		"esc":       tcell.KeyESC,
		"escape":    tcell.KeyESC,
		"enter":     tcell.KeyEnter,
		"tab":       tcell.KeyTab,
		"backspace": tcell.KeyBackspace2,
		"pgup":      tcell.KeyPgUp,
		"pgdn":      tcell.KeyPgDn,
		"up":        tcell.KeyUp,
		"down":      tcell.KeyDown,
		"left":      tcell.KeyLeft,
		"right":     tcell.KeyRight,
		"home":      tcell.KeyHome,
		"end":       tcell.KeyEnd,
	}
	if key, ok := named[lower]; ok {
		return key, nil
	}

	return 0, fmt.Errorf("%q is not a key, use a name such as Esc, PgUp or Ctrl+X", text)
}

// ParseModKey reads a modifier: Alt, Ctrl or Shift.
func ParseModKey(text string) (tcell.ModMask, error) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "alt":
		return tcell.ModAlt, nil
	case "ctrl":
		return tcell.ModCtrl, nil
	case "shift":
		return tcell.ModShift, nil
	}
	return 0, fmt.Errorf("%q is not a modifier, use Alt, Ctrl or Shift", text)
}
