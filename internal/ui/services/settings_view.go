package services

import (
	"fmt"
	"maps"
	"strings"

	"aws-tui/internal/settings"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const currentThemeMarker = "● in use"

// The pages of the settings, one for each section of the settings file
const (
	settingsThemePage = "Theme"
	settingsFilesPage = "Files"
	settingsKeysPage  = "Keys"
)

// SettingsStore keeps the settings the user chooses
type SettingsStore interface {
	Settings() settings.Settings
	// Update changes the settings and saves them in the settings file
	Update(change func(settings *settings.Settings)) error
}

// NewSettingsView lets the user change the settings of the app, with a page
// for each section of the settings file. Changes apply straight away and are
// saved.
func NewSettingsView(appCtx *core.AppContext, store SettingsStore) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x7A7A7A))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceRootView = core.NewServiceRootView(string(core.SETTINGS), appCtx)
	serviceRootView.
		AddAndSwitchToPage(settingsThemePage, newThemeSettings(appCtx), true).
		AddPage(settingsFilesPage, newFilesSettings(appCtx, store), true, true).
		AddPage(settingsKeysPage, newKeySettings(appCtx, store), true, true)
	serviceRootView.InitPageNavigation()

	return serviceRootView
}

// newThemeSettings lists the themes. Picking one applies it straight away.
func newThemeSettings(appCtx *core.AppContext) *core.ServicePageView {
	var listView = tview.NewList().
		SetHighlightFullLine(true)

	listView.
		SetBorder(true).
		SetBorderPadding(1, 0, 1, 1).
		SetTitle("Theme")

	var themeList = core.NewFilterableList(listView, appCtx)
	var serviceView = core.NewServicePageView(appCtx)

	var currentIdx = 0
	for i, preset := range core.ThemePresets() {
		var marker = ""
		if preset.Name == appCtx.Theme.Name {
			marker = currentThemeMarker
			currentIdx = i
		}

		themeList.AddItem(preset.Name, marker, 0, func() {
			appCtx.SetTheme(preset)
		})
	}
	themeList.SetCurrentItem(currentIdx)

	serviceView.MainPage.AddItem(themeList, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{themeList},
		},
	)
	return serviceView
}

// settingRow is a setting the settings tables list and change
type settingRow struct {
	group       string
	name        string
	description string
	// hint tells what the setting can be set to
	hint         string
	defaultValue string
	// empty is shown for a setting set to nothing
	empty string
	value func() string
	// change uses the value and saves it, and returns why it could not
	change func(value string) error
}

const (
	settingsEditPage = "EDIT_SETTING"
	settingsNameCol  = 1
)

// settingsTable lists settings and changes them: Enter asks for a new value,
// and the remove key puts back the default one
type settingsTable struct {
	*core.ServicePageView
	table   *core.SelectableTable[string]
	input   *core.InputField
	rows    []settingRow
	editing string
	appCtx  *core.AppContext
}

func newSettingsTable(appCtx *core.AppContext, title string, rows []settingRow) *settingsTable {
	var view = &settingsTable{
		ServicePageView: core.NewServicePageView(appCtx),
		table: core.NewSelectableTable[string](
			title, core.TableRow{"Group", "Setting", "Value", "Default", "Does"}, appCtx,
		),
		input:  core.NewInputField(appCtx.Theme),
		rows:   rows,
		appCtx: appCtx,
	}

	view.table.HighlightSearch = true
	view.table.SetRowColourFunc(func(_ core.TableRow, name string) (tcell.Color, bool) {
		var row, ok = view.row(name)
		return appCtx.Theme.TertiaryTextColour, ok && row.value() != row.defaultValue
	})
	view.table.SetSelectionChangedFunc(func(row, column int) {})
	view.table.SetSelectedFunc(func(row, column int) {
		if row > 0 {
			view.edit(view.table.GetPrivateData(row, settingsNameCol))
		}
	})
	view.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == core.APP_KEY_BINDINGS.RemoveItem {
			var selected, _ = view.table.GetTable().GetSelection()
			if row, ok := view.row(view.table.GetPrivateData(selected, settingsNameCol)); ok {
				view.change(row, row.defaultValue)
			}
			return nil
		}
		return event
	})

	var removeKeys = core.FormatRune(core.APP_KEY_BINDINGS.RemoveItem)
	if sequence := core.APP_KEY_BINDINGS.RemoveItemSequence; len(sequence) > 0 {
		removeKeys += ", " + sequence
	}
	view.table.HelpView.View.
		AddKey(core.APP_KEY_BINDINGS.Done, "Change the setting").
		AddItem(removeKeys, "Put back the default", nil)

	view.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			var row, ok = view.row(view.editing)
			view.closeEdit()
			if ok {
				view.change(row, view.input.GetText())
			}
		case tcell.KeyEscape:
			view.closeEdit()
		}
	})

	view.Pages.AddPage(settingsEditPage, core.FloatingView("Change", view.input, 80, 3), true, false)
	view.MainPage.AddItem(view.table, 0, 1, true)
	view.InitViewNavigation(
		[][]core.View{
			{view.table},
		},
	)
	view.populate()
	return view
}

func (inst *settingsTable) row(name string) (settingRow, bool) {
	for _, row := range inst.rows {
		if row.name == name {
			return row, true
		}
	}
	return settingRow{}, false
}

// shown is the value to show, which is the empty text of the setting when it
// is set to nothing
func (row settingRow) shown(value string) string {
	if value == "" {
		return row.empty
	}
	return value
}

// populate lists the settings as they are, keeping the row selected
func (inst *settingsTable) populate() {
	var selected, _ = inst.table.GetTable().GetSelection()
	var rows []core.TableRow
	var names []string
	var changed = 0
	for _, row := range inst.rows {
		var value = row.value()
		rows = append(rows, core.TableRow{
			row.group, row.name, row.shown(value), row.shown(row.defaultValue), row.description,
		})
		names = append(names, row.name)
		if value != row.defaultValue {
			changed++
		}
	}

	var extra = ""
	if changed > 0 {
		extra = fmt.Sprintf("%d changed", changed)
	}
	inst.table.SetTitleExtra(extra)
	inst.table.SetData(rows, names, settingsNameCol)
	inst.table.GetCell(0, 4).SetExpansion(1)
	if selected > 1 {
		inst.table.Select(selected, 0)
	}
}

// edit asks for the new value of the setting
func (inst *settingsTable) edit(name string) {
	var row, ok = inst.row(name)
	if !ok {
		return
	}
	inst.editing = name
	inst.input.
		SetLabel(name + " ").
		SetPlaceholder(row.hint).
		SetText(row.value())
	inst.ShowPage(settingsEditPage)
	inst.SendToFront(settingsEditPage)
	inst.appCtx.App.SetFocus(inst.input)
}

func (inst *settingsTable) closeEdit() {
	inst.HidePage(settingsEditPage)
	inst.appCtx.App.SetFocus(inst.table.GetTable())
}

func (inst *settingsTable) change(row settingRow, value string) {
	var err = row.change(strings.TrimSpace(value))
	inst.populate()
	if err != nil {
		inst.DisplayMessage(core.ErrorPrompt, "%s", err.Error())
	}
}

// newFilesSettings sets the directory tables are saved to
func newFilesSettings(appCtx *core.AppContext, store SettingsStore) *settingsTable {
	return newSettingsTable(appCtx, "Files", []settingRow{{
		group:       "Files",
		name:        "downloadDir",
		description: "The directory tables are saved to with :w. While the app runs it follows the last file saved",
		hint:        "A directory such as ~/Downloads, or nothing for the working directory",
		empty:       "the working directory",
		value:       func() string { return store.Settings().Files.DownloadDir },
		change: func(dir string) error {
			appCtx.DownloadDir = dir
			return store.Update(func(saved *settings.Settings) { saved.Files.DownloadDir = dir })
		},
	}})
}

// newKeySettings lists the keys and changes them. A key that changes is saved
// and the other pages are built again with it.
func newKeySettings(appCtx *core.AppContext, store SettingsStore) *settingsTable {
	var rows []settingRow
	for _, info := range core.KeyBindingInfos() {
		var name = info.Name
		rows = append(rows, settingRow{
			group:        info.Group,
			name:         name,
			description:  info.Description,
			hint:         core.KeyBindingHint(name),
			defaultValue: core.DefaultKeyBinding(name),
			empty:        "off",
			value:        func() string { return core.CurrentKeyBinding(name) },
			change:       func(key string) error { return changeKey(appCtx, store, name, key) },
		})
	}
	return newSettingsTable(appCtx, "Keys", rows)
}

// changeKey binds the key, saves it and builds the other pages again with it.
// A key that can't be used changes nothing.
func changeKey(appCtx *core.AppContext, store SettingsStore, name string, key string) error {
	if err := core.ChangeKeyBinding(name, key); err != nil {
		return err
	}

	var err = store.Update(func(saved *settings.Settings) {
		var keys = maps.Clone(saved.Keys)
		if keys == nil {
			keys = map[string]string{}
		}
		// Only the keys that differ from their default are kept
		delete(keys, name)
		if current := core.CurrentKeyBinding(name); current != core.DefaultKeyBinding(name) {
			keys[name] = current
		}
		if len(keys) == 0 {
			keys = nil
		}
		saved.Keys = keys
	})

	appCtx.KeyBindingsChanged()
	if err != nil {
		return fmt.Errorf("the key is in use, but it could not be saved: %w", err)
	}
	return nil
}
