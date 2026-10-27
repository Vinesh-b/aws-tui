package core

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestApplyKeyBindingOverrides__ChangesRuneKeyAndModifierBindings(t *testing.T) {
	var bindings = APP_KEY_BINDINGS

	var errs = applyKeyBindingOverrides(&bindings, map[string]string{
		"reset":            "R",
		"clearTable":       "ctrl+d",
		"saveTable":        "space",
		"pageChangeModKey": "Ctrl",
	})

	if len(errs) != 0 {
		t.Fatalf("errors = %v", errs)
	}
	if bindings.Reset != 'R' || bindings.ClearTable != tcell.KeyCtrlD ||
		bindings.SaveTable != ' ' || bindings.PageChangeModKey != tcell.ModCtrl {
		t.Errorf("bindings were not changed: %+v", bindings)
	}
}

func TestApplyKeyBindingOverrides__SkipsWhatCannotBeUsed(t *testing.T) {
	var tests = map[string]struct {
		name, value, wantErr string
	}{
		"unknown name":      {"nope", "x", "unknown key binding"},
		"two characters":    {"reset", "rr", "single character"},
		"empty":             {"reset", "", "single character"},
		"bad key":           {"clearTable", "banana", "not a key"},
		"bad modifier":      {"viewResizeModKey", "meta", "not a modifier"},
		"table conflict":    {"reset", "d", `already used by "saveTable"`},
		"focus conflict":    {"viewFocusUp", "Ctrl+J", `already used by "viewFocusDown"`},
		"text view clash":   {"textViewWordLeft", "w", `already used by "textViewWordRight"`},
		"one key sequence":  {"moveTopSequence", "g", "not a sequence"},
		"space in sequence": {"moveTopSequence", "g g", "not a sequence"},
		"same sequence":     {"openLinkSequence", "gt", `already used by "pageForwardSequence"`},
		"sequence prefix":   {"openLinkSequence", "g", "not a sequence"},
		"longer sequence":   {"openLinkSequence", "ggx", `already used by "moveTopSequence"`},
		"key starts one":    {"tableScan", "g", `already used by "moveTopSequence"`},
		"strands its key":   {"moveTopSequence", "", `leaves "moveTop" unreachable`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var bindings = APP_KEY_BINDINGS

			var errs = applyKeyBindingOverrides(&bindings, map[string]string{tt.name: tt.value})

			if len(errs) != 1 || !strings.Contains(errs[0].Error(), tt.wantErr) {
				t.Fatalf("errors = %v, want one containing %q", errs, tt.wantErr)
			}
			if bindings != APP_KEY_BINDINGS {
				t.Errorf("a skipped override changed the bindings: %+v", bindings)
			}
		})
	}
}

func TestApplyKeyBindingOverrides__KeepsTheValidOnesAlongsideBadOnes(t *testing.T) {
	var bindings = APP_KEY_BINDINGS

	var errs = applyKeyBindingOverrides(&bindings, map[string]string{"reset": "R", "nope": "x"})

	if len(errs) != 1 || bindings.Reset != 'R' {
		t.Errorf("errors = %v, reset = %q", errs, bindings.Reset)
	}
}

func TestApplyKeyBindingOverrides__AllowsSwappingTwoKeys(t *testing.T) {
	var bindings = APP_KEY_BINDINGS

	var errs = applyKeyBindingOverrides(&bindings, map[string]string{"reset": "d", "saveTable": "r"})

	if len(errs) != 0 || bindings.Reset != 'd' || bindings.SaveTable != 'r' {
		t.Errorf("errors = %v, reset = %q, saveTable = %q", errs, bindings.Reset, bindings.SaveTable)
	}
}

func TestApplyKeyBindingOverrides__ReportsBothSidesOfAClash(t *testing.T) {
	var bindings = APP_KEY_BINDINGS

	var errs = applyKeyBindingOverrides(&bindings, map[string]string{"reset": "x", "saveTable": "x"})

	if len(errs) == 0 || bindings.Reset == bindings.SaveTable {
		t.Errorf("errors = %v, reset = %q, saveTable = %q", errs, bindings.Reset, bindings.SaveTable)
	}
}

func TestDefaultKeyBindings__HaveNoConflicts(t *testing.T) {
	var bindings = APP_KEY_BINDINGS
	var fields = keyBindingFields(&bindings)

	for _, group := range keyBindingConflictGroups {
		for _, name := range group {
			if _, ok := fields[name]; !ok {
				t.Errorf("conflict group names %q, which is not a binding", name)
			}
			if other := conflictingBinding(fields, name); other != "" {
				t.Errorf("default %q shares its key with %q", name, other)
			}
		}
	}
	for name := range fields {
		if other := sequenceConflict(fields, name); other != "" {
			t.Errorf("default %q can't be told apart from %q", name, other)
		}
	}
}

func TestApplyKeyBindingOverrides__SequencesCanMoveOrBeTurnedOff(t *testing.T) {
	var bindings = APP_KEY_BINDINGS

	var errs = applyKeyBindingOverrides(&bindings, map[string]string{
		"moveTopSequence":  "zz",
		"openLinkSequence": "",
	})

	if len(errs) != 0 || bindings.MoveTopSequence != "zz" || bindings.OpenLinkSequence != "" {
		t.Errorf("errors = %v, moveTop = %q, openLink = %q", errs, bindings.MoveTopSequence, bindings.OpenLinkSequence)
	}
}

func TestParseKey__ReadsWhatFormatKeyWrites(t *testing.T) {
	for _, key := range []tcell.Key{
		tcell.KeyESC, tcell.KeyEnter, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyUp,
		tcell.KeyCtrlX, tcell.KeyCtrlA, tcell.KeyCtrlZ, tcell.KeyCtrlSpace,
		tcell.KeyBackspace,
	} {
		if got, err := ParseKey(FormatKey(key)); err != nil || got != key {
			t.Errorf("ParseKey(%q) = %v, %v; want %v", FormatKey(key), got, err, key)
		}
	}
}
