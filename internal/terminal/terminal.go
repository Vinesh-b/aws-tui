// Package terminal opens a program in a new terminal window.
package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// environment is what the terminal to use is picked from, which tests replace
type environment struct {
	goos     string
	getenv   func(key string) string
	lookPath func(file string) (string, error)
}

// emulator is a terminal that can run a program in a new window
type emulator struct {
	name string
	// hint is true if the terminal the app runs in is this one, which is then
	// preferred to the others
	hint func(env environment) bool
	// args are the arguments to run argv in dir
	args func(dir string, argv []string) []string
}

func hasEnv(key string) func(env environment) bool {
	return func(env environment) bool { return env.getenv(key) != "" }
}

func never(environment) bool { return false }

// emulators are in the order they are tried when none is the terminal in use
var emulators = []emulator{
	{"tmux", hasEnv("TMUX"), func(dir string, argv []string) []string {
		return []string{"new-window", "-c", dir, shellLine(argv)}
	}},
	{"kitty", hasEnv("KITTY_WINDOW_ID"), func(dir string, argv []string) []string {
		return append([]string{"--directory", dir}, argv...)
	}},
	{"wezterm", hasEnv("WEZTERM_PANE"), func(dir string, argv []string) []string {
		return append([]string{"start", "--cwd", dir, "--"}, argv...)
	}},
	{"alacritty", hasEnv("ALACRITTY_LOG"), func(dir string, argv []string) []string {
		return append([]string{"--working-directory", dir, "-e"}, argv...)
	}},
	{"konsole", hasEnv("KONSOLE_VERSION"), func(dir string, argv []string) []string {
		return append([]string{"--workdir", dir, "-e"}, argv...)
	}},
	{"gnome-terminal", hasEnv("GNOME_TERMINAL_SERVICE"), func(dir string, argv []string) []string {
		return append([]string{"--working-directory=" + dir, "--"}, argv...)
	}},
	{"foot", func(env environment) bool { return strings.HasPrefix(env.getenv("TERM"), "foot") },
		func(dir string, argv []string) []string {
			return append([]string{"--working-directory", dir}, argv...)
		}},
	{"x-terminal-emulator", never, func(_ string, argv []string) []string {
		return append([]string{"-e"}, argv...)
	}},
	{"xterm", never, func(_ string, argv []string) []string {
		return append([]string{"-e"}, argv...)
	}},
}

// Open runs the program and its arguments in a new terminal window, in the
// working directory, and returns once the window is being opened. The window
// is not tied to this process, so it stays when the app quits.
func Open(argv []string) error {
	var dir, err = os.Getwd()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	cmd, err = command(environment{runtime.GOOS, os.Getenv, exec.LookPath}, argv, dir)
	if err != nil {
		return err
	}

	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("can't open a terminal: %w", err)
	}
	// The terminal is not waited on, but its exit has to be collected
	go func() { _ = cmd.Wait() }()
	return nil
}

// command is the command that opens a terminal running argv in dir
func command(env environment, argv []string, dir string) (*exec.Cmd, error) {
	switch env.goos {
	case "darwin":
		return macCommand(env, argv, dir)
	case "windows":
		return nil, fmt.Errorf("opening a terminal is not supported on windows")
	}

	// $TERMINAL is the terminal the user chose, which only takes -e
	if name := env.getenv("TERMINAL"); name != "" {
		if path, err := env.lookPath(name); err == nil {
			return exec.Command(path, append([]string{"-e"}, argv...)...), nil
		}
	}

	var candidates []emulator
	for _, term := range emulators {
		if term.hint(env) {
			candidates = append(candidates, term)
		}
	}
	candidates = append(candidates, emulators...)

	for _, term := range candidates {
		if path, err := env.lookPath(term.name); err == nil {
			return exec.Command(path, term.args(dir, argv)...), nil
		}
	}
	return nil, fmt.Errorf("no terminal found to open, set $TERMINAL to one")
}

// macCommand opens a window of Terminal, or of iTerm if that is in use
func macCommand(env environment, argv []string, dir string) (*exec.Cmd, error) {
	var line = "cd " + shellQuote(dir) + " && " + shellLine(argv)
	var script string
	if env.getenv("TERM_PROGRAM") == "iTerm.app" {
		script = `tell application "iTerm" to create window with default profile command "` +
			escapeAppleScript("sh -c "+shellQuote(line)) + `"`
	} else {
		script = `tell application "Terminal"` + "\n" +
			`do script "` + escapeAppleScript(line) + `"` + "\n" +
			`activate` + "\n" + `end tell`
	}

	var path, err = env.lookPath("osascript")
	if err != nil {
		return nil, fmt.Errorf("osascript is needed to open a terminal: %w", err)
	}
	return exec.Command(path, "-e", script), nil
}

func escapeAppleScript(text string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text)
}

// shellLine is argv as one line for a shell
func shellLine(argv []string) string {
	var quoted = make([]string, len(argv))
	for i, arg := range argv {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}
