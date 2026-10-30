package services

import (
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/settings"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func keepGlobalStyle(t *testing.T) {
	t.Helper()
	var styles, borders = tview.Styles, tview.Borders
	t.Cleanup(func() { tview.Styles, tview.Borders = styles, borders })
}

func newSettingsAppContext() *core.AppContext {
	var appCtx = uitest.NewAppContext()
	*appCtx.Theme = core.DefaultTheme()
	return appCtx
}

func TestSettings__ListsThemesAndMarksTheOneInUse(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx, &fakeSettingsStore{}))
	appCtx.App.QueueUpdateDraw(func() {})

	for _, preset := range core.ThemePresets() {
		uitest.AwaitScreenText(t, appCtx.App, screen, preset.Name)
	}
	uitest.AwaitScreenText(t, appCtx.App, screen, currentThemeMarker)
}

func TestSettings__SelectingAThemeAppliesIt(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	var changes = 0
	appCtx.OnThemeChanged(func() { changes++ })
	uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx, &fakeSettingsStore{}))

	uitest.PressRune(t, appCtx.App, 'j')
	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.Eventually(t, appCtx.App, func() bool { return appCtx.Theme.Name == "High Contrast" })
	if changes != 1 {
		t.Errorf("theme listeners ran %d times, want 1", changes)
	}
}

func TestSettings__StartsOnTheThemeInUse(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	appCtx.SetTheme(core.ThemePresets()[2])
	uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx, &fakeSettingsStore{}))

	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.Eventually(t, appCtx.App, func() bool { return appCtx.Theme.Name == "Nord" })
}

// fakeSettingsStore keeps the settings in memory. The views update it on the
// UI goroutine while the tests read it.
type fakeSettingsStore struct {
	mu      sync.Mutex
	saved   settings.Settings
	updates int
	err     error
}

func (f *fakeSettingsStore) Settings() settings.Settings {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saved
}

func (f *fakeSettingsStore) Update(change func(saved *settings.Settings)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(&f.saved)
	f.updates++
	return f.err
}

func (f *fakeSettingsStore) keys() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return maps.Clone(f.saved.Keys)
}

func (f *fakeSettingsStore) updateCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.updates
}

func TestSettings__HasAPageForEachSectionOfTheSettingsFile(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx, &fakeSettingsStore{}))
	appCtx.App.QueueUpdateDraw(func() {})

	for _, page := range []string{"🭃Theme🭎", "🭃Files🭎", "🭃Keys🭎"} {
		uitest.AwaitScreenText(t, appCtx.App, screen, page)
	}
}

type keySettingsTest struct {
	t      *testing.T
	appCtx *core.AppContext
	store  *fakeSettingsStore
	view   *settingsTable
	screen tcell.SimulationScreen
	builds int
}

func newKeySettingsTest(t *testing.T) *keySettingsTest {
	return newSettingsTableTest(t, &fakeSettingsStore{}, newKeySettings)
}

func newSettingsTableTest(
	t *testing.T, store *fakeSettingsStore, newTable func(*core.AppContext, SettingsStore) *settingsTable,
) *keySettingsTest {
	keepGlobalStyle(t)
	var original = core.APP_KEY_BINDINGS
	t.Cleanup(func() { core.APP_KEY_BINDINGS = original })
	var test = &keySettingsTest{t: t, appCtx: newSettingsAppContext(), store: store}
	test.appCtx.OnKeyBindingsChanged(func() { test.builds++ })
	test.view = newTable(test.appCtx, test.store)
	test.screen = uitest.ShowPage(t, test.appCtx.App, test.view)
	return test
}

// selectKey selects the row of the binding
func (test *keySettingsTest) selectKey(name string) {
	test.t.Helper()
	uitest.OnUI(test.t, test.appCtx.App, func() {
		for row := 1; row < test.view.table.GetTable().GetRowCount(); row++ {
			if test.view.table.GetPrivateData(row, settingsNameCol) == name {
				test.view.table.Select(row, 0)
				return
			}
		}
		test.t.Fatalf("%s is not listed", name)
	})
}

// changeKey changes the key of the binding as the user does, with Enter
func (test *keySettingsTest) changeKey(name string, key string) {
	test.t.Helper()
	test.selectKey(name)
	uitest.PressKey(test.t, test.appCtx.App, tcell.KeyEnter)
	uitest.OnUI(test.t, test.appCtx.App, func() {
		if test.appCtx.App.GetFocus() != test.view.input {
			test.t.Fatal("Enter did not ask for the value")
		}
		test.view.input.SetText(key)
	})
	uitest.PressKey(test.t, test.appCtx.App, tcell.KeyEnter)
}

func (test *keySettingsTest) rowOf(name string) core.TableRow {
	test.t.Helper()
	var found core.TableRow
	uitest.OnUI(test.t, test.appCtx.App, func() {
		for row := 1; row < test.view.table.GetTable().GetRowCount(); row++ {
			if test.view.table.GetPrivateData(row, settingsNameCol) == name {
				for col := range 4 {
					found = append(found, test.view.table.GetCellText(row, col))
				}
			}
		}
	})
	return found
}

func TestKeySettings__ListsEveryKeyWithItsDefault(t *testing.T) {
	var test = newKeySettingsTest(t)

	if got := test.rowOf("reset"); strings.Join(got, "|") != "reset|r|r|Load the table again" {
		t.Errorf("reset row = %v", got)
	}
	var rows int
	uitest.OnUI(t, test.appCtx.App, func() { rows = test.view.table.GetTable().GetRowCount() })
	var sections = map[string]bool{}
	for _, info := range core.KeyBindingInfos() {
		sections[info.Section()] = true
	}
	if want := len(core.KeyBindingInfos()) + len(sections) + 1; rows != want {
		t.Errorf("rows = %d, want a row for each of the %d keys and %d headings", rows-1, len(core.KeyBindingInfos()), len(sections))
	}
}

func TestKeySettings__ListsEachSectionUnderAHeading(t *testing.T) {
	var test = newKeySettingsTest(t)

	var headings []string
	var firstSelected string
	uitest.OnUI(t, test.appCtx.App, func() {
		for row := 1; row < test.view.table.GetTable().GetRowCount(); row++ {
			if test.view.table.GetPrivateData(row, settingsNameCol) == "" {
				headings = append(headings, test.view.table.GetCellText(row, 0))
			}
		}
		var row, _ = test.view.table.GetTable().GetSelection()
		firstSelected = test.view.table.GetPrivateData(row, settingsNameCol)
	})

	if len(headings) < 3 || headings[0] != "General: Moving around" || !slices.Contains(headings, "CloudWatch") {
		t.Errorf("headings = %v, want General: Moving around first and CloudWatch among them", headings)
	}
	if firstSelected == "" {
		t.Error("a heading is selected")
	}
}

func TestKeySettings__ChangingAKeyUsesItSavesItAndBuildsThePagesAgain(t *testing.T) {
	var test = newKeySettingsTest(t)

	test.changeKey("reset", "R")

	uitest.Eventually(t, test.appCtx.App, func() bool { return core.APP_KEY_BINDINGS.Reset == 'R' }, "reset is on R")
	if got := test.store.keys(); !maps.Equal(got, map[string]string{"reset": "R"}) {
		t.Errorf("saved keys = %v, want reset on R", got)
	}
	if got := test.rowOf("reset"); got[1] != "R" || got[2] != "r" {
		t.Errorf("reset row = %v, want key R and default r", got)
	}
	uitest.OnUI(t, test.appCtx.App, func() {
		if test.builds != 1 {
			t.Errorf("pages built again %d times, want once", test.builds)
		}
		if row, _ := test.view.table.GetTable().GetSelection(); test.view.table.GetPrivateData(row, settingsNameCol) != "reset" {
			t.Error("the row of the key changed is no longer selected")
		}
	})
}

func TestKeySettings__TheRemoveKeyPutsBackTheDefault(t *testing.T) {
	var test = newKeySettingsTest(t)
	test.changeKey("reset", "R")
	uitest.Eventually(t, test.appCtx.App, func() bool { return core.APP_KEY_BINDINGS.Reset == 'R' }, "reset is on R")

	uitest.PressRune(t, test.appCtx.App, core.APP_KEY_BINDINGS.RemoveItem)

	uitest.Eventually(t, test.appCtx.App, func() bool { return core.APP_KEY_BINDINGS.Reset == 'r' }, "reset is back on r")
	if got := test.store.keys(); len(got) != 0 {
		t.Errorf("saved keys = %v, want none as every key is its default", got)
	}
}

func TestKeySettings__AKeyThatCantBeUsedIsReportedAndNotSaved(t *testing.T) {
	var test = newKeySettingsTest(t)

	test.changeKey("reset", "y")

	uitest.AwaitScreenText(t, test.appCtx.App, test.screen, `"y" is already used by "textCopy"`)
	if core.APP_KEY_BINDINGS.Reset != 'r' || test.store.updateCount() != 0 {
		t.Errorf("reset = %q, updates = %d, want nothing changed", core.APP_KEY_BINDINGS.Reset, test.store.updateCount())
	}
}

func TestKeySettings__ASequenceCanBeTurnedOff(t *testing.T) {
	var test = newKeySettingsTest(t)

	test.changeKey("openLinkSequence", "")

	uitest.Eventually(t, test.appCtx.App, func() bool { return core.APP_KEY_BINDINGS.OpenLinkSequence == "" }, "gx is off")
	if got := test.rowOf("openLinkSequence"); got[1] != "off" {
		t.Errorf("row = %v, want the key shown as off", got)
	}
	if got := test.store.keys(); !maps.Equal(got, map[string]string{"openLinkSequence": ""}) {
		t.Errorf("saved keys = %v", got)
	}
}

func TestKeySettings__AKeyThatCantBeSavedIsStillUsedAndReported(t *testing.T) {
	var test = newKeySettingsTest(t)
	test.store.err = errors.New("disk full")

	test.changeKey("reset", "R")

	uitest.AwaitScreenText(t, test.appCtx.App, test.screen, "disk full")
	if core.APP_KEY_BINDINGS.Reset != 'R' {
		t.Error("the key was not used")
	}
}

func TestFilesSettings__ChangingTheDownloadDirectorySavesIt(t *testing.T) {
	var store = &fakeSettingsStore{saved: settings.Settings{Files: settings.Files{DownloadDir: "~/old"}}}
	var test = newSettingsTableTest(t, store, newFilesSettings)
	if got := test.rowOf("downloadDir"); got[1] != "~/old" || got[2] != "the working directory" {
		t.Errorf("row = %v, want the saved directory and the working directory by default", got)
	}
	var dir = filepath.Join(t.TempDir(), "csv")

	test.changeKey("downloadDir", dir)

	uitest.Eventually(t, test.appCtx.App, func() bool { return store.Settings().Files.DownloadDir == dir }, "the directory is saved")
	uitest.OnUI(t, test.appCtx.App, func() {
		if test.appCtx.DownloadDir != dir {
			t.Errorf("download directory = %q, want %q", test.appCtx.DownloadDir, dir)
		}
		if test.builds != 0 {
			t.Error("changing a file setting built the pages again")
		}
	})
}

func TestFilesSettings__TheRemoveKeyPutsBackTheWorkingDirectory(t *testing.T) {
	var store = &fakeSettingsStore{saved: settings.Settings{Files: settings.Files{DownloadDir: "~/old"}}}
	var test = newSettingsTableTest(t, store, newFilesSettings)
	test.selectKey("downloadDir")

	uitest.PressRune(t, test.appCtx.App, core.APP_KEY_BINDINGS.RemoveItem)

	uitest.Eventually(t, test.appCtx.App, func() bool { return store.Settings().Files.DownloadDir == "" }, "the directory is cleared")
}
