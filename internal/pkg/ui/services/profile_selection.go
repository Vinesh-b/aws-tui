package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"context"
	"slices"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ListItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	SelectedFunc  func()
}

type AwsProfilesView struct {
	*core.SearchableView
	filteredList     *tview.List
	serviceListItems []ServiceListItem
}

func NewProfileSelectionView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var listView = tview.NewList().
		SetSecondaryTextColor(tcell.ColorDarkGray).
		SetSelectedTextColor(appCtx.Theme.SecondaryTextColour).
		SetHighlightFullLine(true)

	listView.
		SetBorder(true).
		SetBorderPadding(1, 0, 1, 1).
		SetTitle("Available Profiles")

	var view = &ServicesHomeView{
		SearchableView:   core.NewSearchableView(listView, appCtx),
		filteredList:     listView,
		serviceListItems: []ServiceListItem{},
	}

	listView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = listView.GetCurrentItem()
		var numItems = listView.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case core.APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				listView.SetCurrentItem(currentIdx)
				return nil
			case core.APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				listView.SetCurrentItem(currentIdx)
				return nil
			}
		}
		return event
	})

	view.SetSearchChangedFunc(func(search string) {
		listView.Clear()

		if len(search) == 0 {
			for _, item := range view.serviceListItems {
				listView.AddItem(
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
			listView.AddItem(
				item.MainText, item.SecondaryText, item.Shortcut, item.SelectedFunc,
			)
		}
	})

	var manager = awsapi.NewAWSClientManager()

	var profiles, err = manager.ListAvailableProfiles()
	if err != nil {
		appCtx.Logger.Println(err)
	}

	slices.Sort(profiles)

	for _, profile := range profiles {
		view.AddItem(profile, "", '*', func() {
			var cfg, err = manager.SwitchToProfile(context.Background(), profile)
			if err != nil {
				appCtx.Logger.Println(err)
				return
			}
			appCtx.ResetApiClients(cfg, profile)
		})
	}

	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(view, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{view},
		},
	)

	var serviceRootView = core.NewServiceRootView(string(PROFILE_SELECTION), appCtx)
	serviceRootView.
		AddAndSwitchToPage("Profiles", serviceView, true)

	return serviceRootView
}

func (inst *AwsProfilesView) AddItem(
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
