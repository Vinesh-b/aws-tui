package core

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchDateTimeView struct {
	*tview.Flex
	inputField     *tview.InputField
	startTimeInput *tview.InputField
	endTimeInput   *tview.InputField
	startDateTime  time.Time
	endDateTime    time.Time
	viewNavigation *ViewNavigation1D
}

func NewSearchDateTimeView(label string, app *tview.Application) *SearchDateTimeView {
	var inputField = tview.NewInputField().
		SetLabel(fmt.Sprintf("%s ", label)).
		SetFieldWidth(0)

	var dateTimelayout = "2006-01-02 15:04:05"
	var startTimeInput = tview.NewInputField().
		SetPlaceholder(dateTimelayout).
		SetPlaceholderTextColor(PlaceHolderTextColor).
		SetLabel("Start Time ")

	var endTimeInput = tview.NewInputField().
		SetPlaceholder(dateTimelayout).
		SetPlaceholderTextColor(PlaceHolderTextColor).
		SetLabel("End Time ")

	var wrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(startTimeInput, 0, 1, true).
				AddItem(endTimeInput, 0, 1, true),
			1, 0, true).
		AddItem(tview.NewBox(), 1, 0, true).
		AddItem(inputField, 1, 0, true)

	var view = &SearchDateTimeView{
		Flex:           wrapper,
		inputField:     inputField,
		startTimeInput: startTimeInput,
		endTimeInput:   endTimeInput,
		startDateTime:  time.Now(),
		endDateTime:    time.Now(),
		viewNavigation: NewViewNavigation1D(
			wrapper,
			[]View{
				startTimeInput,
				endTimeInput,
				inputField,
			},
			app,
		),
	}

	startTimeInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			var start, err = time.Parse(dateTimelayout, startTimeInput.GetText())
			if err != nil {
				view.startDateTime = time.Now()
				startTimeInput.SetFieldTextColor(tcell.ColorDarkRed)

			} else {
				view.startDateTime = start
				startTimeInput.SetFieldTextColor(TextColour)
				app.SetFocus(endTimeInput)
			}
		}
		return
	})

	endTimeInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			var end, err = time.Parse(dateTimelayout, endTimeInput.GetText())
			if err != nil {
				view.endDateTime = time.Now()
				endTimeInput.SetFieldTextColor(tcell.ColorDarkRed)
			} else {
				view.endDateTime = end
				endTimeInput.SetFieldTextColor(TextColour)
				app.SetFocus(inputField)
			}
		}
		return
	})

	return view
}

func (inst *SearchDateTimeView) SetDoneFunc(handler func(key tcell.Key)) *tview.InputField {
	return inst.inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			if inst.endDateTime.After(inst.startDateTime) {
				handler(key)
			}
		}
	})
}

type FloatingSearchDateTimeView struct {
	*SearchDateTimeView
	RootView *tview.Flex
}

func NewFloatingSearchDateTimeView(
	label string,
	width int,
	app *tview.Application,
) *FloatingSearchDateTimeView {
	var searchView = NewSearchDateTimeView(label, app)

	return &FloatingSearchDateTimeView{
		SearchDateTimeView: searchView,
		RootView:           FloatingView("", searchView, width, 5),
	}
}

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
