package bedrock

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newTabsPage is the tabs over the list, as the pages of WAF have them
func newTabsPage(
	tabView *core.TabViewHorizontal, list core.View, appCtx *core.AppContext,
) *core.ServicePageView {
	const tabViewSize = 5000
	const listSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		list, listSize,
		tview.FlexRow,
		appCtx,
	)
	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{list},
		},
	)
	return serviceView
}

const (
	ModelTabNameDetails  = "Details"
	ModelTabNameProfiles = "Inference Profiles"
)

// FoundationModelsPageView is the foundation models, with the details and
// inference profiles of the one selected in tabs over them
type FoundationModelsPageView struct {
	*core.ServicePageView
	ModelsTable   *FoundationModelsTable
	DetailsTable  *ModelDetailsTable
	ProfilesTable *ModelProfilesTable
	TabView       *core.TabViewHorizontal
}

func NewFoundationModelsPageView(
	modelsTable *FoundationModelsTable,
	detailsTable *ModelDetailsTable,
	profilesTable *ModelProfilesTable,
	serviceCtx *core.ServiceContext[awsapi.BedrockApi],
) *FoundationModelsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(ModelTabNameDetails, detailsTable, 0, 1, true).
		AddTab(ModelTabNameProfiles, profilesTable, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		profilesTable.SetVisible(tabName == ModelTabNameProfiles)
	})

	var serviceView = newTabsPage(tabView, modelsTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	modelsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	profilesTable.ErrorMessageCallback = errorHandler

	// What the list has is shown at once, but whether the model can be used
	// is only loaded for the one the selection stops on
	modelsTable.SetSelectionChangedFunc(func(row, column int) {
		var model = modelsTable.GetSelectedModel()
		detailsTable.SetModel(model)
		profilesTable.SetModel(model)
	})
	modelsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshAvailability()
	})

	return &FoundationModelsPageView{
		ServicePageView: serviceView,
		ModelsTable:     modelsTable,
		DetailsTable:    detailsTable,
		ProfilesTable:   profilesTable,
		TabView:         tabView,
	}
}

const ProfileTabNameDetails = "Details"

// InferenceProfilesPageView is the inference profiles, with the details of
// the one selected over them
type InferenceProfilesPageView struct {
	*core.ServicePageView
	ProfilesTable *InferenceProfilesTable
	DetailsTable  *InferenceProfileDetailsTable
	TabView       *core.TabViewHorizontal
}

func NewInferenceProfilesPageView(
	profilesTable *InferenceProfilesTable,
	detailsTable *InferenceProfileDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.BedrockApi],
) *InferenceProfilesPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(ProfileTabNameDetails, detailsTable, 0, 1, true)

	var serviceView = newTabsPage(tabView, profilesTable, serviceCtx.AppContext)

	profilesTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	profilesTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetProfile(profilesTable.GetSelectedProfile())
	})

	return &InferenceProfilesPageView{
		ServicePageView: serviceView,
		ProfilesTable:   profilesTable,
		DetailsTable:    detailsTable,
		TabView:         tabView,
	}
}

// The pages of the service
const (
	bedrockPageModels = iota
	bedrockPageProfiles
)

func isOpenableBedrockResource(resource arn.ARN) bool {
	var named, err = awsapi.ParseBedrockArn(resource.String())
	if err != nil {
		return false
	}
	switch named.Kind {
	case awsapi.BedrockFoundationModel, awsapi.BedrockInferenceProfile, awsapi.BedrockApplicationInferenceProfile:
		return true
	}
	return false
}

func NewBedrockHomeView(appCtx *core.AppContext, api awsapi.BedrockApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x01A88D))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		modelsView = NewFoundationModelsPageView(
			NewFoundationModelsTable(serviceCtx),
			NewModelDetailsTable(serviceCtx),
			NewModelProfilesTable(serviceCtx),
			serviceCtx,
		)
		profilesView = NewInferenceProfilesPageView(
			NewInferenceProfilesTable(serviceCtx),
			NewInferenceProfileDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.BEDROCK), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Foundation Models", modelsView, true).
		AddPage("Inference Profiles", profilesView, true, true)

	serviceRootView.InitPageNavigation()

	// A model is found by its id in the region shown, which is the same model
	// as in the region of its ARN
	serviceRootView.SetResourceOpener("bedrock", isOpenableBedrockResource, func(resource arn.ARN) {
		var named, _ = awsapi.ParseBedrockArn(resource.String())
		switch named.Kind {
		case awsapi.BedrockFoundationModel:
			serviceRootView.ChangePage(bedrockPageModels, nil)
			modelsView.ModelsTable.Open(named.Id)
		case awsapi.BedrockInferenceProfile, awsapi.BedrockApplicationInferenceProfile:
			serviceRootView.ChangePage(bedrockPageProfiles, nil)
			profilesView.ProfilesTable.Open(named.Id)
		}
	})

	return serviceRootView
}
