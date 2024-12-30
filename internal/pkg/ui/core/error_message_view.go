package core

import (
	"github.com/rivo/tview"
)

type ErrorMessageView struct {
	*tview.Flex
	textView *tview.TextView
	button   *tview.Button
}

func NewErrorMessageView(app *tview.Application) *ErrorMessageView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var textView = tview.NewTextView()
	var button = tview.NewButton("OK").
		SetSelectedFunc(func() {

		})

	var view = &ErrorMessageView{
		Flex:     flex,
		textView: textView,
		button:   button,
	}

	view.
		AddItem(textView, 0, 1, false).
		AddItem(button, 1, 0, true)

	return view
}

func (inst *ErrorMessageView) SetText(text string) {
	inst.textView.SetText(text)
}

func (inst *ErrorMessageView) SetSelectedFunc(handler func()) {
	inst.button.SetSelectedFunc(handler)
}
