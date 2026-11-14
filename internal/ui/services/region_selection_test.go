package services

import (
	"strings"
	"testing"

	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
)

func TestRegionSelection__ListsTheRegionsWithWhereTheyAre(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewRegionSelectionView(appCtx)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	// Nothing is drawn until something happens
	appCtx.App.QueueUpdateDraw(func() {})

	uitest.AwaitScreenText(t, appCtx.App, screen, "us-east-1")
	uitest.AwaitScreenText(t, appCtx.App, screen, "US East (N. Virginia)")
}

func TestRegionSelection__SelectingARegionSwitchesTheApiClientsKeepingTheProfile(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	appCtx.ResetApiClients(aws.Config{Region: "eu-west-1"}, "beta")
	var page = NewRegionSelectionView(appCtx)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	// The second region of the list
	uitest.PressRune(t, appCtx.App, 'j')
	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.AwaitScreenText(t, appCtx.App, screen, "Switched to region us-east-2")
	var clients = appCtx.GetApiClients()
	if clients.Config.Region != "us-east-2" || clients.Profile != "beta" {
		t.Errorf("clients = region %q, profile %q, want us-east-2 for beta", clients.Config.Region, clients.Profile)
	}
}

func TestAwsRegions__AreListedOnceAndNamed(t *testing.T) {
	var seen = map[string]bool{}
	for _, region := range awsRegions {
		if seen[region.code] {
			t.Errorf("region %s is listed twice", region.code)
		}
		seen[region.code] = true
		if strings.TrimSpace(region.name) == "" || strings.Count(region.code, "-") < 2 {
			t.Errorf("region %+v has no name or is not the code of a region", region)
		}
	}
}
