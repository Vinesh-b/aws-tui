package core

import (
	"encoding/json"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TextArea struct {
	*syntaxTextArea
	ErrorMessageCallback func(text string, a ...any)
	appTheme             *AppTheme
	lineCount            int
	title                string
	titleExtra           string
}

func NewTextArea(title string, appTheme *AppTheme) *TextArea {
	var t = newSyntaxTextArea()
	var view = &TextArea{
		syntaxTextArea:       t,
		ErrorMessageCallback: func(text string, a ...any) {},
		appTheme:             appTheme,
		lineCount:            0,
		title:                title,
		titleExtra:           "",
	}

	view.
		SetClipboard(
			func(s string) { clipboard.WriteAll(s) },
			func() string {
				var res, _ = clipboard.ReadAll()
				return res
			},
		).
		SetSelectedStyle(
			tcell.Style{}.Background(appTheme.MoreContrastBackgroundColor),
		)

	view.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	return view
}

// SetJsonHighlighting colours json keys, strings, numbers and literals as the
// text is typed. It turns line wrapping off.
func (inst *TextArea) SetJsonHighlighting(enabled bool) {
	inst.syntaxTextArea.setJsonHighlighting(enabled)
	if enabled {
		inst.syntaxTextArea.TextArea.SetChangedFunc(inst.syntaxTextArea.refreshLines)
	} else {
		inst.syntaxTextArea.TextArea.SetChangedFunc(nil)
	}
}

// SetText replaces the text, keeping the highlighting up to date.
func (inst *TextArea) SetText(text string, cursorAtTheEnd bool) *tview.TextArea {
	inst.syntaxTextArea.setText(text, cursorAtTheEnd)
	return inst.syntaxTextArea.TextArea
}

func (inst *TextArea) SetTitleExtra(text string) {
	inst.titleExtra = text
	inst.SetTitle(fmt.Sprintf("%s ❬%s❭", inst.title, inst.titleExtra))
}

func (inst *TextArea) FormatAsJson() {
	var payload = make(map[string]any)
	if err := json.Unmarshal([]byte(inst.GetText()), &payload); err != nil {
		inst.ErrorMessageCallback("%s", err.Error())
		return
	}

	var jsonPayload, err = json.MarshalIndent(payload, "", "  ")
	if err != nil {
		inst.ErrorMessageCallback("%s", err.Error())
	}

	inst.SetText(string(jsonPayload), false)
}
