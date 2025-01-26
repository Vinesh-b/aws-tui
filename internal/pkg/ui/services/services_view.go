package services

import (
	"aws-tui/internal/pkg/ui/core"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ViewId string

const (
	LAMBDA                   ViewId = "Lambda"
	CLOUDWATCH_LOGS_GROUPS   ViewId = "Log Groups"
	CLOUDWATCH_LOGS_INSIGHTS ViewId = "Log Insights"
	CLOUDWATCH_ALARMS        ViewId = "Alarms"
	CLOUDWATCH_METRICS       ViewId = "Metrics"
	CLOUDFORMATION           ViewId = "CloudFormation"
	DYNAMODB                 ViewId = "DynamoDB"
	S3BUCKETS                ViewId = "S3 Buckets"
	STATE_MACHINES           ViewId = "State Machines"
	SYSTEMS_MANAGER          ViewId = "Systems Manager"

	HELP       ViewId = "Help"
	SETTINGS   ViewId = "Settings"
	DEBUG_LOGS ViewId = "Debug Logs"
)

type ServiceListItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	SelectedFunc  func()
}

type ServicesHomeView struct {
	*core.SearchableView
	filteredList     *tview.List
	serviceListItems []ServiceListItem
}

func NewServicesHomeView(app *tview.Application, logger *log.Logger) *ServicesHomeView {
	var servicesList = tview.NewList().
		SetSecondaryTextColor(tcell.ColorGrey).
		SetSelectedTextColor(core.TertiaryTextColor).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorGrey)

	servicesList.SetBorderPadding(0, 0, 1, 1)

	var view = &ServicesHomeView{
		SearchableView:   core.NewSearchableView(servicesList, app),
		filteredList:     servicesList,
		serviceListItems: []ServiceListItem{},
	}

	servicesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = servicesList.GetCurrentItem()
		var numItems = servicesList.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case core.APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				servicesList.SetCurrentItem(currentIdx)
				return nil
			case core.APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				servicesList.SetCurrentItem(currentIdx)
				return nil
			}
		}
		return event
	})

	view.SetSearchChangedFunc(func(search string) {
		servicesList.Clear()

		if len(search) == 0 {
			for _, item := range view.serviceListItems {
				servicesList.AddItem(
					item.MainText, item.SecondaryText, item.Shortcut, item.SelectedFunc,
				)
			}
			return
		}

		var filteredItems = core.FuzzySearch(search, view.serviceListItems,
			func(listItem ServiceListItem) string {
				return listItem.MainText + " " + listItem.SecondaryText
			},
		)

		for _, item := range filteredItems {
			servicesList.AddItem(
				item.MainText, item.SecondaryText, item.Shortcut, item.SelectedFunc,
			)
		}
		return
	})

	return view
}

func (inst *ServicesHomeView) AddItem(
	mainText string, secondaryText string, shortcut rune, selected func(),
) {
	inst.filteredList.AddItem(mainText, secondaryText, shortcut, selected)
	inst.serviceListItems = append(inst.serviceListItems, ServiceListItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		SelectedFunc:  selected,
	})
}

func (inst *ServicesHomeView) IsEscapable() bool {
	return !inst.SearchableView.IsEscapable()
}
