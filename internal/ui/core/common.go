package core

import (
	"aws-tui/internal/awsapi"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppContext struct {
	// Ctx is cancelled when the app shuts down; pass it to AWS api calls.
	Ctx    context.Context
	App    *tview.Application
	Logger *log.Logger
	Theme  *AppTheme
	// PollSessionAtStart looks up the session details shown on the paginator as
	// soon as it is built, instead of after the first poll interval
	PollSessionAtStart bool
	// DownloadDir is the directory proposed when saving a file. It is empty for
	// the working directory, and follows the directory of the last file saved.
	// It is only used on the UI goroutine.
	DownloadDir string
	// QueriesPath is the file the Logs Insights queries are kept in, which is
	// empty if they are not kept
	QueriesPath string

	listenersMtx      sync.Mutex
	apiResetListeners []*func()
	themeListeners    []func()
	keyListeners      []func()
	openResource      func(resource arn.ARN) error
	// focusedTable is the table that last had the focus, to save with :w, and
	// focusedLinks the details that last had it, to open their links. They
	// are only used on the UI goroutine.
	focusedTable tableSaver
	focusedLinks linkOpener
	// session is the one monitor of the session for all the status bars
	session     *SessionMonitor
	sessionOnce sync.Once
	clipboard   appClipboard
}

// linkOpener is a view with values that link to resources
type linkOpener interface {
	holds(view tview.Primitive) bool
	openSelectedLink()
}

// OpenLink opens the resource the selected value of the focused view links
// to. Views without links are left alone.
func (inst *AppContext) OpenLink(focused tview.Primitive) {
	if inst.focusedLinks != nil && inst.focusedLinks.holds(focused) {
		inst.focusedLinks.openSelectedLink()
	}
}

// tableSaver is a table that can be saved to a csv file
type tableSaver interface {
	holds(view tview.Primitive) bool
	defaultCsvPath() string
	saveCsv(filename string, done func(savedPath string, err error))
}

// SaveTable saves the table that has the focus to a csv file, to the path
// proposed for it when the filename is empty, and then calls done on the UI
// goroutine. It is an error if the focus is not on a table.
func (inst *AppContext) SaveTable(
	focused tview.Primitive, filename string, done func(savedPath string, err error),
) error {
	if inst.focusedTable == nil || !inst.focusedTable.holds(focused) {
		return fmt.Errorf("the selected pane is not a table")
	}
	if strings.TrimSpace(filename) == "" {
		filename = inst.focusedTable.defaultCsvPath()
	}
	inst.focusedTable.saveCsv(filename, done)
	return nil
}

func (inst *AppContext) GetApiClients() *awsapi.AwsApiClients {
	return awsapi.GetAwsApiClients()
}

// ResetApiClients replaces the api clients and then tells the listeners, so
// they can drop whatever they loaded with the old clients. Call it on the UI
// goroutine as the listeners change views.
func (inst *AppContext) ResetApiClients(cfg aws.Config, profile string) {
	awsapi.ResetAwsApiClients(cfg, profile)

	// A copy is used as listeners can remove themselves, such as when the
	// views they belong to are replaced
	inst.listenersMtx.Lock()
	var listeners = slices.Clone(inst.apiResetListeners)
	inst.listenersMtx.Unlock()

	for _, listener := range listeners {
		(*listener)()
	}
}

// OnApiClientsReset adds a listener and returns a func that removes it
func (inst *AppContext) OnApiClientsReset(listener func()) (remove func()) {
	var entry = &listener

	inst.listenersMtx.Lock()
	defer inst.listenersMtx.Unlock()
	inst.apiResetListeners = append(inst.apiResetListeners, entry)

	return func() {
		inst.listenersMtx.Lock()
		defer inst.listenersMtx.Unlock()
		inst.apiResetListeners = slices.DeleteFunc(inst.apiResetListeners, func(l *func()) bool {
			return l == entry
		})
	}
}

// SetTheme changes the colours of the app and then tells the listeners, so they
// can build the views again. Views take their colours when they are built,
// so the ones that exist keep the old colours until then. Call it on the UI
// goroutine as the listeners change views.
func (inst *AppContext) SetTheme(theme AppTheme) {
	*inst.Theme = theme
	inst.Theme.ResetGlobalStyle()
	for _, listener := range inst.themeListeners {
		listener()
	}
}

func (inst *AppContext) OnThemeChanged(listener func()) {
	inst.themeListeners = append(inst.themeListeners, listener)
}

// KeyBindingsChanged tells the listeners that APP_KEY_BINDINGS changed, so the
// views that took their keys when they were built can be built again. It is
// called on the UI goroutine.
func (inst *AppContext) KeyBindingsChanged() {
	for _, listener := range inst.keyListeners {
		listener()
	}
}

func (inst *AppContext) OnKeyBindingsChanged(listener func()) {
	inst.keyListeners = append(inst.keyListeners, listener)
}

func NewAppContext(
	ctx context.Context, app *tview.Application, config *aws.Config, logger *log.Logger, theme *AppTheme,
) *AppContext {
	return &AppContext{
		Ctx:       ctx,
		App:       app,
		Logger:    logger,
		Theme:     theme,
		clipboard: newAppClipboard(),
	}
}

type ServiceContext[AwsApi any] struct {
	*AppContext
	Api AwsApi
}

func NewServiceViewContext[AwsApi any](
	appContext *AppContext, api AwsApi,
) *ServiceContext[AwsApi] {
	return &ServiceContext[AwsApi]{
		AppContext: appContext,
		Api:        api,
	}
}

type PrivateDataTable[T any, U any] interface {
	GetPrivateData(row int, column int) T
	SetSelectionChangedFunc(handler func(row int, column int)) U
	SetSearchText(text string)
	GetSearchText() string
}

func CreateJsonTableDataView[T any, U any](
	appCtx *AppContext,
	table PrivateDataTable[T, U],
	fixedColIdx int,
) *SearchableTextView {
	var expandedView = NewSearchableTextView("Message", appCtx)

	table.SetSelectionChangedFunc(func(row, column int) {
		var col = column
		if fixedColIdx > -1 {
			col = fixedColIdx
		}

		var privateData = any(table.GetPrivateData(row, col))
		var anyJson any

		switch pd := privateData.(type) {
		case string:
			if err := json.Unmarshal([]byte(pd), &anyJson); err != nil {
				expandedView.SetText(pd, false)
				expandedView.SetSearchText(table.GetSearchText())
				return
			}
		case map[string]any:
			anyJson = pd
		default:
			var text = fmt.Sprintf("%v", privateData)
			expandedView.SetText(text, false)
			expandedView.SetSearchText(table.GetSearchText())
			return
		}

		var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
		expandedView.SetText(string(jsonBytes), false)
		expandedView.SetSearchText(table.GetSearchText())
	})

	return expandedView
}

type JsonTextView[T any] struct {
	TextView        *SearchableTextView
	ExtractTextFunc func(data T) string
	// Syntax highlighting is on by default; set this for views that scroll
	// through large amounts of json.
	DisableHighlighting bool
}

func (inst *JsonTextView[T]) SetText(data T) {
	var anyJson map[string]any

	var highlight = !inst.DisableHighlighting
	if inst.TextView.JsonHighlighting() != highlight {
		inst.TextView.SetJsonHighlighting(highlight)
	}

	var logText = inst.ExtractTextFunc(data)
	var err = json.Unmarshal([]byte(logText), &anyJson)
	if err != nil {
		inst.TextView.SetText(logText, false)
		return
	}
	var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
	logText = string(jsonBytes)
	inst.TextView.SetText(logText, false)
}

func (inst *JsonTextView[T]) SetTitle(title string) {
	inst.TextView.SetTitle(title)
}

type DropDown struct {
	*tview.DropDown
	appTheme *AppTheme
}

func NewDropDown(appTheme *AppTheme) *DropDown {
	var view = &DropDown{
		DropDown: tview.NewDropDown(),
		appTheme: appTheme,
	}
	var blurStyle = appTheme.GetBlurFormItemStyle()
	var focusStyle = appTheme.GetFocusFormItemStyle()

	view.DropDown.
		SetListStyles(blurStyle, focusStyle).
		SetFieldWidth(500).
		SetBlurFunc(func() {
			var fg, bg, _ = blurStyle.Decompose()
			view.DropDown.SetLabelColor(fg)
			view.DropDown.SetBackgroundColor(bg)
		}).
		SetFocusFunc(func() {
			var fg, bg, _ = focusStyle.Decompose()
			view.DropDown.SetLabelColor(fg)
			view.DropDown.SetBackgroundColor(bg)
		})

	return view
}

type InputField struct {
	*tview.InputField
	appTheme *AppTheme
}

func NewInputField(appTheme *AppTheme) *InputField {
	var view = &InputField{
		InputField: tview.NewInputField(),
		appTheme:   appTheme,
	}

	view.InputField.
		SetPlaceholderTextColor(appTheme.PlaceholderTextColour).
		SetBlurFunc(func() {
			view.InputField.SetLabelStyle(appTheme.GetBlurFormItemStyle())
		}).
		SetFocusFunc(func() {
			view.InputField.SetLabelStyle(appTheme.GetFocusFormItemStyle())
		})

	return view
}

type Button struct {
	*tview.Button
	appTheme *AppTheme
}

func NewButton(label string, appTheme *AppTheme) *Button {
	var view = &Button{
		Button:   tview.NewButton(label),
		appTheme: appTheme,
	}

	view.Button.
		SetActivatedStyle(appTheme.GetFocusFormItemStyle()).
		SetStyle(tcell.Style{}.
			Background(appTheme.ContrastBackgroundColor).
			Foreground(appTheme.PrimaryTextColour),
		)
	return view
}

type DateTimeInputField struct {
	*tview.InputField
	appTheme *AppTheme
	layout   string
	// location is the time zone the text is written and read in
	location *time.Location
}

func NewDateTimeInputField(appTheme *AppTheme) *DateTimeInputField {
	var view = &DateTimeInputField{
		InputField: tview.NewInputField(),
		appTheme:   appTheme,
		layout:     "2006-01-02 15:04:05",
		location:   time.Local,
	}

	view.InputField.
		SetPlaceholderTextColor(appTheme.PlaceholderTextColour).
		SetPlaceholder(view.layout).
		SetBlurFunc(func() {
			view.InputField.SetLabelStyle(appTheme.GetBlurFormItemStyle())
		}).
		SetFocusFunc(func() {
			view.InputField.SetLabelStyle(appTheme.GetFocusFormItemStyle())
		})

	var pattern = regexp.MustCompile(`\d|-|\s|:`)

	view.InputField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if len(textToCheck) == 0 || len(textToCheck) > len(view.layout) {
			return false
		}

		return pattern.Match([]byte{byte(lastChar)})
	})

	return view
}

// ValidateInput reads the text as a time in the local time zone, which is the
// zone SetTextTime writes it in.
func (inst *DateTimeInputField) ValidateInput() (time.Time, error) {
	var input = inst.GetText()
	return time.ParseInLocation(inst.layout, input, inst.location)
}

// SetTextTime shows a time in the local time zone, whatever zone it is in.
func (inst *DateTimeInputField) SetTextTime(datetime time.Time) {
	inst.SetText(datetime.In(inst.location).Format(inst.layout))
}

type OverlayView interface {
	tview.Primitive
	GetLastFocusedView() tview.Primitive
}

type OverlayInfo struct {
	Id               string
	View             OverlayView
	IsHidden         bool
	KeyRune          rune
	Keybinding       tcell.Key
	Toggle           bool
	InputCaptureFunc func(event *tcell.EventKey) *tcell.EventKey
}

type BaseView struct {
	*tview.Pages
	mainPageView tview.Primitive
	overlays     map[string]*OverlayInfo
	appCtx       *AppContext
}

func NewBaseView(appContext *AppContext) *BaseView {
	var view = &BaseView{
		Pages:    tview.NewPages(),
		overlays: map[string]*OverlayInfo{},
		appCtx:   appContext,
	}

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		for _, id := range view.GetPageNames(false) {
			var overlay, ok = view.overlays[id]
			if !ok {
				continue
			}

			if e := overlay.InputCaptureFunc(event); e == nil {
				return nil
			} else {
				event = e
			}
		}

		return event
	})

	return view
}

func (inst *BaseView) SetMainView(view tview.Primitive) *BaseView {
	inst.mainPageView = view
	inst.AddAndSwitchToPage("MAIN_PAGE", view, true)
	inst.SendToBack("MAIN_PAGE")
	return inst
}

// This will overwrite the input capture handler of the view passed in.
func (inst *BaseView) AddRuneToggleOverlay(
	id string, view OverlayView, keybinding rune, toggle bool,
) *BaseView {
	inst.AddPage(id, view, true, false)
	var overlay = &OverlayInfo{
		Id:         id,
		View:       view,
		IsHidden:   true,
		KeyRune:    keybinding,
		Keybinding: -1,
		Toggle:     toggle,
	}

	overlay.InputCaptureFunc = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Escape:
			if overlay.IsHidden == false {
				inst.hideOverlay(overlay)
				return nil
			}
		}

		switch event.Rune() {
		case overlay.KeyRune:
			if overlay.IsHidden && inst.IsAnOverlayVisible() == false {
				inst.showOverlay(overlay)
				return nil
			} else if overlay.Toggle && !overlay.IsHidden {
				inst.hideOverlay(overlay)
				return nil
			}
		}
		return event
	}

	inst.overlays[id] = overlay

	return inst
}

// This will overwrite the input capture handler of the view passed in.
func (inst *BaseView) AddKeyToggleOverlay(
	id string, view OverlayView, keybinding tcell.Key, toggle bool,
) *BaseView {
	inst.AddPage(id, view, true, false)
	var overlay = &OverlayInfo{
		Id:         id,
		View:       view,
		IsHidden:   true,
		KeyRune:    0,
		Keybinding: keybinding,
		Toggle:     toggle,
	}

	overlay.InputCaptureFunc = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Escape:
			if overlay.IsHidden == false {
				inst.hideOverlay(overlay)
				return nil
			}
		case overlay.Keybinding:
			if overlay.IsHidden && inst.IsAnOverlayVisible() == false {
				inst.showOverlay(overlay)
				return nil
			} else if overlay.Toggle && !overlay.IsHidden {
				inst.hideOverlay(overlay)
				return nil
			}
		}
		return event
	}

	inst.overlays[id] = overlay

	return inst
}

func (inst *BaseView) IsAnOverlayVisible() bool {
	for _, overlay := range inst.overlays {
		if overlay.IsHidden == false {
			return true
		}
	}
	return false
}

func (inst *BaseView) HideAllOverlays() {
	for _, overlay := range inst.overlays {
		inst.hideOverlay(overlay)
	}
}

func (inst *BaseView) IsOverlayHidden(id string) bool {
	var overlay, ok = inst.overlays[id]
	if ok {
		return overlay.IsHidden
	}
	return true
}

func (inst *BaseView) ToggleOverlay(id string, hide bool) {
	var overlay, ok = inst.overlays[id]
	if ok {
		if hide {
			inst.hideOverlay(overlay)
		} else {
			inst.HideAllOverlays()
			inst.showOverlay(overlay)
		}
	}
}

func (inst *BaseView) showOverlay(overlay *OverlayInfo) {
	inst.SendToFront(overlay.Id)
	inst.ShowPage(overlay.Id)
	if shown, ok := overlay.View.(interface{ OnShow() }); ok {
		shown.OnShow()
	}
	inst.appCtx.App.SetFocus(overlay.View.GetLastFocusedView())
	overlay.IsHidden = false
}

func (inst *BaseView) hideOverlay(overlay *OverlayInfo) {
	inst.HidePage(overlay.Id)
	overlay.IsHidden = true
}
