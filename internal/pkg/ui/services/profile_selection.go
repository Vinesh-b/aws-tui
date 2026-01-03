package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AwsProfilesView struct {
	*core.ServicePageView
	ListView *tview.List
}

func NewProfileSelectionView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var profilesListView = tview.NewList().
		SetSecondaryTextColor(tcell.ColorDarkGray).
		SetSelectedTextColor(appCtx.Theme.SecondaryTextColour).
		SetHighlightFullLine(true)

	profilesListView.
		SetBorder(true).
		SetBorderPadding(1, 0, 1, 1).
		SetTitle("Available Profiles")

	profilesListView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = profilesListView.GetCurrentItem()
		var numItems = profilesListView.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case core.APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				profilesListView.SetCurrentItem(currentIdx)
				return nil
			case core.APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				profilesListView.SetCurrentItem(currentIdx)
				return nil
			}
		}
		return event
	})

	var manager = awsapi.NewAWSClientManager()

	var profiles, err = manager.ListAvailableProfiles()
	if err != nil {
		appCtx.Logger.Println(err)
	}

	for _, profile := range profiles {
		profilesListView.AddItem(profile, "", '*', func() {
			var cfg, err = manager.SwitchToProfile(context.Background(), profile)
			if err != nil {
				appCtx.Logger.Println(err)
				return
			}
			appCtx.ResetApiClients(cfg, profile)
		})
	}

	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(profilesListView, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{profilesListView},
		},
	)

	var serviceRootView = core.NewServiceRootView(string(PROFILE_SELECTION), appCtx)
	serviceRootView.
		AddAndSwitchToPage("Profiles", serviceView, true)

	return serviceRootView
}
