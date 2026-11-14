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
		var page core.ServicePage = &countingPage{Box: tview.NewBox()}
		if name == core.CLOUDWATCH_LOGS {
			page = newPagedPage("logs", "Groups", "Streams", "Events", "Insights")
		}
		items = append(items, withAliases(staticServiceItem("󰘧 "+string(name), "", 'x', page), aliasesOf(name)...))
	}
	test.commands = &commands{
		items:  items,
		appCtx: appCtx,
		line:   test.line,
		show: func(item *ServiceItem, page string) {
			test.shown = append(test.shown, strings.TrimSpace(serviceName(item)+" "+page))
		},
		quit: func() { test.quits++ },
	}
	return test
}

func TestCommands__GoToThePageOfTheServiceTheyName(t *testing.T) {
	var tests = map[string]string{
		"cloudwatchlogs streams": "CloudWatch Logs Streams",
		"logs Streams":           "CloudWatch Logs Streams",
		"logs  ev":               "CloudWatch Logs Events",
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

func TestCommands__ReportPagesTheyCannotGoTo(t *testing.T) {
	var tests = map[string]string{
		"logs nope":   "CloudWatch Logs has no page nope",
		"lambda tabs": "Lambda has no pages",
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

func TestCommands__CompleteThePagesOfTheService(t *testing.T) {
	var test = newCommandsTest(uitest.NewAppContext())

	var tests = map[string][]string{
		"logs ":            {"logs groups", "logs streams", "logs events", "logs insights"},
		"cloudwatchlogs s": {"cloudwatchlogs streams"},
		"LOGS e":           {"logs events"},
		"logs streams":     nil,
		"logs x":           nil,
		"lambda ":          nil,
		"nope ":            nil,
		"w out.csv":        nil,
	}
	for text, want := range tests {
		if got := test.commands.Complete(text); !slices.Equal(got, want) {
			t.Errorf("%q completes to %v, want %v", text, got, want)
		}
	}
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

// aliasesOf are the short commands the service is listed with in the app
func aliasesOf(id core.ViewId) []string {
	for _, service := range awsServices {
		if service.Id == id {
			return service.Aliases
		}
	}
	return nil
}

// A service is added to the app with what its package gives, so what it gives
// is checked here for every service.
func TestAwsServices__EachCanBeListedAndBuilt(t *testing.T) {
	var ids = map[core.ViewId]bool{}
	for _, service := range awsServices {
		if service.Id == "" || service.Icon == "" || service.Description == "" || service.Build == nil {
			t.Errorf("service %q is missing its name, icon, description or build func: %+v", service.Id, service)
		}
		if ids[service.Id] {
			t.Errorf("service %q is listed twice", service.Id)
		}
		ids[service.Id] = true
	}
}

func TestAwsServices__CommandsNameOnlyOneService(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var owners = map[string]core.ViewId{}
	for _, service := range awsServices {
		for _, command := range serviceCommands(awsServiceItem(service, appCtx)) {
			if owner, taken := owners[command]; taken {
				t.Errorf("command %q goes to both %s and %s", command, owner, service.Id)
			}
			owners[command] = service.Id
		}
	}
}

func TestAwsServices__KeepTheirShortCommands(t *testing.T) {
	var want = map[string]core.ViewId{
		"logs":    core.CLOUDWATCH_LOGS,
		"alarms":  core.CLOUDWATCH_ALARMS,
		"metrics": core.CLOUDWATCH_METRICS,
		"cfn":     core.CLOUDFORMATION,
		"ddb":     core.DYNAMODB,
		"s3":      core.S3BUCKETS,
		"secrets": core.SECRETS_MANAGER,
		"sfn":     core.STATE_MACHINES,
		"ssm":     core.SYSTEMS_MANAGER,
	}
	for alias, id := range want {
		if !slices.Contains(aliasesOf(id), alias) {
			t.Errorf("%s has the short commands %v, want %s among them", id, aliasesOf(id), alias)
		}
	}
}

// The item of a service is built with the app context, and is one that is
// built again when the api clients change
func TestAwsServiceItem__IsNamedAndBuiltFromTheService(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = &countingPage{Box: tview.NewBox()}
	var builtWith *core.AppContext
	// The icons of the services are in the private use area, which is how the
	// name is told from them
	var icon = awsServices[0].Icon
	var service = core.Service{
		Id: core.SQS, Icon: icon, Description: "Queues", Shortcut: 'q', Aliases: []string{"queues"},
		Build: func(appCtx *core.AppContext) core.ServicePage {
			builtWith = appCtx
			return page
		},
	}

	var item = awsServiceItem(service, appCtx)

	if item.MainText != icon+" SQS" || item.SecondaryText != "Queues" || item.Shortcut != 'q' || !item.HoldsAwsData {
		t.Errorf("item = %+v, want it named and described by the service, holding AWS data", item)
	}
	if item.ServicePage != core.ServicePage(page) || builtWith != appCtx {
		t.Errorf("page = %v built with %p, want the page of the service built with the app context", item.ServicePage, builtWith)
	}
	if got := serviceCommands(item); !slices.Equal(got, []string{"sqs", "queues"}) {
		t.Errorf("commands = %v, want sqs and queues", got)
	}
}
