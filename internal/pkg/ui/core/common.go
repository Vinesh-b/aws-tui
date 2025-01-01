package core

import (
	"encoding/json"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StringSet map[string]struct{}

func CreateSearchInput(label string) *tview.InputField {
	var inputField = tview.NewInputField().
		SetLabel(fmt.Sprintf("%s ", label)).
		SetFieldWidth(0)
	inputField.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle("Search").
		SetTitleAlign(tview.AlignLeft)

	return inputField
}

func CreateTextArea(title string) *tview.TextArea {
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

type PrivateDataTable[T any] interface {
	SetSelectionChangedFunc(handler func(row int, column int)) T
	SetSelectedFunc(handler func(row int, column int)) T
	GetCell(row int, column int) *tview.TableCell
}

func CreateExpandedLogView[T any](
	app *tview.Application,
	table PrivateDataTable[T],
	fixedColIdx int,
	dataType MessageDataType,
) *tview.TextArea {
	var expandedView = CreateTextArea("Message")

	table.SetSelectionChangedFunc(func(row, column int) {
		var col = column
		if fixedColIdx >= 0 {
			col = fixedColIdx
		}

		var privateData = table.GetCell(row, col).Reference
		if row < 1 || privateData == nil {
			return
		}

		var anyJson map[string]interface{}
		var logText = ""

		switch dataType {
		case DATA_TYPE_STRING:
			var logText = privateData.(string)
			var err = json.Unmarshal([]byte(logText), &anyJson)
			if err != nil {
				expandedView.SetText(logText, false)
				return
			}
		case DATA_TYPE_MAP_STRING_ANY:
			anyJson = privateData.(map[string]interface{})
		}

		var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
		logText = string(jsonBytes)
		expandedView.SetText(logText, false)
	})

	table.SetSelectedFunc(func(row, column int) {
		app.SetFocus(expandedView)
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

type ViewNavigation struct {
	app          *tview.Application
	rootView     RootView
	orderedViews []View
	viewIdx      int
	numViews     int
	keyForward   tcell.Key
	keyBack      tcell.Key
}

func NewViewNavigation(rootView RootView, orderedViews []View, app *tview.Application) *ViewNavigation {
	var view = &ViewNavigation{
		rootView:     rootView,
		orderedViews: orderedViews,
		app:          app,
		viewIdx:      len(orderedViews),
		numViews:     len(orderedViews),
		keyForward:   tcell.KeyTab,
		keyBack:      tcell.KeyBacktab,
	}

	view.rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case view.keyForward:
			view.viewIdx = (view.viewIdx - 1 + view.numViews) % view.numViews
			view.app.SetFocus(view.orderedViews[view.viewIdx])
			return nil
		case view.keyBack:
			view.viewIdx = (view.viewIdx + 1) % view.numViews
			view.app.SetFocus(view.orderedViews[view.viewIdx])
			return nil
		}

		return event
	})

	return view
}

func (inst *ViewNavigation) SetNavigationKeys(keyForward tcell.Key, keyBack tcell.Key) {
	inst.keyForward = keyForward
	inst.keyBack = keyBack
}

func (inst *ViewNavigation) UpdateOrderedViews(orderedViews []View, intitalIxd int) {
	inst.orderedViews = orderedViews
	inst.numViews = len(inst.orderedViews)
	inst.viewIdx = (intitalIxd + inst.numViews) % inst.numViews
}

func (inst *ViewNavigation) GetLastFocusedView() tview.Primitive {
	return inst.orderedViews[inst.viewIdx]
}
