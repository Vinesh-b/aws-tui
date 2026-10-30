package core

import (
	"strings"
	"testing"
)

func TestKeyBindingInfos__ListEveryBindingOnce(t *testing.T) {
	var bindings = APP_KEY_BINDINGS
	var fields = keyBindingFields(&bindings)
	var listed = map[string]int{}
	for _, info := range KeyBindingInfos() {
		listed[info.Name]++
		if info.Group == "" || info.Description == "" {
			t.Errorf("%s has no group or description", info.Name)
		}
	}

	for name := range fields {
		if listed[name] != 1 {
			t.Errorf("%s is listed %d times, want once", name, listed[name])
		}
	}
	for name := range listed {
		if _, ok := fields[name]; !ok {
			t.Errorf("%s is listed but is not a binding", name)
		}
	}
}

func TestDefaultKeyBinding__IsWrittenAsTheSettingsFileReadsIt(t *testing.T) {
	for _, info := range KeyBindingInfos() {
		var bindings = defaultKeyBindings
		var written = DefaultKeyBinding(info.Name)

		if errs := applyKeyBindingOverrides(&bindings, map[string]string{info.Name: written}); len(errs) != 0 {
			t.Errorf("%s: %q can't be read back: %v", info.Name, written, errs)
		}
		if bindings != defaultKeyBindings {
			t.Errorf("%s: reading %q back changed the bindings", info.Name, written)
		}
	}
}

func TestKeyBindingHint__TellsWhatTheKindOfKeyTakes(t *testing.T) {
	var tests = map[string]string{
		"reset":           "One character",
		"clearTable":      "Ctrl+X",
		"historyModKey":   "Alt, Ctrl or Shift",
		"moveTopSequence": "Two or more characters",
	}
	for name, want := range tests {
		if got := KeyBindingHint(name); !strings.Contains(got, want) {
			t.Errorf("hint of %s = %q, want it to mention %q", name, got, want)
		}
	}
}

func TestChangeKeyBinding__SetsTheKey(t *testing.T) {
	keepBindings(t)

	if err := ChangeKeyBinding("reset", "R"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if APP_KEY_BINDINGS.Reset != 'R' || CurrentKeyBinding("reset") != "R" {
		t.Errorf("reset = %q", APP_KEY_BINDINGS.Reset)
	}
	if DefaultKeyBinding("reset") != "r" {
		t.Errorf("default of reset = %q, want r", DefaultKeyBinding("reset"))
	}
}

func TestChangeKeyBinding__KeepsTheKeysWhenTheNewOneCantBeUsed(t *testing.T) {
	keepBindings(t)
	var before = APP_KEY_BINDINGS

	var errs = []error{
		ChangeKeyBinding("reset", "y"),
		ChangeKeyBinding("reset", "rr"),
		ChangeKeyBinding("nope", "x"),
	}

	for i, err := range errs {
		if err == nil {
			t.Errorf("change %d was taken", i)
		}
	}
	if !strings.Contains(errs[0].Error(), `already used by "textCopy"`) {
		t.Errorf("error = %v", errs[0])
	}
	if APP_KEY_BINDINGS != before {
		t.Error("a change that failed changed the bindings")
	}
}

func TestApplyKeyBindingOverrides__StartsFromTheDefaults(t *testing.T) {
	keepBindings(t)
	APP_KEY_BINDINGS.Find = '?'

	var errs = ApplyKeyBindingOverrides(map[string]string{"reset": "R", "nope": "x"})

	if APP_KEY_BINDINGS.Find != '/' || APP_KEY_BINDINGS.Reset != 'R' {
		t.Errorf("find = %q, reset = %q, want the default find and the saved reset", APP_KEY_BINDINGS.Find, APP_KEY_BINDINGS.Reset)
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "in the settings file") {
		t.Errorf("errors = %v, want the unknown binding reported as in the settings file", errs)
	}
}

func TestKeyBindingInfos__ListGeneralKeysBeforeTheKeysOfEachService(t *testing.T) {
	var seenService = false
	for _, info := range KeyBindingInfos() {
		if info.Service != "" {
			seenService = true
		} else if seenService {
			t.Errorf("general key %s is listed after the keys of a service", info.Name)
		}
	}

	var services = map[string]string{"tableScan": "DynamoDB", "download": "S3", "follow": "CloudWatch"}
	for _, info := range KeyBindingInfos() {
		if want, ok := services[info.Name]; ok && info.Section() != want {
			t.Errorf("%s is in %q, want %q", info.Name, info.Section(), want)
		}
	}
	if got := KeyBindingInfos()[0].Section(); got != "General: Moving around" {
		t.Errorf("first section = %q", got)
	}
}
