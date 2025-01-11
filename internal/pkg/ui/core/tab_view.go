package core

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TabView struct {
	*tview.Flex
	tabNames []string
	list     *tview.List
	pages    *tview.Pages
	tabs     map[string]*ServicePageView
	app      *tview.Application
	logger   *log.Logger
}

func NewTabView(tabs []string, app *tview.Application, logger *log.Logger) *TabView {
	var view = &TabView{
		Flex:     tview.NewFlex(),
		tabNames: tabs,
		list:     tview.NewList(),
		pages:    tview.NewPages(),
		tabs:     map[string]*ServicePageView{},
		app:      app,
		logger:   logger,
	}

	view.list.
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(TextColour).
		SetSelectedTextColor(InverseTextColor)
	view.list.
		SetBorder(true).
		SetTitle("Tabs").
		SetTitleAlign(tview.AlignLeft)

	view.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = view.list.GetCurrentItem()
		var numItems = view.list.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				view.list.SetCurrentItem(currentIdx)
				view.list.GetItemSelectedFunc(currentIdx)()
				return nil
			case APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				view.list.SetCurrentItem(currentIdx)
				view.list.GetItemSelectedFunc(currentIdx)()
				return nil
			}
		}

		return event
	})

	for _, name := range view.tabNames {
		var servicePage = NewServicePageView(app, logger)
		view.pages.AddPage(name, servicePage, true, true)
		view.tabs[name] = servicePage

		view.list.AddItem(name, "", 0, func() {
			view.pages.SwitchToPage(name)
		})
	}

	view.pages.SwitchToPage(tabs[0])
	view.Flex.SetDirection(tview.FlexColumn).
		AddItem(view.list, 18, 0, true).
		AddItem(view.pages, 0, 1, true)

	return view
}

func (inst *TabView) GetTabsList() *tview.List {
	return inst.list
}

func (inst *TabView) GetTab(name string) *ServicePageView {
	if tab, ok := inst.tabs[name]; ok {
		return tab
	}
	return nil
}

func (inst *TabView) GetTabDisplayView() *tview.Pages {
	return inst.pages
}
