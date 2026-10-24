package core

import (
	"github.com/rivo/tview"
)

type MessagePromptView struct {
	*tview.Flex
	textView *tview.TextView
	button   *tview.Button
}

func NewMessagePromptView(app *tview.Application) *MessagePromptView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var textView = tview.NewTextView()
	var button = tview.NewButton("OK").
		SetSelectedFunc(func() {

		})

	var view = &MessagePromptView{
		Flex:     flex,
		textView: textView,
		button:   button,
	}

	view.
		AddItem(textView, 0, 1, false).
		AddItem(button, 1, 0, true)

	return view
}

func (inst *MessagePromptView) SetText(text string) {
	inst.textView.SetText(text)
}

func (inst *MessagePromptView) SetSelectedFunc(handler func()) {
	inst.button.SetSelectedFunc(handler)
}
