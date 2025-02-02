package core

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

type StringSet map[string]struct{}

func ClampStringLen(input *string, maxLen int) string {
	if len(*input) < maxLen {
		return *input
	}
	return (*input)[0:maxLen-1] + "…"
}

func TryFormatToJson(text string) (string, bool) {
	var anyJson map[string]interface{}
	var err = json.Unmarshal([]byte(text), &anyJson)
	if err != nil {
		return text, false
	}

	var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")

	return string(jsonBytes), true
}

type PrivateDataTable[T any, U any] interface {
	GetPrivateData(row int, column int) T
	SetSelectionChangedFunc(handler func(row int, column int)) U
	SetSearchText(text string)
	GetSearchText() string
}

func CreateJsonTableDataView[T any, U any](
	app *tview.Application,
	table PrivateDataTable[T, U],
	fixedColIdx int,
) *SearchableTextView {
	var expandedView = NewSearchableTextView("Message", app)

	table.SetSelectionChangedFunc(func(row, column int) {
		var col = column
		if fixedColIdx > -1 {
			col = fixedColIdx
		}

		var privateData = any(table.GetPrivateData(row, col))
		var anyJson any

		switch privateData.(type) {
		case string:
			var text = privateData.(string)
			if err := json.Unmarshal([]byte(text), &anyJson); err != nil {
				expandedView.SetText(text, false)
				expandedView.SetSearchText(table.GetSearchText())
				return
			}
		case map[string]interface{}:
			anyJson = privateData.(map[string]interface{})
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
}

func (inst *JsonTextView[T]) SetText(data T) {
	var anyJson map[string]interface{}

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

func FuzzySearch[T any](search string, values []T, handler func(val T) string) []T {
	if len(values) == 0 {
		return nil
	}

	if len(search) == 0 {
		return values
	}

	var names = make([]string, 0, len(values))
	for _, v := range values {
		names = append(names, handler(v))
	}

	var matches = fuzzy.RankFindFold(search, names)
	sort.Sort(matches)

	var results = make([]int, 0, len(matches))
	for _, m := range matches {
		results = append(results, m.OriginalIndex)
	}

	var found = []T{}
	for _, matchIdx := range results {
		found = append(found, values[matchIdx])
	}

	return found
}

type DropDown struct {
	*tview.DropDown
}

func NewDropDown() *DropDown {
	var view = &DropDown{
		DropDown: tview.NewDropDown(),
	}

	view.DropDown.
		SetListStyles(OnBlurStyle, OnFocusStyle).
		SetFieldWidth(500).
		SetBlurFunc(func() {
			var fg, bg, _ = OnBlurStyle.Decompose()
			view.DropDown.SetLabelColor(fg)
			view.DropDown.SetBackgroundColor(bg)
		}).
		SetFocusFunc(func() {
			var fg, bg, _ = OnFocusStyle.Decompose()
			view.DropDown.SetLabelColor(fg)
			view.DropDown.SetBackgroundColor(bg)
		})

	return view
}

type InputField struct {
	*tview.InputField
}

func NewInputField() *InputField {
	var view = &InputField{
		InputField: tview.NewInputField(),
	}

	view.InputField.
		SetPlaceholderTextColor(PlaceHolderTextColor).
		SetBlurFunc(func() {
			view.InputField.SetLabelStyle(OnBlurStyle)
		}).
		SetFocusFunc(func() {
			view.InputField.SetLabelStyle(OnFocusStyle)
		})

	return view
}

type Button struct {
	*tview.Button
}

func NewButton(label string) *Button {
	var view = &Button{
		Button: tview.NewButton(label),
	}

	view.Button.
		SetActivatedStyle(OnFocusStyle).
		SetStyle(tcell.Style{}.
			Background(ContrastBackgroundColor).
			Foreground(TextColour),
		)
	return view
}

type DateTimeInputField struct {
	*tview.InputField
	layout string
}

func NewDateTimeInputField() *DateTimeInputField {
	var view = &DateTimeInputField{
		InputField: tview.NewInputField(),
		layout:     "2006-01-02 15:04:05",
	}

	view.InputField.
		SetPlaceholderTextColor(PlaceHolderTextColor).
		SetPlaceholder(view.layout).
		SetBlurFunc(func() {
			view.InputField.SetLabelStyle(OnBlurStyle)
		}).
		SetFocusFunc(func() {
			view.InputField.SetLabelStyle(OnFocusStyle)
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

func (inst *DateTimeInputField) ValidateInput() (time.Time, error) {
	var input = inst.GetText()
	return time.Parse(inst.layout, input)
}

func (inst *DateTimeInputField) SetTextTime(datetime time.Time) {
	inst.SetText(datetime.Format(inst.layout))
}

type OverlayView interface {
	tview.Primitive
	GetLastFocusedView() tview.Primitive
}

type OverlayInfo struct {
	Id               string
	View             OverlayView
	IsHidden         bool
	ToggleRune       rune
	ToggleKey        tcell.Key
	InputCaptureFunc func(event *tcell.EventKey) *tcell.EventKey
}

type BaseView struct {
	*tview.Pages
	app          *tview.Application
	logger       *log.Logger
	mainPageView tview.Primitive
	overlays     map[string]*OverlayInfo
}

func NewBaseView(app *tview.Application, logger *log.Logger) *BaseView {
	var view = &BaseView{
		Pages:    tview.NewPages(),
		app:      app,
		logger:   logger,
		overlays: map[string]*OverlayInfo{},
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
func (inst *BaseView) AddRuneToggleOverlay(id string, view OverlayView, viewToggle rune) *BaseView {
	inst.AddPage(id, view, true, false)
	var overlay = &OverlayInfo{
		Id:         id,
		View:       view,
		IsHidden:   true,
		ToggleRune: viewToggle,
		ToggleKey:  -1,
	}

	overlay.InputCaptureFunc = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Escape:
			if overlay.IsHidden == false {
				inst.HidePage(overlay.Id)
				overlay.IsHidden = true
				return nil
			}
		}

		switch event.Rune() {
		case overlay.ToggleRune:
			if overlay.IsHidden {
				inst.SendToFront(overlay.Id)
				inst.ShowPage(overlay.Id)
				inst.app.SetFocus(view.GetLastFocusedView())
			} else {
				inst.HidePage(overlay.Id)
			}
			overlay.IsHidden = !overlay.IsHidden
			return nil
		}
		return event
	}

	inst.overlays[id] = overlay

	return inst
}

// This will overwrite the input capture handler of the view passed in.
func (inst *BaseView) AddKeyToggleOverlay(id string, view OverlayView, viewToggle tcell.Key) *BaseView {
	inst.AddPage(id, view, true, false)
	var overlay = &OverlayInfo{
		Id:         id,
		View:       view,
		IsHidden:   true,
		ToggleRune: 0,
		ToggleKey:  viewToggle,
	}

	overlay.InputCaptureFunc = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Escape:
			if overlay.IsHidden == false {
				inst.HidePage(overlay.Id)
				overlay.IsHidden = true
				return nil
			}
		case overlay.ToggleKey:
			if overlay.IsHidden {
				inst.SendToFront(overlay.Id)
				inst.ShowPage(overlay.Id)
				inst.app.SetFocus(view.GetLastFocusedView())
			} else {
				inst.HidePage(overlay.Id)
			}
			overlay.IsHidden = !overlay.IsHidden
			return nil
		}
		return event
	}

	inst.overlays[id] = overlay

	return inst
}

func (inst *BaseView) HideAllOverlays() {
	for _, overlay := range inst.overlays {
		inst.HidePage(overlay.Id)
		overlay.IsHidden = true
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
		overlay.IsHidden = hide
		if hide {
			inst.HidePage(overlay.Id)
		} else {
			inst.ShowPage(overlay.Id)
			inst.app.SetFocus(overlay.View.GetLastFocusedView())
		}
	}
}
