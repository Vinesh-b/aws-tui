package core

import (
	"unicode/utf8"

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

type TabViewHorizontal struct {
	*tview.Flex
	pages           *tview.Pages
	pageIdxMap      map[string]int
	pageViewMap     map[string]*ServicePageView
	orderedTabs     []string
	currentTabIdx   int
	defaultTabIdx   int
	onTabChangeFunc func(tabName string, index int)
	appCtx          *AppContext
}

func NewTabViewHorizontal(appCtx *AppContext) *TabViewHorizontal {
	var view = &TabViewHorizontal{
		Flex:          tview.NewFlex(),
		pages:         tview.NewPages(),
		pageIdxMap:    map[string]int{},
		pageViewMap:   map[string]*ServicePageView{},
		orderedTabs:   []string{},
		currentTabIdx: 0,
		appCtx:        appCtx,
	}

	view.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			view.currentTabIdx = (view.currentTabIdx + 1) % len(view.orderedTabs)
			view.SwitchToTab(view.orderedTabs[view.currentTabIdx])
			return nil
		case tcell.KeyBacktab:
			view.currentTabIdx = (view.currentTabIdx - 1 + len(view.orderedTabs)) % len(view.orderedTabs)
			view.SwitchToTab(view.orderedTabs[view.currentTabIdx])
			return nil
		}
		return event
	})

	view.Flex.SetDirection(tview.FlexRow).
		AddItem(view.pages, 0, 1, true)

	var TAB_BAR_OFFSET = 1
	var TAB_LEFT = "🭃"
	var TAB_RIGHT = "🭎"

	var tabActiveStyle = tcell.StyleDefault.
		Background(appCtx.Theme.MoreContrastBackgroundColor).
		Foreground(appCtx.Theme.SecondaryTextColour)

	var tabInactiveStyle = tcell.StyleDefault.
		Background(appCtx.Theme.ContrastBackgroundColor).
		Foreground(appCtx.Theme.PrimaryTextColour)

	view.pages.SetDrawFunc(
		func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
			var rectX, rectY, rectW, rectH = view.pages.GetRect()
			var tabPos = rectX + TAB_BAR_OFFSET

			for idx, tabName := range view.orderedTabs {
				var tabStyle = tabInactiveStyle

				if idx == view.currentTabIdx {
					tabStyle = tabActiveStyle
				}

				var textLen = utf8.RuneCountInString(tabName)
				var leftPadLen = utf8.RuneCountInString(TAB_LEFT)
				var rightPadLen = utf8.RuneCountInString(TAB_RIGHT)

				for x := range textLen {
					// Fill the tab name text background
					screen.SetContent(tabPos+leftPadLen+x, rectY, ' ', nil, tabStyle)
				}

				var fg, bg, _ = tabStyle.Decompose()

				tview.Print(screen, TAB_LEFT, tabPos, rectY, rectW, tview.AlignLeft, bg)
				tabPos += leftPadLen

				tview.Print(screen, tabName, tabPos, rectY, rectW, tview.AlignLeft, fg)
				tabPos += textLen

				tview.Print(screen, TAB_RIGHT, tabPos, rectY, rectW, tview.AlignLeft, bg)
				tabPos += rightPadLen
			}

			return rectX, rectY + 1, rectW, rectH - 1
		},
	)

	view.pages.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {

		if action != tview.MouseLeftClick {
			return action, event
		}

		var x, y = event.Position()

		if !view.pages.InRect(x, y) {
			return action, event
		}

		var rectX, rectY, _, _ = view.pages.GetRect()
		// We are only interested in clicks on the tab bar on the top row
		if y != rectY {
			return action, event
		}

		var tabPos = rectX + TAB_BAR_OFFSET
		for _, tabName := range view.orderedTabs {
			var tabLen = utf8.RuneCountInString(tabName) +
				utf8.RuneCountInString(TAB_LEFT) +
				utf8.RuneCountInString(TAB_RIGHT)

			if x >= tabPos && x < tabPos+tabLen {
				view.SwitchToTab(tabName)
				return tview.MouseConsumed, nil
			}

			tabPos += tabLen
		}

		return action, event
	})

	return view
}

func (inst *TabViewHorizontal) AddTab(
	name string, view tview.Primitive, fixedSize int, proportion int, focus bool,
) *TabViewHorizontal {
	var servicePage = NewServicePageView(inst.appCtx)
	servicePage.MainPage.AddItem(view, fixedSize, proportion, focus)
	inst.pages.AddPage(name, servicePage, true, false)
	inst.pageViewMap[name] = servicePage
	inst.orderedTabs = append(inst.orderedTabs, name)

	return inst
}

func (inst *TabViewHorizontal) AddAndSwitchToTab(
	name string, view tview.Primitive, fixedSize int, proportion int, focus bool,
) *TabViewHorizontal {
	//Used to set the default Tab
	inst.AddTab(name, view, fixedSize, proportion, focus)
	inst.pages.SwitchToPage(name)
	for i, tabName := range inst.orderedTabs {
		if tabName == name {
			inst.currentTabIdx = i
			inst.defaultTabIdx = i
			break
		}
	}
	return inst
}

func (inst *TabViewHorizontal) SwitchToTab(name string) *TabViewHorizontal {
	inst.pages.SwitchToPage(name)
	for i, tabName := range inst.orderedTabs {
		if tabName == name {
			inst.currentTabIdx = i
			if inst.onTabChangeFunc != nil {
				inst.onTabChangeFunc(name, i)
			}
			break
		}
	}
	return inst
}

func (inst *TabViewHorizontal) GetTab(name string) *ServicePageView {
	if tab, ok := inst.pageViewMap[name]; ok {
		return tab
	}
	return nil
}

func (inst *TabViewHorizontal) GetTabDisplayView() *tview.Pages {
	return inst.pages
}

func (inst *TabViewHorizontal) SetOnTabChangeFunc(f func(tabName string, index int)) {
	inst.onTabChangeFunc = f
}

func (inst *TabViewHorizontal) GetDefaultTab() (string, int) {
	return inst.orderedTabs[inst.defaultTabIdx], inst.defaultTabIdx
}

func (inst *TabViewHorizontal) GetCurrentTab() (string, int) {
	return inst.orderedTabs[inst.currentTabIdx], inst.currentTabIdx
}
