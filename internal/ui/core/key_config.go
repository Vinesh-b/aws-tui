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
	seq  *string
}

// keyBindingConflictGroups lists the bindings that are active in the same view
// and so must not share a key. Bindings that are never active together are in
// no common group, so they may share one.
var keyBindingConflictGroups = [][]string{
	{
		"help", "reset", "find", "textCopy",
		"moveUp", "moveDown", "moveTop", "moveBottom", "mark", "removeItem",
		"commandLine",
	},
	{
		"help", "find", "nextSearch", "prevSearch", "moveUp", "moveDown",
		"moveLeft", "moveRight", "moveLineStart", "moveLineEnd", "moveTop",
		"moveBottom", "textCopy", "textViewWordRight", "textViewWordLeft",
		"textViewUndo", "textViewVisualMode", "textViewInsertMode",
		"commandLine",
	},
	{"viewFocusUp", "viewFocusDown", "viewFocusLeft", "viewFocusRight"},
	{"historyBack", "historyForward"},
}

func keyBindingFields(bindings *KeyBindings) map[string]keyBinding {
	return map[string]keyBinding{
		"help":               {rune: &bindings.Help},
		"commandLine":        {rune: &bindings.CommandLine},
		"escape":             {key: &bindings.Escape},
		"toggleServicesMenu": {key: &bindings.ToggleServicesMenu},
		"toggleServicePages": {key: &bindings.ToggleServicePages},
		"reset":              {rune: &bindings.Reset},
		"clearTable":         {key: &bindings.ClearTable},
		"done":               {key: &bindings.Done},
		"find":               {rune: &bindings.Find},
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
		"mark":               {rune: &bindings.Mark},
		"removeItem":         {rune: &bindings.RemoveItem},
		"textCopy":           {rune: &bindings.TextCopy},
		"toggleRawView":      {rune: &bindings.ToggleRawView},
		"textViewPageUp":     {key: &bindings.TextViewPageUp},
		"textViewPageDown":   {key: &bindings.TextViewPageDown},
		"textViewWordRight":  {rune: &bindings.TextViewWordRight},
		"textViewWordLeft":   {rune: &bindings.TextViewWordLeft},
		"textViewUndo":       {rune: &bindings.TextViewUndo},
		"textViewRedo":       {key: &bindings.TextViewRedo},
		"textViewVisualMode": {rune: &bindings.TextViewVisualMode},
		"textViewInsertMode": {rune: &bindings.TextViewInsertMode},

		"moveTopSequence":     {seq: &bindings.MoveTopSequence},
		"pageForwardSequence": {seq: &bindings.PageForwardSequence},
		"pageBackSequence":    {seq: &bindings.PageBackSequence},
		"openLinkSequence":    {seq: &bindings.OpenLinkSequence},
		"removeItemSequence":  {seq: &bindings.RemoveItemSequence},
	}
}

// ApplyKeyBindingOverrides sets APP_KEY_BINDINGS to the default keys with what
// the user chose in the settings file, keyed by binding name such as "reset"
// or "viewFocusUp". It must run before any view is built, as views list their
// keys in their help when they are created.
//
// An override that names no binding, cannot be read, or would give two
// bindings of one view the same key is skipped and returned as an error, so
// the caller can report it and the app still starts.
func ApplyKeyBindingOverrides(overrides map[string]string) []error {
	var bindings = defaultKeyBindings
	var errs = applyKeyBindingOverrides(&bindings, overrides)
	APP_KEY_BINDINGS = bindings
	for i, err := range errs {
		errs[i] = fmt.Errorf("in the settings file: %w", err)
	}
	return errs
}

// ChangeKeyBinding sets the named binding to the key, written as in the
// settings file, unless the key can't be read or another binding of its views
// has it. It is only used on the UI goroutine, and views built before take the
// key when they are built again.
func ChangeKeyBinding(name string, key string) error {
	var bindings = APP_KEY_BINDINGS
	if errs := applyKeyBindingOverrides(&bindings, map[string]string{name: key}); len(errs) > 0 {
		return errs[0]
	}
	APP_KEY_BINDINGS = bindings
	return nil
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
			errs = append(errs, fmt.Errorf("unknown key binding %q", name))
			continue
		}

		var undo, err = setKeyBinding(field, overrides[name])
		if err != nil {
			errs = append(errs, fmt.Errorf("key binding %q: %w", name, err))
			continue
		}
		undos[name] = undo
	}

	for _, name := range names {
		var undo, applied = undos[name]
		if !applied {
			continue
		}
		var other = conflictingBinding(fields, name)
		if other == "" {
			other = sequenceConflict(fields, name)
		}
		if other != "" {
			undo()
			errs = append(errs, fmt.Errorf(
				"key binding %q: %q is already used by %q", name, overrides[name], other,
			))
			continue
		}
		if done, other := turnedOffSequenceConflict(fields, name); other != "" {
			undo()
			errs = append(errs, fmt.Errorf(
				"key binding %q: turning it off leaves %q unreachable, as %q starts with its key",
				name, done, other,
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
	case field.seq != nil:
		var parsed, err = ParseKeySequence(text)
		if err != nil {
			return nil, err
		}
		var old = *field.seq
		*field.seq = parsed
		return func() { *field.seq = old }, nil
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
		if letter == "space" {
			return tcell.KeyCtrlSpace, nil
		}
		if len(letter) == 1 && letter[0] >= 'a' && letter[0] <= 'z' {
			return tcell.KeyCtrlA + tcell.Key(letter[0]-'a'), nil
		}
		return 0, fmt.Errorf("%q is not a key, Ctrl needs a letter or Space such as Ctrl+X", text)
	}

	var named = map[string]tcell.Key{
		"esc":       tcell.KeyESC,
		"escape":    tcell.KeyESC,
		"enter":     tcell.KeyEnter,
		"tab":       tcell.KeyTab,
		"backspace": tcell.KeyBackspace,
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

// sequenceRunes are the rune bindings each sequence does, which are only
// reached through the sequence while it is set, so they may share its first key
var sequenceRunes = map[string]string{
	"moveTopSequence":    "moveTop",
	"removeItemSequence": "removeItem",
}

// modifiedRunes are the rune bindings only used with a modifier, which never
// reach the sequences
var modifiedRunes = []string{"viewResizeReset"}

// sequenceConflict returns a binding the named one can't be told apart from
// because of the sequences, or "" if there is none. A sequence clashes with a
// sequence that starts it, and with a key that is its first key, as that key
// would always wait for the rest of the sequence.
func sequenceConflict(fields map[string]keyBinding, name string) string {
	var field = fields[name]
	if field.seq == nil && (field.rune == nil || slices.Contains(modifiedRunes, name)) {
		return ""
	}

	var names = make([]string, 0, len(fields))
	for other := range fields {
		names = append(names, other)
	}
	sort.Strings(names)

	for _, other := range names {
		if other == name {
			continue
		}
		var o = fields[other]
		switch {
		case field.seq != nil && o.seq != nil:
			if startsWith(*field.seq, *o.seq) || startsWith(*o.seq, *field.seq) {
				return other
			}
		case field.seq != nil && o.rune != nil:
			if runeStartsSequence(fields, other, *field.seq) {
				return other
			}
		case field.rune != nil && o.seq != nil:
			if runeStartsSequence(fields, name, *o.seq) {
				return other
			}
		}
	}
	return ""
}

// startsWith is true when the sequence is set and starts the other one
func startsWith(sequence string, other string) bool {
	return len(sequence) > 0 && len(other) > 0 && strings.HasPrefix(other, sequence)
}

// runeStartsSequence is true when the rune binding is the first key of the
// sequence and can't be reached because of it
func runeStartsSequence(fields map[string]keyBinding, runeName string, sequence string) bool {
	if len(sequence) == 0 || slices.Contains(modifiedRunes, runeName) {
		return false
	}
	var first, _ = utf8.DecodeRuneInString(sequence)
	if *fields[runeName].rune != first {
		return false
	}
	// A key a sequence does is reached through it while that sequence is set
	for seqName, done := range sequenceRunes {
		if done == runeName && len(*fields[seqName].seq) > 0 {
			return false
		}
	}
	return true
}

// ParseKeySequence reads a sequence of two or more keys such as "gg", or ""
// to turn the sequence off.
func ParseKeySequence(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	if utf8.RuneCountInString(text) < 2 {
		return "", fmt.Errorf("%q is not a sequence, use two or more keys such as gg, or nothing to turn it off", text)
	}
	for _, r := range text {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", fmt.Errorf("%q is not a sequence, it can only have keys that type a character", text)
		}
	}
	return text, nil
}

// turnedOffSequenceConflict returns the rune binding a sequence that is off
// does, and a sequence that starts with that rune, which then can't be
// reached. Both are "" when there is no such clash.
func turnedOffSequenceConflict(fields map[string]keyBinding, name string) (string, string) {
	var field = fields[name]
	var done, ok = sequenceRunes[name]
	if field.seq == nil || len(*field.seq) > 0 || !ok {
		return "", ""
	}
	return done, sequenceConflict(fields, done)
}
