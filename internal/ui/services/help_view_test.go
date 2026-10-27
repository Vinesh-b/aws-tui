package services

import (
	"strings"
	"testing"

	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

// helpLine is the line of the help text that tells what the action does
func helpLine(t *testing.T, text string, action string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.HasSuffix(line, action) {
			return line
		}
	}
	t.Fatalf("the help has no line to %q:\n%s", action, text)
	return ""
}

func TestHelpText__ListsTheDefaultKeys(t *testing.T) {
	var text = helpText()

	if line := helpLine(t, text, "Show or hide the services menu"); !strings.Contains(line, "Ctrl+Space") {
		t.Errorf("line = %q, want Ctrl+Space", line)
	}
	if line := helpLine(t, text, "Go to the page on the left / right"); !strings.Contains(line, "gT / gt") {
		t.Errorf("line = %q, want gT / gt", line)
	}
	if line := helpLine(t, text, "Go back / forward through the pages shown"); !strings.Contains(line, "Alt+Left / Alt+Right") {
		t.Errorf("line = %q, want Alt+Left / Alt+Right", line)
	}
}

func TestHelpText__FollowsTheKeyBindings(t *testing.T) {
	var original = core.APP_KEY_BINDINGS
	t.Cleanup(func() { core.APP_KEY_BINDINGS = original })
	core.APP_KEY_BINDINGS.Reset = 'R'
	core.APP_KEY_BINDINGS.ToggleServicesMenu = tcell.KeyCtrlB

	var text = helpText()

	if line := helpLine(t, text, "Load the table again"); !strings.HasPrefix(strings.TrimSpace(line), "R ") {
		t.Errorf("line = %q, want R", line)
	}
	if line := helpLine(t, text, "Show or hide the services menu"); !strings.Contains(line, "Ctrl+B") {
		t.Errorf("line = %q, want Ctrl+B", line)
	}
}
