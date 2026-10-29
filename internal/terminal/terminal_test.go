package terminal

import (
	"fmt"
	"slices"
	"testing"
)

// testEnv has the programs installed and the variables set
func testEnv(goos string, vars map[string]string, installed ...string) environment {
	return environment{
		goos:   goos,
		getenv: func(key string) string { return vars[key] },
		lookPath: func(file string) (string, error) {
			if slices.Contains(installed, file) {
				return "/bin/" + file, nil
			}
			return "", fmt.Errorf("%s not found", file)
		},
	}
}

func TestCommand(t *testing.T) {
	var argv = []string{"/opt/aws-tui", "-open", "logs streams"}

	var tests = []struct {
		name      string
		env       environment
		want      []string
		wantError bool
	}{
		{
			name: "the terminal in use is preferred to the first one installed",
			env:  testEnv("linux", map[string]string{"KITTY_WINDOW_ID": "1"}, "gnome-terminal", "kitty"),
			want: []string{"/bin/kitty", "--directory", "/work", "/opt/aws-tui", "-open", "logs streams"},
		},
		{
			name: "tmux opens a window with the program as one quoted line",
			env:  testEnv("linux", map[string]string{"TMUX": "/tmp/tmux-1/default,1,0"}, "tmux", "xterm"),
			want: []string{"/bin/tmux", "new-window", "-c", "/work", "'/opt/aws-tui' '-open' 'logs streams'"},
		},
		{
			name: "the terminal in use is skipped when it is not installed",
			env:  testEnv("linux", map[string]string{"KITTY_WINDOW_ID": "1"}, "xterm"),
			want: []string{"/bin/xterm", "-e", "/opt/aws-tui", "-open", "logs streams"},
		},
		{
			name: "TERMINAL is used before the others",
			env:  testEnv("linux", map[string]string{"TERMINAL": "st"}, "st", "xterm"),
			want: []string{"/bin/st", "-e", "/opt/aws-tui", "-open", "logs streams"},
		},
		{
			name: "gnome terminal gets the working directory",
			env:  testEnv("linux", nil, "gnome-terminal"),
			want: []string{
				"/bin/gnome-terminal", "--working-directory=/work", "--", "/opt/aws-tui", "-open", "logs streams",
			},
		},
		{
			name:      "no terminal installed",
			env:       testEnv("linux", nil),
			wantError: true,
		},
		{
			name:      "windows is not supported",
			env:       testEnv("windows", nil, "xterm"),
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cmd, err = command(test.env, argv, "/work")
			if test.wantError {
				if err == nil {
					t.Fatalf("expected an error, got %v", cmd.Args)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(cmd.Args, test.want) {
				t.Errorf("got %q, want %q", cmd.Args, test.want)
			}
		})
	}
}

func TestCommandOnMac(t *testing.T) {
	var argv = []string{"/opt/aws-tui", "-open", "it's"}

	var cmd, err = command(testEnv("darwin", nil, "osascript"), argv, "/work dir")
	if err != nil {
		t.Fatal(err)
	}
	var want = `tell application "Terminal"` + "\n" +
		`do script "cd '/work dir' && '/opt/aws-tui' '-open' 'it'\\''s'"` + "\n" +
		"activate\nend tell"
	if got := cmd.Args[len(cmd.Args)-1]; got != want {
		t.Errorf("got script %q, want %q", got, want)
	}

	cmd, err = command(testEnv("darwin", map[string]string{"TERM_PROGRAM": "iTerm.app"}, "osascript"), argv, "/work")
	if err != nil {
		t.Fatal(err)
	}
	if got := cmd.Args[len(cmd.Args)-1]; got == "" || got[:len(`tell application "iTerm"`)] != `tell application "iTerm"` {
		t.Errorf("expected a script for iTerm, got %q", got)
	}

	if _, err := command(testEnv("darwin", nil), argv, "/work"); err == nil {
		t.Error("expected an error without osascript")
	}
}
