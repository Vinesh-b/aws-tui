package core

import (
	"encoding/json"
	"fmt"
	"sort"

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
