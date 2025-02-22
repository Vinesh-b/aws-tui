package core

import (
	"encoding/json"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TextArea struct {
	*tview.TextArea
	ErrorMessageCallback func(text string, a ...any)
	appTheme             *AppTheme
	lineCount            int
	title                string
	titleExtra           string
}

func NewTextArea(title string, appTheme *AppTheme) *TextArea {
	var t = tview.NewTextArea()
	var view = &TextArea{
		TextArea:             t,
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

func (inst *TextArea) SetTitleExtra(text string) {
	inst.titleExtra = text
	inst.SetTitle(fmt.Sprintf("%s ❬%s❭", inst.title, inst.titleExtra))
}

func (inst *TextArea) FormatAsJson() {
	var payload = make(map[string]any)
	if err := json.Unmarshal([]byte(inst.GetText()), &payload); err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	var jsonPayload, err = json.MarshalIndent(payload, "", "  ")
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
	}

	inst.SetText(string(jsonPayload), false)
}
