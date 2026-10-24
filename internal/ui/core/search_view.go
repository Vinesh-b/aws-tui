package core

import (
	"fmt"

	"github.com/rivo/tview"
)

type FloatingSearchView struct {
	*tview.Flex
	InputField *tview.InputField
}

func NewFloatingSearchView(label string, width int, height int) *FloatingSearchView {
	var inputField = tview.NewInputField().
		SetLabel(fmt.Sprintf("%s ", label)).
		SetFieldWidth(0)

	return &FloatingSearchView{
		Flex:       FloatingView("", inputField, width, height),
		InputField: inputField,
	}
}

func (inst *FloatingSearchView) GetLastFocusedView() tview.Primitive {
	return inst
}
