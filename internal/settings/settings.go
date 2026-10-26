// Package settings reads and writes the settings the user has chosen in the app,
// which are kept in a file between runs.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	appDirName   = "aws-tui"
	fileName     = "settings.json"
	dirPerm      = 0o700
	filePerm     = 0o600
	tempFilePatt = ".settings-*.tmp"
)

// Settings are the choices of the user. A field that is empty has not been
// chosen, and the app uses its default.
type Settings struct {
	Theme string `json:"theme,omitempty"`
	// DownloadDir is the directory proposed when saving a table to a file. It
	// is not set in the app, but by editing the settings file.
	DownloadDir string `json:"downloadDir,omitempty"`
	// KeyBindings replaces default keys, by binding name such as "reset". Like
	// DownloadDir it is set by editing the settings file.
	KeyBindings map[string]string `json:"keyBindings,omitempty"`
}

// DefaultPath is where the settings file is kept, in the config directory of the
// user: ~/.config/aws-tui/settings.json on Linux.
func DefaultPath() (string, error) {
	var dir, err = os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the config directory: %w", err)
	}

	return filepath.Join(dir, appDirName, fileName), nil
}

// Load reads the settings file. A file that does not exist gives empty settings
// and no error, as the app has not been configured yet. A file that cannot be
// read or understood gives empty settings and the error, so the caller can
// report it and carry on with the defaults.
func Load(path string) (Settings, error) {
	var loaded Settings
	if err := readJSON(path, "settings", &loaded); err != nil {
		return Settings{}, err
	}

	return loaded, nil
}

// Save writes the settings file, creating its directory if needed. The file is
// written to a temporary file and moved into place, so a crash cannot leave a
// half written file behind.
func Save(path string, settings Settings) error {
	return writeJSON(path, "settings", settings)
}

// readJSON reads a file into the value, which is left as it is if there is no
// file. What the file holds is called what in errors.
func readJSON(path string, what string, value any) error {
	var data, err = os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", what, err)
	}

	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("cannot read %s from %s: %w", what, path, err)
	}

	return nil
}

func writeJSON(path string, what string, value any) error {
	var data, err = json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	var dir = filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, err)
	}

	var temp *os.File
	if temp, err = os.CreateTemp(dir, tempFilePatt); err != nil {
		return fmt.Errorf("cannot save %s: %w", what, err)
	}
	var tempPath = temp.Name()
	defer os.Remove(tempPath)

	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("cannot save %s: %w", what, err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("cannot save %s: %w", what, err)
	}
	if err = os.Chmod(tempPath, filePerm); err != nil {
		return fmt.Errorf("cannot save %s: %w", what, err)
	}
	if err = os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("cannot save %s: %w", what, err)
	}

	return nil
}
