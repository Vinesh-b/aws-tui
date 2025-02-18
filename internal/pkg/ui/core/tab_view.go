package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TabView struct {
	*tview.Flex
	list       *tview.List
	pages      *tview.Pages
	pageIdxMap map[string]int
	tabs       map[string]*ServicePageView
	appCtx     *AppContext
}

func NewTabView(appCtx *AppContext) *TabView {
	var view = &TabView{
		Flex:       tview.NewFlex(),
		list:       tview.NewList(),
		pages:      tview.NewPages(),
		pageIdxMap: map[string]int{},
		tabs:       map[string]*ServicePageView{},
		appCtx:     appCtx,
	}

	view.list.
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(appCtx.Theme.PrimaryTextColour).
		SetSelectedTextColor(appCtx.Theme.InverseTextColour)
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

	view.list.SetBorderPadding(0, 0, 1, 1)
	view.Flex.SetDirection(tview.FlexColumn).
		AddItem(view.list, 20, 0, true).
		AddItem(view.pages, 0, 1, true)

	return view
}

func (inst *TabView) GetTabsList() *tview.List {
	return inst.list
}

func (inst *TabView) AddTab(
	name string, view tview.Primitive, fixedSize int, proportion int, focus bool,
) *TabView {
	var servicePage = NewServicePageView(inst.appCtx)
	servicePage.MainPage.AddItem(view, fixedSize, proportion, focus)
	inst.pages.AddPage(name, servicePage, true, false)
	inst.tabs[name] = servicePage

	inst.pageIdxMap[name] = inst.list.GetItemCount()
	inst.list.AddItem(name, "", 0, func() {
		inst.pages.SwitchToPage(name)
	})
	return inst
}

func (inst *TabView) AddAndSwitchToTab(
	name string, view tview.Primitive, fixedSize int, proportion int, focus bool,
) *TabView {
	inst.AddTab(name, view, fixedSize, proportion, focus)
	inst.pages.SwitchToPage(name)
	return inst
}

func (inst *TabView) SwitchToTab(name string) *TabView {
	inst.pages.SwitchToPage(name)
	var idx = inst.pageIdxMap[name]
	inst.list.SetCurrentItem(idx)
	return inst
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
