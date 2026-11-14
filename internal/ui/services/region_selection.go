package services

import (
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// awsRegion is a region and where it is
type awsRegion struct {
	code string
	name string
}

// awsRegions are the regions that can be switched to. The list is fixed, so
// nothing has to be asked of AWS, and a region that is not turned on for the
// account fails when it is used as it would when asked for by name.
var awsRegions = []awsRegion{
	{"us-east-1", "US East (N. Virginia)"},
	{"us-east-2", "US East (Ohio)"},
	{"us-west-1", "US West (N. California)"},
	{"us-west-2", "US West (Oregon)"},
	{"af-south-1", "Africa (Cape Town)"},
	{"ap-east-1", "Asia Pacific (Hong Kong)"},
	{"ap-south-1", "Asia Pacific (Mumbai)"},
	{"ap-south-2", "Asia Pacific (Hyderabad)"},
	{"ap-southeast-1", "Asia Pacific (Singapore)"},
	{"ap-southeast-2", "Asia Pacific (Sydney)"},
	{"ap-southeast-3", "Asia Pacific (Jakarta)"},
	{"ap-southeast-4", "Asia Pacific (Melbourne)"},
	{"ap-southeast-5", "Asia Pacific (Malaysia)"},
	{"ap-northeast-1", "Asia Pacific (Tokyo)"},
	{"ap-northeast-2", "Asia Pacific (Seoul)"},
	{"ap-northeast-3", "Asia Pacific (Osaka)"},
	{"ca-central-1", "Canada (Central)"},
	{"ca-west-1", "Canada West (Calgary)"},
	{"eu-central-1", "Europe (Frankfurt)"},
	{"eu-central-2", "Europe (Zurich)"},
	{"eu-west-1", "Europe (Ireland)"},
	{"eu-west-2", "Europe (London)"},
	{"eu-west-3", "Europe (Paris)"},
	{"eu-north-1", "Europe (Stockholm)"},
	{"eu-south-1", "Europe (Milan)"},
	{"eu-south-2", "Europe (Spain)"},
	{"il-central-1", "Israel (Tel Aviv)"},
	{"me-south-1", "Middle East (Bahrain)"},
	{"me-central-1", "Middle East (UAE)"},
	{"sa-east-1", "South America (São Paulo)"},
}

// NewRegionSelectionView lists the regions, and switches the api clients to
// the one that is picked, for the profile in use
func NewRegionSelectionView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var listView = tview.NewList().
		SetHighlightFullLine(true)

	listView.
		SetBorder(true).
		SetBorderPadding(1, 0, 1, 1).
		SetTitle("Available Regions")

	var view = core.NewFilterableList(listView, appCtx)
	var serviceView = core.NewServicePageView(appCtx)

	for _, region := range awsRegions {
		view.AddItem(region.code, region.name, '*', func() {
			appCtx.SwitchRegion(region.code)
			serviceView.DisplayMessage(core.InfoPrompt, "Switched to region %s", region.code)
		})
	}

	serviceView.MainPage.AddItem(view, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{view},
		},
	)

	var serviceRootView = core.NewServiceRootView(string(core.REGION_SELECTION), appCtx)
	serviceRootView.
		AddAndSwitchToPage("Regions", serviceView, true)

	return serviceRootView
}
