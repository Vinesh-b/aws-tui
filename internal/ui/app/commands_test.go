package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/rivo/tview"
)

// fakeLine keeps what the commands showed, which saving shows from the UI
// goroutine
type fakeLine struct {
	mu       sync.Mutex
	messages []string
	errors   []string
}

func (f *fakeLine) ShowMessage(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, text)
}

func (f *fakeLine) ShowError(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errors = append(f.errors, text)
}

func (f *fakeLine) shown() (messages []string, errors []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.messages), slices.Clone(f.errors)
}

type commandsTest struct {
	commands *commands
	line     *fakeLine
	shown    []string
	quits    int
}

func newCommandsTest(appCtx *core.AppContext) *commandsTest {
	var test = &commandsTest{line: &fakeLine{}}
	var items []*ServiceItem
	for _, name := range []core.ViewId{
		core.LAMBDA, core.CLOUDWATCH_LOGS, core.CLOUDWATCH_ALARMS, core.DYNAMODB, core.DEBUG_LOGS, core.HELP,
	} {
		items = append(items, staticServiceItem("󰘧 "+string(name), "", 'x', &countingPage{Box: tview.NewBox()}))
	}
	test.commands = &commands{
		items:  items,
		appCtx: appCtx,
		line:   test.line,
		show:   func(item *ServiceItem) { test.shown = append(test.shown, serviceName(item)) },
		quit:   func() { test.quits++ },
	}
	return test
}

func TestCommands__GoToTheServiceTheyName(t *testing.T) {
	var tests = map[string]string{
		"lambda":         "Lambda",
		"cloudwatchlogs": "CloudWatch Logs",
		"logs":           "CloudWatch Logs",
		"LOGS":           "CloudWatch Logs",
		"ddb":            "DynamoDB",
		"help":           "Help",
		"dyn":            "DynamoDB",
		"cloudwatcha":    "CloudWatch Alarms",
	}
	for command, want := range tests {
		t.Run(command, func(t *testing.T) {
			var test = newCommandsTest(uitest.NewAppContext())

			test.commands.Run(command, nil)

			if !slices.Equal(test.shown, []string{want}) {
				t.Errorf("shown = %v, want %s", test.shown, want)
			}
		})
	}
}

func TestCommands__ReportWhatTheyCannotGoTo(t *testing.T) {
	var tests = map[string]string{
		"nope": "not a command or service: nope",
		"d":    "d could be DynamoDB, Debug Logs",
	}
	for command, want := range tests {
		t.Run(command, func(t *testing.T) {
			var test = newCommandsTest(uitest.NewAppContext())

			test.commands.Run(command, nil)

			if _, errors := test.line.shown(); !slices.Equal(errors, []string{want}) || len(test.shown) != 0 {
				t.Errorf("errors = %v, shown = %v, want %q", errors, test.shown, want)
			}
		})
	}
}

func TestCommands__QQuits(t *testing.T) {
	for _, command := range []string{"q", "quit", " q "} {
		var test = newCommandsTest(uitest.NewAppContext())

		test.commands.Run(command, nil)

		if test.quits != 1 {
			t.Errorf("%q quit %d times, want once", command, test.quits)
		}
	}
}

func TestCommands__CompleteWhatIsTyped(t *testing.T) {
	var test = newCommandsTest(uitest.NewAppContext())

	if got := test.commands.Complete("cloudwatch"); !slices.Equal(got, []string{"cloudwatchlogs", "cloudwatchalarms"}) {
		t.Errorf("cloudwatch completes to %v", got)
	}
	if got := test.commands.Complete("q"); !slices.Equal(got, []string{"quit"}) {
		t.Errorf("q completes to %v", got)
	}
	for _, text := range []string{"", "w out.csv", "nope"} {
		if got := test.commands.Complete(text); len(got) != 0 {
			t.Errorf("%q completes to %v, want nothing", text, got)
		}
	}
}

func TestCommands__WSavesTheTableThatHasTheFocus(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var test = newCommandsTest(appCtx)
	var table = core.NewSelectableTable[string]("Functions", core.TableRow{"Name"}, appCtx)
	table.SetData([]core.TableRow{{"orders"}}, nil, 0)
	uitest.RunApp(t, appCtx.App)
	uitest.OnUI(t, appCtx.App, func() { appCtx.App.SetFocus(table.GetTable()) })
	var path = filepath.Join(t.TempDir(), "functions.csv")

	uitest.OnUI(t, appCtx.App, func() { test.commands.Run("w "+path, table.GetTable()) })

	uitest.Eventually(t, appCtx.App, func() bool {
		var messages, _ = test.line.shown()
		return slices.Equal(messages, []string{"Saved to " + path})
	}, "the save is reported")
	var written, _ = os.ReadFile(path)
	if !strings.Contains(string(written), "orders") {
		t.Errorf("file = %q, want the table rows", written)
	}
}

func TestCommands__WWithoutATableIsAnError(t *testing.T) {
	var test = newCommandsTest(uitest.NewAppContext())

	test.commands.Run("w", tview.NewBox())

	if _, errors := test.line.shown(); !slices.Equal(errors, []string{"the selected pane is not a table"}) {
		t.Errorf("errors = %v", errors)
	}
}
