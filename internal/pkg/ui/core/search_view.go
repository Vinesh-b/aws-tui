package core

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchDateTimeView struct {
	InputField     *tview.InputField
	startTimeInput *tview.InputField
	endTimeInput   *tview.InputField
	StartDateTime  time.Time
	EndDateTime    time.Time
	RootView       *tview.Flex
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

	var startDateTime = time.Now()
	var endDateTime = time.Now()

	startTimeInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			var start, err = time.Parse(dateTimelayout, startTimeInput.GetText())
			if err != nil {
				startDateTime = time.Now()
				startTimeInput.SetFieldTextColor(tcell.ColorDarkRed)

			} else {
				startDateTime = start
				startTimeInput.SetFieldTextColor(TextColour)
				app.SetFocus(endTimeInput)
			}
		}
		return
	})

	endTimeInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			var end, err = time.Parse(dateTimelayout, endTimeInput.GetText())
			if err != nil {
				endDateTime = time.Now()
				endTimeInput.SetFieldTextColor(tcell.ColorDarkRed)
			} else {
				endDateTime = end
				endTimeInput.SetFieldTextColor(TextColour)
				app.SetFocus(inputField)
			}
		}
		return
	})

	InitViewTabNavigation(wrapper,
		[]View{
			startTimeInput,
			endTimeInput,
			inputField,
		},
		app,
	)

	return &SearchDateTimeView{
		InputField:     inputField,
		startTimeInput: startTimeInput,
		endTimeInput:   endTimeInput,
		StartDateTime:  startDateTime,
		EndDateTime:    endDateTime,
		RootView:       wrapper,
	}
}

func (inst *SearchDateTimeView) SetDoneFunc(handler func(key tcell.Key)) *tview.InputField {
	return inst.InputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if inst.EndDateTime.After(inst.StartDateTime) {
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
		RootView:           FloatingView("", searchView.RootView, width, 5),
	}
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
