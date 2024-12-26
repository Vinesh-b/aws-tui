package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Todo add timeout param
func LoadData(
	app *tview.Application,
	view *tview.Box,
	resultChannel chan struct{},
	updateViewFunc func(),
) {
	var (
		idx           = 0
		originalTitle = view.GetTitle()
		loadingSymbol = [8]string{"⢎⡰", "⢎⡡", "⢎⡑", "⢎⠱", "⠎⡱", "⢊⡱", "⢌⡱", "⢆⡱"}

		timeoutCtx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	)
	defer cancel()

	for {
		select {
		case <-resultChannel:
			app.QueueUpdateDraw(updateViewFunc)
			return
		case <-timeoutCtx.Done():
			app.QueueUpdateDraw(func() {
				view.SetTitle("Timed out")
			})
			return
		default:
			app.QueueUpdateDraw(func() {
				view.SetTitle(fmt.Sprintf(originalTitle+"%s", loadingSymbol[idx]))
			})
			idx = (idx + 1) % len(loadingSymbol)
			time.Sleep(time.Millisecond * 100)
		}
	}
}

type TableCreationParams struct {
	App    *tview.Application
	Logger *log.Logger
}

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

func CreateExpandedLogView(
	app *tview.Application,
	table *tview.Table,
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

func InitViewTabNavigation(rootView RootView, orderedViews []View, app *tview.Application) {
	// Sets current view index when selected
	var viewIdx = len(orderedViews)
	var numViews = len(orderedViews)
	rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBacktab:
			viewIdx = (viewIdx - 1 + numViews) % numViews
			app.SetFocus(orderedViews[viewIdx])
			return nil
		case tcell.KeyTab:
			viewIdx = (viewIdx + 1) % numViews
			app.SetFocus(orderedViews[viewIdx])
			return nil
		}

		return event
	})
}

