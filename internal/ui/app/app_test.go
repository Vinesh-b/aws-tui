package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/settings"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/sqs"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/smithy-go/logging"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCreateApplication__BuildsRootViewWithoutAwsProfiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	var tuiApp = CreateApplication(context.Background(), aws.Config{Region: "us-east-1"}, "test-version")

	if tuiApp == nil {
		t.Fatal("application was not created")
	}
}

func TestCreateApplication__ApiClientsLogToTheInAppLogger(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	CreateApplication(context.Background(), aws.Config{Region: "us-east-1"}, "test-version")

	var logger = awsapi.GetAwsApiClients().Config.Logger
	if _, ok := logger.(logging.StandardLogger); !ok {
		t.Errorf("api client logger = %T, want the in-app logger so SDK output stays off the terminal", logger)
	}
}

type countingPage struct {
	*tview.Box
	builds int
	closed int
}

func (p *countingPage) Close() { p.closed++ }

func (p *countingPage) GetLastFocusedView() tview.Primitive { return p.Box }

func TestRebuildServicePages__ReplacesDataPagesAndKeepsOthers(t *testing.T) {
	var pages = tview.NewPages()
	var builds = 0
	var build = func() core.ServicePage {
		builds++
		return &countingPage{Box: tview.NewBox(), builds: builds}
	}
	var data = newServiceItem("data", "", 'd', build)
	var static = staticServiceItem("static", "", 's', &countingPage{Box: tview.NewBox()})
	var oldStatic = static.ServicePage
	var oldData = data.ServicePage
	pages.AddPage("data", data.ServicePage, true, false)
	pages.AddPage("static", static.ServicePage, true, true)
	pages.AddPage("floating", tview.NewBox(), true, false)
	pages.AddPage("home", tview.NewBox(), true, false)

	rebuildServicePages(pages, []*ServiceItem{data, static})

	if builds != 2 || data.ServicePage == oldData {
		t.Errorf("data page was not rebuilt (builds=%d)", builds)
	}
	if oldData.(*countingPage).closed != 1 {
		t.Error("the replaced page should be closed")
	}
	if static.ServicePage != oldStatic {
		t.Error("static page should be left alone")
	}
	// Page names are listed from front to back
	if got := pages.GetPageNames(false); !slices.Equal(got[:2], []string{"home", "floating"}) {
		t.Errorf("page order = %v, want the service pages behind the home and floating pages", got)
	}
	if got := pages.GetPageNames(true); !slices.Equal(got, []string{"static"}) {
		t.Errorf("visible pages = %v, want only static", got)
	}
}

func TestRebuildServicePages__KeepsAVisibleDataPageVisible(t *testing.T) {
	var pages = tview.NewPages()
	var item = newServiceItem("data", "", 'd', func() core.ServicePage {
		return &countingPage{Box: tview.NewBox()}
	})
	pages.AddPage("data", item.ServicePage, true, true)

	rebuildServicePages(pages, []*ServiceItem{item})

	if got := pages.GetPageNames(true); !slices.Equal(got, []string{"data"}) {
		t.Errorf("visible pages = %v, want data", got)
	}
}

type fakeSqsApi struct {
	awsapi.SqsApi
	queues []string
}

func (f *fakeSqsApi) ListQueues(ctx context.Context, force bool) ([]string, error) {
	return f.queues, nil
}

func (f *fakeSqsApi) GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error) {
	return nil, nil
}

func (f *fakeSqsApi) ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error) {
	return nil, nil
}

func TestResetApiClients__ClearsDataLoadedByServicePages(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var pages = tview.NewPages()
	var item = newServiceItem("sqs", "", 'q', func() core.ServicePage {
		return sqs.NewSqsHomeView(appCtx, &fakeSqsApi{queues: []string{"https://sqs.example.com/1/orders-queue"}})
	})
	pages.AddPage("sqs", item.ServicePage, true, true)
	appCtx.OnApiClientsReset(func() { rebuildServicePages(pages, []*ServiceItem{item}) })
	var screen = uitest.RunApp(t, appCtx.App)
	uitest.OnUI(t, appCtx.App, func() {
		appCtx.App.SetRoot(pages, true)
		appCtx.App.SetFocus(item.ServicePage.GetLastFocusedView())
	})
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-queue")

	uitest.OnUI(t, appCtx.App, func() {
		appCtx.ResetApiClients(aws.Config{
			Region:      "eu-west-1",
			Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
		}, "other")
		appCtx.App.SetFocus(item.ServicePage.GetLastFocusedView())
	})

	appCtx.App.QueueUpdateDraw(func() {})
	uitest.Eventually(t, appCtx.App, func() bool {
		return !strings.Contains(uitest.ScreenText(screen), "orders-queue")
	})
}

func TestNewDebugLogView__KeepsOnlyTheNewestLines(t *testing.T) {
	var view = NewDebugLogView(tview.NewApplication())
	for i := 0; i < maxDebugLogLines+100; i++ {
		fmt.Fprintf(view, "line %d\n", i)
	}

	var screen = tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	view.SetRect(0, 0, 40, 10)
	view.Draw(screen)

	var text = view.GetText(true)
	if strings.Contains(text, "line 0\n") {
		t.Error("oldest line was not dropped")
	}
	if !strings.Contains(text, fmt.Sprintf("line %d\n", maxDebugLogLines+99)) {
		t.Error("newest line is missing")
	}
	if got := strings.Count(text, "\n"); got > maxDebugLogLines {
		t.Errorf("kept %d lines, want at most %d", got, maxDebugLogLines)
	}
}

func TestRebuildThemedPages__ReplacesEveryPageThatCanBeBuiltAgain(t *testing.T) {
	var pages = tview.NewPages()
	var build = func() core.ServicePage { return &countingPage{Box: tview.NewBox()} }
	var data = newServiceItem("data", "", 'd', build)
	var themed = themedServiceItem("themed", "", 't', build)
	var static = staticServiceItem("static", "", 's', &countingPage{Box: tview.NewBox()})
	var oldData, oldThemed, oldStatic = data.ServicePage, themed.ServicePage, static.ServicePage
	for _, item := range []*ServiceItem{data, themed, static} {
		pages.AddPage(item.MainText, item.ServicePage, true, item.MainText == "themed")
	}

	rebuildThemedPages(pages, []*ServiceItem{data, themed, static})

	if data.ServicePage == oldData || themed.ServicePage == oldThemed {
		t.Error("pages that can be built again should have been")
	}
	if static.ServicePage != oldStatic {
		t.Error("a static page should be left alone")
	}
	if got := pages.GetPageNames(true); !slices.Equal(got, []string{"themed"}) {
		t.Errorf("visible pages = %v, want only themed", got)
	}
}

func TestRebuildServicePages__LeavesPagesThatOnlyTakeTheThemeAlone(t *testing.T) {
	var pages = tview.NewPages()
	var themed = themedServiceItem("themed", "", 't', func() core.ServicePage {
		return &countingPage{Box: tview.NewBox()}
	})
	var old = themed.ServicePage
	pages.AddPage("themed", themed.ServicePage, true, true)

	rebuildServicePages(pages, []*ServiceItem{themed})

	if themed.ServicePage != old {
		t.Error("changing profile must not replace a page that holds no AWS data")
	}
}

func TestRebuildThemedPages__KeepsTheHomePageHiddenWhenTheVisiblePageIsReplaced(t *testing.T) {
	var pages = tview.NewPages()
	var settings = themedServiceItem("settings", "", 's', func() core.ServicePage {
		return &countingPage{Box: tview.NewBox()}
	})
	pages.AddPage("settings", settings.ServicePage, true, true)
	pages.AddPage("floating", tview.NewBox(), true, false)
	pages.AddPage("home", tview.NewBox(), true, false)

	rebuildThemedPages(pages, []*ServiceItem{settings})

	if got := pages.GetPageNames(true); !slices.Equal(got, []string{"settings"}) {
		t.Errorf("visible pages = %v, want only settings", got)
	}
}

// startApp creates the application on a simulated screen, the way the tests of
// the settings page need it: with credentials, so nothing polls without any.
func startApp(t *testing.T, opts ...Option) (*tview.Application, tcell.SimulationScreen) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	var styles, borders = tview.Styles, tview.Borders
	t.Cleanup(func() { tview.Styles, tview.Borders = styles, borders })

	var tuiApp = CreateApplication(context.Background(), aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
	}, "test-version", opts...)

	return tuiApp, uitest.RunApp(t, tuiApp)
}

func openSettings(app *tview.Application) {
	// The home page starts with the services list focused, 's' opens Settings
	app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
}

// pressHistoryKey presses the key with the history modifier, through the app
func pressHistoryKey(app *tview.Application, key tcell.Key) {
	app.QueueEvent(tcell.NewEventKey(key, 0, core.APP_KEY_BINDINGS.HistoryModKey))
}

func TestCreateApplication__HistoryKeysGoBackAndForwardBetweenServices(t *testing.T) {
	var tuiApp, screen = startApp(t)
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")
	tuiApp.QueueEvent(tcell.NewEventKey(core.APP_KEY_BINDINGS.ToggleServicesMenu, 0, tcell.ModNone))
	tuiApp.QueueEvent(tcell.NewEventKey(tcell.KeyRune, '?', tcell.ModNone))
	uitest.AwaitScreenText(t, tuiApp, screen, "The bar at the bottom of a service")

	pressHistoryKey(tuiApp, core.APP_KEY_BINDINGS.HistoryBack)
	awaitInUse(t, tuiApp, screen, "Default")

	pressHistoryKey(tuiApp, core.APP_KEY_BINDINGS.HistoryForward)
	uitest.AwaitScreenText(t, tuiApp, screen, "The bar at the bottom of a service")
}

func typeRunes(app *tview.Application, text string) {
	for _, r := range text {
		app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
}

func TestCreateApplication__SequencesChangeThePage(t *testing.T) {
	var tuiApp, screen = startApp(t)
	// The home page starts with the services list focused, 7 opens the logs
	typeRunes(tuiApp, "7")
	uitest.AwaitScreenText(t, tuiApp, screen, "Log Group Details")

	typeRunes(tuiApp, core.APP_KEY_BINDINGS.PageForwardSequence)
	uitest.AwaitScreenText(t, tuiApp, screen, "Log Stream Details")

	typeRunes(tuiApp, core.APP_KEY_BINDINGS.PageBackSequence)
	uitest.AwaitScreenText(t, tuiApp, screen, "Log Group Details")
}

func pressEnter(app *tview.Application) {
	app.QueueEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
}

func TestCreateApplication__ColonCommandsGoToServices(t *testing.T) {
	var tuiApp, screen = startApp(t)
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")

	typeRunes(tuiApp, ":cloudwatchlogs")
	pressEnter(tuiApp)

	uitest.AwaitScreenText(t, tuiApp, screen, "Log Group Details")
}

func TestCreateApplication__ColonCommandsShowWhyTheyFailed(t *testing.T) {
	var tuiApp, screen = startApp(t)
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")

	typeRunes(tuiApp, ":nope")
	pressEnter(tuiApp)

	uitest.AwaitScreenText(t, tuiApp, screen, "not a command or service: nope")
}

func TestCreateApplication__EnterOnACompletionRunsIt(t *testing.T) {
	var tuiApp, screen = startApp(t)
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")

	// cloudwatch alone could be several services, and its first completion
	// is the logs
	typeRunes(tuiApp, ":cloudwatch")
	uitest.AwaitScreenText(t, tuiApp, screen, "cloudwatchalarms")
	pressEnter(tuiApp)

	uitest.AwaitScreenText(t, tuiApp, screen, "Log Group Details")
}

func TestCreateApplication__PickingAThemeInSettingsKeepsTheSettingsPage(t *testing.T) {
	var tuiApp, screen = startApp(t)
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")

	uitest.PressKey(t, tuiApp, tcell.KeyDown)
	uitest.PressKey(t, tuiApp, tcell.KeyEnter)

	uitest.Eventually(t, tuiApp, func() bool {
		return tview.Styles.BorderColor == core.ThemePresets()[1].BorderColour
	})
	awaitInUse(t, tuiApp, screen, "High Contrast")
	if text := uitest.ScreenText(screen); strings.Contains(text, "test-version") {
		t.Errorf("the home page came back over the settings page:\n%s", text)
	}

	// The cursor is still in the rebuilt page, so keys still reach it
	uitest.PressKey(t, tuiApp, tcell.KeyDown)
	uitest.PressKey(t, tuiApp, tcell.KeyEnter)
	awaitInUse(t, tuiApp, screen, "Nord")
}

func TestCreateApplication__SavesTheThemeWhenItChanges(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "config", "settings.json")
	var tuiApp, screen = startApp(t, WithSettingsFile(path))
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")

	uitest.PressKey(t, tuiApp, tcell.KeyDown)
	uitest.PressKey(t, tuiApp, tcell.KeyEnter)
	awaitInUse(t, tuiApp, screen, "High Contrast")

	var saved, err = settings.Load(path)
	if err != nil || saved.Theme != "High Contrast" {
		t.Errorf("saved settings = %+v, %v, want the High Contrast theme", saved, err)
	}

	uitest.PressKey(t, tuiApp, tcell.KeyDown)
	uitest.PressKey(t, tuiApp, tcell.KeyEnter)
	awaitInUse(t, tuiApp, screen, "Nord")
	if saved, _ = settings.Load(path); saved.Theme != "Nord" {
		t.Errorf("saved theme = %q, want Nord", saved.Theme)
	}
}

func TestCreateApplication__StartsWithTheSavedTheme(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "settings.json")
	if err := settings.Save(path, settings.Settings{Theme: "Nord"}); err != nil {
		t.Fatal(err)
	}

	var tuiApp, screen = startApp(t, WithSettingsFile(path))
	openSettings(tuiApp)

	awaitInUse(t, tuiApp, screen, "Nord")
	var nord, _ = core.FindThemePreset("Nord")
	if tview.Styles.BorderColor != nord.BorderColour {
		t.Error("the global style was not set from the saved theme")
	}
}

func TestCreateApplication__StartsWithTheSavedKeyBindings(t *testing.T) {
	var original = core.APP_KEY_BINDINGS
	t.Cleanup(func() { core.APP_KEY_BINDINGS = original })

	var path = filepath.Join(t.TempDir(), "settings.json")
	var chosen = settings.Settings{KeyBindings: map[string]string{"reset": "R", "nope": "x"}}
	if err := settings.Save(path, chosen); err != nil {
		t.Fatal(err)
	}

	startApp(t, WithSettingsFile(path))

	if core.APP_KEY_BINDINGS.Reset != 'R' {
		t.Errorf("reset key = %q, want the saved R", core.APP_KEY_BINDINGS.Reset)
	}
}

func TestCreateApplication__StartsWithTheDefaultThemeForSettingsItCannotUse(t *testing.T) {
	for name, content := range map[string]string{
		"unknown theme":  `{"theme": "Does Not Exist"}`,
		"invalid json":   `{not json`,
		"missing":        "",
		"no theme saved": `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			var path = filepath.Join(t.TempDir(), "settings.json")
			if content != "" {
				os.WriteFile(path, []byte(content), 0o600)
			}

			var tuiApp, screen = startApp(t, WithSettingsFile(path))
			openSettings(tuiApp)

			awaitInUse(t, tuiApp, screen, "Default")
			if content != "" {
				if data, _ := os.ReadFile(path); string(data) != content {
					t.Errorf("starting the app changed the settings file to %q", data)
				}
			}
		})
	}
}

func TestCreateApplication__WithoutASettingsFileNothingIsSaved(t *testing.T) {
	var configDir = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)
	var tuiApp, screen = startApp(t)
	openSettings(tuiApp)
	awaitInUse(t, tuiApp, screen, "Default")

	uitest.PressKey(t, tuiApp, tcell.KeyDown)
	uitest.PressKey(t, tuiApp, tcell.KeyEnter)
	awaitInUse(t, tuiApp, screen, "High Contrast")

	if entries, _ := os.ReadDir(configDir); len(entries) != 0 {
		t.Errorf("config directory holds %d entries, want none", len(entries))
	}
}

// awaitInUse waits for the settings page to show the theme as the one in use,
// which is on the line below its name.
func awaitInUse(t *testing.T, app *tview.Application, screen tcell.SimulationScreen, theme string) {
	t.Helper()
	var deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		app.QueueUpdateDraw(func() {})
		var lines = strings.Split(uitest.ScreenText(screen), "\n")
		for i, line := range lines[:len(lines)-1] {
			if strings.Contains(line, theme) && strings.Contains(lines[i+1], "in use") {
				return
			}
		}
		time.Sleep(20 * time.Millisecond) // sleep-ok: polling the screen
	}
	t.Fatalf("settings never showed %s as in use:\n%s", theme, uitest.ScreenText(screen))
}
