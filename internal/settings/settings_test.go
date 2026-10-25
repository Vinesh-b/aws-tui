package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoad__MissingFileGivesEmptySettings(t *testing.T) {
	var loaded, err = Load(filepath.Join(t.TempDir(), "settings.json"))

	if err != nil || loaded != (Settings{}) {
		t.Errorf("Load = %+v, %v, want empty settings and no error", loaded, err)
	}
}

func TestSave__ThenLoadGivesTheSameSettings(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "settings.json")

	if err := Save(path, Settings{Theme: "Nord"}); err != nil {
		t.Fatal(err)
	}
	var loaded, err = Load(path)

	if err != nil || loaded.Theme != "Nord" {
		t.Errorf("Load = %+v, %v, want the Nord theme", loaded, err)
	}
}

func TestSave__CreatesMissingDirectories(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "a", "b", "settings.json")

	if err := Save(path, Settings{Theme: "Nord"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("settings file was not created: %v", err)
	}
}

func TestSave__ReplacesTheOldFileAndLeavesNoTemporaryFiles(t *testing.T) {
	var dir = t.TempDir()
	var path = filepath.Join(dir, "settings.json")
	Save(path, Settings{Theme: "Nord"})

	if err := Save(path, Settings{Theme: "Default"}); err != nil {
		t.Fatal(err)
	}

	var loaded, _ = Load(path)
	if loaded.Theme != "Default" {
		t.Errorf("theme = %q, want Default", loaded.Theme)
	}
	var entries, _ = os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("directory holds %d files, want only the settings file", len(entries))
	}
}

func TestSave__FileIsPrivateToTheUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not used on windows")
	}
	var path = filepath.Join(t.TempDir(), "settings.json")

	Save(path, Settings{Theme: "Nord"})

	var info, _ = os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestLoad__InvalidJsonGivesEmptySettingsAndAnError(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{not json"), 0o600)

	var loaded, err = Load(path)

	if err == nil || !strings.Contains(err.Error(), path) || loaded != (Settings{}) {
		t.Errorf("Load = %+v, %v, want empty settings and an error naming the file", loaded, err)
	}
}

func TestLoad__IgnoresFieldsItDoesNotKnow(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"theme": "Nord", "fromANewerVersion": true}`), 0o600)

	var loaded, err = Load(path)

	if err != nil || loaded.Theme != "Nord" {
		t.Errorf("Load = %+v, %v, want the Nord theme", loaded, err)
	}
}

func TestLoad__DirectoryInPlaceOfTheFileIsAnError(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("expected an error when the path is a directory")
	}
}

func TestDefaultPath__IsInsideTheConfigDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	t.Setenv("HOME", "/home/someone")
	t.Setenv("AppData", `C:\appdata`)

	var path, err = DefaultPath()

	if err != nil || !strings.HasSuffix(path, filepath.Join("aws-tui", "settings.json")) {
		t.Errorf("DefaultPath = %q, %v, want a path ending in aws-tui/settings.json", path, err)
	}
}
