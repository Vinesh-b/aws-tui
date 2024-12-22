package core

import (
	"fmt"

	"github.com/rivo/tview"
)

func FloatingView(title string, p tview.Primitive, width int, height int) *tview.Flex {
	var wrapper = tview.NewFlex().
		AddItem(p, 0, 1, true)
	wrapper.SetBorder(true).SetTitle(title)
	var window = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(wrapper, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)

	return window
}

type FloatingSearchView struct {
	InputField *tview.InputField
	RootView   *tview.Flex
}

func NewFloatingSearchView(label string, width int, height int) *FloatingSearchView {
	var inputField = tview.NewInputField().
		SetLabel(fmt.Sprintf("%s ", label)).
		SetFieldWidth(0)

	return &FloatingSearchView{
		InputField: inputField,
		RootView:   FloatingView("", inputField, width, height),
	}
}
