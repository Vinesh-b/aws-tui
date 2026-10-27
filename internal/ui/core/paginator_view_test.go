package core

import (
	"context"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newSubscribedPaginator is a bar subscribed to a monitor whose polls are
// dropped, so the test sets the session on its own goroutine
func newSubscribedPaginator(t *testing.T) (*PaginatorView, *AppContext) {
	t.Helper()
	var appCtx, monitor, _ = newMonitorTest(t)
	monitor.queue = func(func()) {}
	var view = CreatePaginatorView("service", appCtx)
	t.Cleanup(view.Stop)
	return view, appCtx
}

func TestPaginatorView__ShowsTheSessionOfTheMonitor(t *testing.T) {
	var view, appCtx = newSubscribedPaginator(t)
	if view.session != nil {
		t.Fatalf("session = %+v, want it loading", view.session)
	}

	appCtx.Session().set(&sessionInfo{profile: "dev"})

	if view.session == nil || view.session.profile != "dev" {
		t.Errorf("session = %+v, want the one of the monitor", view.session)
	}
}

func TestPaginatorView__StopEndsTheSubscription(t *testing.T) {
	var view, appCtx = newSubscribedPaginator(t)

	view.Stop()
	view.Stop()
	appCtx.Session().set(&sessionInfo{profile: "dev"})

	if view.session != nil {
		t.Errorf("session = %+v, want a stopped bar left alone", view.session)
	}
}

func TestServiceRootView__CloseStopsThePaginatorMoreThanOnce(t *testing.T) {
	var view = NewServiceRootView("service", newTestAppContext())

	view.Close()
	view.Close()
}

func newTestPaginator(t *testing.T, serviceColour tcell.Color) (*PaginatorView, *AppTheme) {
	t.Helper()
	keepGlobalStyle(t)
	var theme = DefaultTheme()
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &theme)

	tview.Styles.BorderColor = serviceColour
	// The session is set by the tests, so the bar does not look it up
	var view = newPaginatorView("Step Functions", appCtx)

	view.SetPages([]string{"StateMachines", "Execution Details"})
	view.SetCurrentPage(1)
	return view, &theme
}

func drawPaginator(t *testing.T, view *PaginatorView, width int) tcell.SimulationScreen {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, 1)
	view.SetRect(0, 0, width, 1)
	view.Draw(screen)
	screen.Show()
	return screen
}

func screenLine(screen tcell.SimulationScreen) string {
	var cells, width, _ = screen.GetContents()
	var line strings.Builder
	for col := 0; col < width; col++ {
		if runes := cells[col].Runes; len(runes) > 0 {
			line.WriteString(string(runes))
		} else {
			// A cell that was never drawn
			line.WriteString(" ")
		}
	}
	return line.String()
}

// lineColours finds the text on the bar and returns the colours of its first
// cell. The bar has runes outside ASCII so it is searched by rune, not byte.
func lineColours(t *testing.T, screen tcell.SimulationScreen, text string) (fg, bg tcell.Color) {
	t.Helper()
	var line = screenLine(screen)
	var idx = strings.Index(line, text)
	if idx < 0 {
		t.Fatalf("%q is not on the screen: %q", text, line)
	}
	var cells, _, _ = screen.GetContents()
	fg, bg, _ = cells[utf8.RuneCountInString(line[:idx])].Style.Decompose()
	return fg, bg
}

func clickAt(view *PaginatorView, x int) bool {
	var event = tcell.NewEventMouse(x, 0, tcell.Button1, tcell.ModNone)
	var consumed, _ = view.MouseHandler()(tview.MouseLeftClick, event, func(tview.Primitive) {})
	return consumed
}

func TestPaginatorView__DrawsServiceAndPagePills(t *testing.T) {
	var serviceColour = tcell.NewHexColor(0xFF3399)
	var view, theme = newTestPaginator(t, serviceColour)
	var screen = drawPaginator(t, view, 120)

	var line = screenLine(screen)
	if !strings.HasPrefix(line, " 🭃Step Functions🭎 🭃StateMachines🭎 🭃Execution Details🭎") {
		t.Fatalf("line = %q, want the service and page pills", line)
	}

	for _, want := range []struct {
		text   string
		fg, bg tcell.Color
	}{
		{"Step Functions", theme.SecondaryTextColour, serviceColour},
		{"StateMachines", theme.PrimaryTextColour, theme.ContrastBackgroundColor},
		{"Execution Details", theme.SecondaryTextColour, theme.MoreContrastBackgroundColor},
	} {
		if fg, bg := lineColours(t, screen, want.text); fg != want.fg || bg != want.bg {
			t.Errorf("%s = fg %x bg %x, want fg %x bg %x",
				want.text, fg.Hex(), bg.Hex(), want.fg.Hex(), want.bg.Hex())
		}
	}
}

func TestPaginatorView__ColoursSessionDetails(t *testing.T) {
	var view, theme = newTestPaginator(t, tcell.ColorBlue)

	for _, test := range []struct {
		name     string
		duration time.Duration
		text     string
		fg       tcell.Color
	}{
		{"no expiry", math.MinInt64, "no expiry", theme.PrimaryTextColour},
		{"plenty left", time.Hour, "1h00m", theme.PrimaryTextColour},
		{"expiring soon", 10 * time.Minute, "10m", theme.TertiaryTextColour},
		{"expired", 0, "Expired", theme.ErrorTextColour},
	} {
		view.session = &sessionInfo{
			profile: "dev", accountId: "123456789012", userId: "AROA:me", duration: test.duration,
		}
		var screen = drawPaginator(t, view, 160)

		if !strings.HasSuffix(strings.TrimRight(screenLine(screen), " "), "User AROA:me") {
			t.Errorf("%s: line = %q, want the session details right aligned", test.name, screenLine(screen))
		}
		if fg, _ := lineColours(t, screen, "Profile"); fg != theme.PlaceholderTextColour {
			t.Errorf("%s: label fg = %x, want %x", test.name, fg.Hex(), theme.PlaceholderTextColour.Hex())
		}
		if fg, _ := lineColours(t, screen, "123456789012"); fg != theme.PrimaryTextColour {
			t.Errorf("%s: value fg = %x, want %x", test.name, fg.Hex(), theme.PrimaryTextColour.Hex())
		}
		if fg, _ := lineColours(t, screen, test.text); fg != test.fg {
			t.Errorf("%s: duration fg = %x, want %x", test.name, fg.Hex(), test.fg.Hex())
		}
	}
}

func TestPaginatorView__ShowsCredentialErrorsInTheErrorColour(t *testing.T) {
	var view, theme = newTestPaginator(t, tcell.ColorBlue)
	view.session = &sessionInfo{err: "token expired"}
	var screen = drawPaginator(t, view, 120)

	if fg, _ := lineColours(t, screen, "⚠ Credentials Error: token expired"); fg != theme.ErrorTextColour {
		t.Errorf("error fg = %x, want %x", fg.Hex(), theme.ErrorTextColour.Hex())
	}
}

func TestPaginatorView__CutsSessionDetailsOffBeforeThePills(t *testing.T) {
	var view, _ = newTestPaginator(t, tcell.ColorBlue)
	view.session = &sessionInfo{profile: "dev", accountId: "123456789012", userId: "AROA:me", duration: time.Hour}
	var screen = drawPaginator(t, view, 80)

	var line = screenLine(screen)
	if !strings.Contains(line, "🭃Execution Details🭎  gT gt  Profile") {
		t.Errorf("line = %q, want the pills and hint kept and the session details after them", line)
	}
}

func TestPaginatorView__ClickingAPagePillSelectsThePage(t *testing.T) {
	var view, _ = newTestPaginator(t, tcell.ColorBlue)
	var screen = drawPaginator(t, view, 120)
	var selected = ""
	view.SetPageSelectedFunc(func(name string) { selected = name })

	if !strings.Contains(screenLine(screen), "🭃StateMachines🭎") {
		t.Fatalf("line = %q, want the page pills drawn", screenLine(screen))
	}

	// Columns: 1 padding, service pill is 16 wide, then a gap
	var stateMachinesX = 1 + pillWidth("Step Functions") + 1
	var executionDetailsX = stateMachinesX + pillWidth("StateMachines") + 1

	if !clickAt(view, 2) || selected != "" {
		t.Errorf("clicking the service pill selected %q, want nothing and the click consumed", selected)
	}
	if !clickAt(view, stateMachinesX) || selected != "StateMachines" {
		t.Errorf("selected = %q, want StateMachines", selected)
	}
	if !clickAt(view, executionDetailsX+pillWidth("Execution Details")-1) || selected != "Execution Details" {
		t.Errorf("selected = %q, want Execution Details", selected)
	}
	if !clickAt(view, 70) {
		t.Error("expected a click on the empty bar to be consumed so it does not take the focus")
	}
}

func TestServiceRootView__PaginatorPillSwitchesPage(t *testing.T) {
	var appCtx = newTestAppContext()
	var view = NewServiceRootView("service", appCtx)
	t.Cleanup(view.Close)

	view.
		AddAndSwitchToPage("First", NewServicePageView(appCtx), true).
		AddPage("Second", NewServicePageView(appCtx), true, true)
	view.InitPageNavigation()

	view.paginatorView.SetRect(0, 0, 100, 1)
	var secondX = 1 + pillWidth("service") + 1 + pillWidth("First") + 1
	clickAt(view.paginatorView, secondX)

	if name, _ := view.pages.GetFrontPage(); name != "Second" || view.paginatorView.pageBar.GetCurrent() != 1 {
		t.Errorf("front page = %q, current pill = %d, want Second and 1", name, view.paginatorView.pageBar.GetCurrent())
	}
}

func TestPaginatorView__ShowsLoadingUntilTheFirstPoll(t *testing.T) {
	var view, theme = newTestPaginator(t, tcell.ColorBlue)
	var screen = drawPaginator(t, view, 120)

	if line := strings.TrimRight(screenLine(screen), " "); !strings.HasSuffix(line, "Loading session…") {
		t.Errorf("line = %q, want the session loading right aligned", line)
	}
	if fg, _ := lineColours(t, screen, "Loading session…"); fg != theme.PlaceholderTextColour {
		t.Errorf("loading fg = %x, want %x", fg.Hex(), theme.PlaceholderTextColour.Hex())
	}
}

func TestPaginatorView__ShowsPageKeysOnlyWithMoreThanOnePage(t *testing.T) {
	var view, theme = newTestPaginator(t, tcell.ColorBlue)

	var screen = drawPaginator(t, view, 120)
	if !strings.Contains(screenLine(screen), "🭃Execution Details🭎  gT gt") {
		t.Errorf("line = %q, want the page keys after the pills", screenLine(screen))
	}
	if fg, _ := lineColours(t, screen, "gT gt"); fg != theme.PlaceholderTextColour {
		t.Errorf("hint fg = %x, want %x", fg.Hex(), theme.PlaceholderTextColour.Hex())
	}

	view.SetPages([]string{"Only"})
	view.SetCurrentPage(0)
	screen = drawPaginator(t, view, 120)
	if strings.Contains(screenLine(screen), "Alt+") {
		t.Errorf("line = %q, want no page keys for a single page", screenLine(screen))
	}
}

func TestFormatSessionDuration__ShowsSecondsOnlyInTheLastMinute(t *testing.T) {
	for _, test := range []struct {
		duration time.Duration
		want     string
	}{
		{2*time.Hour + 5*time.Minute + 59*time.Second, "2h05m"},
		{time.Hour, "1h00m"},
		{59*time.Minute + 59*time.Second, "59m"},
		{time.Minute, "1m"},
		{59 * time.Second, "59s"},
		{time.Second, "1s"},
	} {
		if got := formatSessionDuration(test.duration); got != test.want {
			t.Errorf("formatSessionDuration(%s) = %q, want %q", test.duration, got, test.want)
		}
	}
}

func TestFirstPollDelay__IsImmediateOnlyWhenPollingAtStart(t *testing.T) {
	var appCtx = newTestAppContext()
	if got := firstPollDelay(appCtx); got != CredsPollRate {
		t.Errorf("delay = %s, want the poll rate by default", got)
	}

	appCtx.PollSessionAtStart = true
	if got := firstPollDelay(appCtx); got != 0 {
		t.Errorf("delay = %s, want no delay when polling at start", got)
	}
}

// notifyApiReset calls the listeners as ResetApiClients does, without
// replacing the global api clients the other tests use
func notifyApiReset(appCtx *AppContext) {
	for _, listener := range slices.Clone(appCtx.apiResetListeners) {
		(*listener)()
	}
}

func TestSessionProfileName__PickedProfileIsShownOverEnvironment(t *testing.T) {
	t.Setenv("AWS_PROFILE", "from-env")

	if got := sessionProfileName("picked"); got != "picked" {
		t.Errorf("profile = %q, want %q", got, "picked")
	}
	if got := sessionProfileName(""); got != "from-env" {
		t.Errorf("profile = %q, want %q", got, "from-env")
	}

	t.Setenv("AWS_PROFILE", "")
	if got := sessionProfileName(""); got != "unset" {
		t.Errorf("profile = %q, want %q", got, "unset")
	}
}

func TestPaginatorView__ApiResetClearsTheSession(t *testing.T) {
	var view, appCtx = newSubscribedPaginator(t)
	appCtx.Session().set(&sessionInfo{profile: "old"})

	notifyApiReset(appCtx)

	if view.session != nil {
		t.Errorf("session = %+v, want it cleared until the new profile is polled", view.session)
	}
	if got := appCtx.Session().generation.Load(); got != 1 {
		t.Errorf("generation = %d, want 1", got)
	}
}

func TestPageKeysHint__FollowsThePageSequences(t *testing.T) {
	var original = APP_KEY_BINDINGS
	t.Cleanup(func() { APP_KEY_BINDINGS = original })

	APP_KEY_BINDINGS.PageForwardSequence = ""
	if got := pageKeysHint(); got != "gT" {
		t.Errorf("hint = %q, want gT", got)
	}
	APP_KEY_BINDINGS.PageBackSequence = ""
	if got := pageKeysHint(); got != "" {
		t.Errorf("hint = %q, want none", got)
	}
}
