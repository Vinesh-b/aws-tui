package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ConfirmView asks the user to confirm an action. Focus starts on the cancel
// button so a stray Enter doesn't confirm.
type ConfirmView struct {
	*tview.Flex
	message       *tview.TextView
	confirmButton *Button
	cancelButton  *Button
	navigator     *ViewNavigation2D
	onShow        func()
}

func NewConfirmView(appCtx *AppContext, confirmLabel string) *ConfirmView {
	var layout = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &ConfirmView{
		Flex:          layout,
		message:       tview.NewTextView().SetDynamicColors(false).SetWordWrap(true),
		confirmButton: NewButton(confirmLabel, appCtx.Theme),
		cancelButton:  NewButton("Cancel", appCtx.Theme),
		onShow:        func() {},
	}

	layout.
		AddItem(view.message, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.confirmButton, 0, 1, true).
				AddItem(view.cancelButton, 0, 1, true),
			1, 0, true,
		)

	view.navigator = NewViewNavigation2D(
		layout, [][]View{{view.cancelButton, view.confirmButton}}, appCtx.App,
	)

	return view
}

func (inst *ConfirmView) SetMessage(text string) {
	inst.message.SetText(text)
}

func (inst *ConfirmView) GetMessage() string {
	return inst.message.GetText(true)
}

// SetOnConfirmFunc sets what happens when the action is confirmed. The view
// stays open, so the handler closes it.
func (inst *ConfirmView) SetOnConfirmFunc(handler func()) {
	inst.confirmButton.SetSelectedFunc(handler)
}

func (inst *ConfirmView) SetOnCancelFunc(handler func()) {
	inst.cancelButton.SetSelectedFunc(handler)
}

// SetOnShowFunc sets a function called each time the view is shown, to fill in
// what is being confirmed.
func (inst *ConfirmView) SetOnShowFunc(handler func()) {
	inst.onShow = handler
}

func (inst *ConfirmView) OnShow() {
	inst.navigator.SetFocusedRow(0)
	inst.onShow()
}

func (inst *ConfirmView) GetLastFocusedView() tview.Primitive {
	return inst.navigator.GetLastFocusedView()
}

type FloatingConfirmView struct {
	*tview.Flex
	Input *ConfirmView
}

func NewFloatingConfirmView(appCtx *AppContext, title string, confirmLabel string) *FloatingConfirmView {
	var input = NewConfirmView(appCtx, confirmLabel)

	return &FloatingConfirmView{
		Flex:  FloatingView(title, input, 60, 7),
		Input: input,
	}
}

func (inst *FloatingConfirmView) OnShow() {
	inst.Input.OnShow()
}

func (inst *FloatingConfirmView) GetLastFocusedView() tview.Primitive {
	return inst.Input.GetLastFocusedView()
}

func (inst *FloatingConfirmView) NavigateWithin(key tcell.Key) bool {
	return inst.Input.navigator.NavigateWithin(key)
}

func (inst *FloatingConfirmView) FocusFrom(key tcell.Key) {
	inst.Input.navigator.FocusFrom(key)
}
