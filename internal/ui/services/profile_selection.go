package services

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"context"
	"slices"

	"github.com/aws/smithy-go/logging"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewProfileSelectionView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var listView = tview.NewList().
		SetHighlightFullLine(true)

	listView.
		SetBorder(true).
		SetBorderPadding(1, 0, 1, 1).
		SetTitle("Available Profiles")

	var view = core.NewFilterableList(listView, appCtx)
	var serviceView = core.NewServicePageView(appCtx)

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
				serviceView.DisplayMessage(core.ErrorPrompt, "%s", err.Error())
				return
			}
			cfg.Logger = logging.StandardLogger{Logger: appCtx.Logger}
			appCtx.ResetApiClients(cfg, profile)
			serviceView.DisplayMessage(
				core.InfoPrompt, "Switched to profile %s (region %s)", profile, cfg.Region,
			)
		})
	}

	serviceView.MainPage.AddItem(view, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{view},
		},
	)

	var serviceRootView = core.NewServiceRootView(string(core.PROFILE_SELECTION), appCtx)
	serviceRootView.
		AddAndSwitchToPage("Profiles", serviceView, true)

	return serviceRootView
}
