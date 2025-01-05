package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

type StringSet map[string]struct{}

func CreateReadOnlyTextArea(title string) *tview.TextArea {
	var textArea = tview.NewTextArea().
		SetClipboard(
			func(s string) { clipboard.WriteAll(s) },
			func() string {
				var res, _ = clipboard.ReadAll()
				return res
			},
		).
		SetSelectedStyle(
			tcell.Style{}.Background(MoreContrastBackgroundColor),
		)
	textArea.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case // Disable text area default edititing key events
			tcell.KeyCtrlY, tcell.KeyCtrlZ, tcell.KeyCtrlX, tcell.KeyCtrlV,
			tcell.KeyCtrlH, tcell.KeyCtrlD, tcell.KeyCtrlK, tcell.KeyCtrlW,
			tcell.KeyCtrlU, tcell.KeyBackspace2, tcell.KeyDelete, tcell.KeyTab:
			return nil
		}
		if unicode.IsControl(event.Rune()) {
			return event
		}
		switch event.Rune() {
		case APP_KEY_BINDINGS.TextViewCopy:
			return tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)
		case APP_KEY_BINDINGS.TextViewUp:
			return tcell.NewEventKey(tcell.KeyUp, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectUp:
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewDown:
			return tcell.NewEventKey(tcell.KeyDown, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectDown:
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewWordRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl)
		case APP_KEY_BINDINGS.TextViewWordSelectRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl|tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewWordLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl)
		case APP_KEY_BINDINGS.TextViewWordSelectLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl|tcell.ModShift)
		}
		return nil
	})

	return textArea
}

type MessageDataType int

const (
	DATA_TYPE_STRING MessageDataType = iota
	DATA_TYPE_MAP_STRING_ANY
)

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
}

func CreateJsonTableDataView[T any, U any](
	app *tview.Application,
	table PrivateDataTable[T, U],
	fixedColIdx int,
) *tview.TextArea {
	var expandedView = CreateReadOnlyTextArea("Message")

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
				return
			}
		case map[string]interface{}:
			anyJson = privateData.(map[string]interface{})
		default:
			var text = fmt.Sprintf("%v", privateData)
			expandedView.SetText(text, false)
			return
		}

		var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
		expandedView.SetText(string(jsonBytes), false)
	})

	return expandedView
}

type JsonTextView[T any] struct {
	TextArea        *tview.TextArea
	ExtractTextFunc func(data T) string
}

func (inst *JsonTextView[T]) SetText(data T) {
	var anyJson map[string]interface{}

	var logText = inst.ExtractTextFunc(data)
	var err = json.Unmarshal([]byte(logText), &anyJson)
	if err != nil {
		inst.TextArea.SetText(logText, false)
		return
	}
	var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
	logText = string(jsonBytes)
	inst.TextArea.SetText(logText, false)
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
