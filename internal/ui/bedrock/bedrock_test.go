package bedrock

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/bedrock/bedrockfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var modelHeadings = core.TableRow{"Provider", "Name", "Model Id", "Status", "Input", "Output"}
var profileHeadings = core.TableRow{"Name", "Id", "Type", "Status", "Models"}

const haikuId = "anthropic.claude-3-haiku-20240307-v1:0"
const titanId = "amazon.titan-embed-text-v2:0"

func modelArnOf(region string, modelId string) string {
	return "arn:aws:bedrock:" + region + "::foundation-model/" + modelId
}

func newModel(provider, name, modelId string, input, output types.ModelModality) types.FoundationModelSummary {
	return types.FoundationModelSummary{
		ProviderName:               aws.String(provider),
		ModelName:                  aws.String(name),
		ModelId:                    aws.String(modelId),
		ModelArn:                   aws.String(modelArnOf("us-east-1", modelId)),
		InputModalities:            []types.ModelModality{input},
		OutputModalities:           []types.ModelModality{output},
		ResponseStreamingSupported: aws.Bool(output == types.ModelModalityText),
		InferenceTypesSupported:    []types.InferenceType{types.InferenceTypeOnDemand},
		ModelLifecycle:             &types.FoundationModelLifecycle{Status: types.FoundationModelLifecycleStatusActive},
	}
}

func haiku() types.FoundationModelSummary {
	return newModel("Anthropic", "Claude 3 Haiku", haikuId, types.ModelModalityText, types.ModelModalityText)
}

func titan() types.FoundationModelSummary {
	return newModel("Amazon", "Titan Text Embeddings V2", titanId, types.ModelModalityText, types.ModelModalityEmbedding)
}

func systemProfile(name, profileId string, modelId string, regions ...string) types.InferenceProfileSummary {
	var profile = types.InferenceProfileSummary{
		InferenceProfileName: aws.String(name),
		InferenceProfileId:   aws.String(profileId),
		InferenceProfileArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:inference-profile/" + profileId),
		Type:                 types.InferenceProfileTypeSystemDefined,
		Status:               types.InferenceProfileStatusActive,
	}
	for _, region := range regions {
		profile.Models = append(profile.Models, types.InferenceProfileModel{ModelArn: aws.String(modelArnOf(region, modelId))})
	}
	return profile
}

func applicationProfile(name, profileId string, modelId string) types.InferenceProfileSummary {
	return types.InferenceProfileSummary{
		InferenceProfileName: aws.String(name),
		InferenceProfileId:   aws.String(profileId),
		InferenceProfileArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/" + profileId),
		Type:                 types.InferenceProfileTypeApplication,
		Status:               types.InferenceProfileStatusActive,
		Models:               []types.InferenceProfileModel{{ModelArn: aws.String(modelArnOf("us-east-1", modelId))}},
	}
}

// The profiles of the tests: two of Bedrock for Haiku, one for Titan and one
// of the account for Haiku
func testProfiles() []types.InferenceProfileSummary {
	return []types.InferenceProfileSummary{
		systemProfile("EU Claude 3 Haiku", "eu."+haikuId, haikuId, "eu-west-1", "eu-central-1"),
		systemProfile("US Claude 3 Haiku", "us."+haikuId, haikuId, "us-east-1", "us-west-2"),
		systemProfile("US Titan Embeddings", "us."+titanId, titanId, "us-east-1"),
		applicationProfile("chat-costs", "a1b2c3", haikuId),
	}
}

func TestRoutesModelIsInAnyRegionButOnlyOfTheModel(t *testing.T) {
	var profile = systemProfile("US Claude 3 Haiku", "us."+haikuId, haikuId, "us-west-2")
	if !routesModel(profile, haikuId) {
		t.Error("the profile should route Haiku, whose ARN is in another region")
	}
	if routesModel(profile, "anthropic.claude-3-haiku") {
		t.Error("the profile should not route a model whose id is only the start of Haiku's")
	}
	var odd = types.InferenceProfileSummary{Models: []types.InferenceProfileModel{{ModelArn: aws.String("not-an-arn")}}}
	if routesModel(odd, haikuId) {
		t.Error("a profile whose model isn't an ARN routes nothing")
	}
}

func newModelsTest(t *testing.T, api *bedrockfake.Api) *FoundationModelsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.BedrockApi](api)
	var view = NewFoundationModelsPageView(
		NewFoundationModelsTable(serviceCtx),
		NewModelDetailsTable(serviceCtx),
		NewModelProfilesTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

// loadModels loads the models of the page and waits for them
func loadModels(t *testing.T, view *FoundationModelsPageView, count int) {
	t.Helper()
	var app = view.ModelsTable.serviceCtx.App
	view.ModelsTable.RefreshFoundationModels()
	uitest.Eventually(t, app, func() bool {
		return view.ModelsTable.GetTable().GetRowCount() == count+1
	}, "the models to be loaded")
}

// openProfilesTab moves from the details tab to the inference profiles
func openProfilesTab(t *testing.T, view *FoundationModelsPageView) {
	t.Helper()
	var app = view.ModelsTable.serviceCtx.App
	uitest.OnUI(t, app, func() { app.SetFocus(view.DetailsTable) })
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.OnUI(t, app, func() {
		if name, _ := view.TabView.GetCurrentTab(); name != ModelTabNameProfiles {
			t.Fatalf("tab = %q, want %q", name, ModelTabNameProfiles)
		}
	})
}

// awaitProfileRows waits for the profiles of the model to be shown, as
// AwaitDetailsRows is satisfied by the rows of another model when the first
// labels match, then checks them and their links
func awaitProfileRows(t *testing.T, app *tview.Application, table *core.DetailsTable, model string, want []core.TableRow) {
	t.Helper()
	uitest.Eventually(t, app, func() bool {
		return strings.Contains(table.GetTitle(), model) && table.GetCellText(1, 1) == want[1][1]
	}, "the profiles of "+model+" to be shown")
	uitest.OnUI(t, app, func() {
		var text []core.TableRow
		for _, row := range want {
			text = append(text, row[:min(len(row), 2)])
		}
		uitest.AssertDetailsRows(t, table, text)
		for row, wantRow := range want {
			var link, _ = table.GetLink(row)
			if len(wantRow) == 3 && link != wantRow[2] {
				t.Errorf("link of row %d = %q, want %q", row, link, wantRow[2])
			}
		}
	})
}

func TestFoundationModelsTable__ResetLoadsTheModels(t *testing.T) {
	var api = bedrockfake.New(bedrockfake.WithModels([]types.FoundationModelSummary{titan(), haiku()}))
	var view = newModelsTest(t, api)
	var app = view.ModelsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.ModelsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.ModelsTable, []core.TableRow{
		modelHeadings,
		{"Amazon", "Titan Text Embeddings V2", titanId, "ACTIVE", "TEXT", "EMBEDDING"},
		{"Anthropic", "Claude 3 Haiku", haikuId, "ACTIVE", "TEXT", "TEXT"},
	})
}

func TestFoundationModelsTable__ErrorIsShown(t *testing.T) {
	var api = bedrockfake.New(bedrockfake.WithErr(errors.New("AccessDeniedException: bedrock:ListFoundationModels")))
	var view = newModelsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.ModelsTable.serviceCtx.App, func() { view.ModelsTable.ErrorMessageCallback = callback })

	view.ModelsTable.RefreshFoundationModels()

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "bedrock:ListFoundationModels") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestFoundationModelsPage__SelectingAModelShowsItThenWhetherItCanBeUsed(t *testing.T) {
	var api = bedrockfake.New(
		bedrockfake.WithModels([]types.FoundationModelSummary{haiku()}),
		bedrockfake.WithAvailability(haikuId, awsapi.ModelAvailability{
			ModelId:        haikuId,
			Authorization:  types.AuthorizationStatusAuthorized,
			Entitlement:    types.EntitlementAvailabilityAvailable,
			Agreement:      types.AgreementStatusError,
			AgreementError: "Payment instrument missing",
			Region:         types.RegionAvailabilityAvailable,
		}),
	)
	var view = newModelsTest(t, api)
	var app = view.ModelsTable.serviceCtx.App

	view.ModelsTable.RefreshFoundationModels()

	var details = view.DetailsTable.DetailsTable
	uitest.Eventually(t, app, func() bool { return uitest.DetailsRowOf(details, "Access") >= 0 },
		"whether the model can be used to be loaded")
	uitest.AwaitDetailsRows(t, app, details, []core.TableRow{
		{"Name", "Claude 3 Haiku"},
		{"Model Id", haikuId},
		{"ARN", modelArnOf("us-east-1", haikuId)},
		{"Provider", "Anthropic"},
		{"Input", "TEXT"},
		{"Output", "TEXT"},
		{"Streaming", "true"},
		{"Inference Types", "ON_DEMAND"},
		{"Customizations", ""},
		{"Lifecycle"},
		{"Status", "ACTIVE"},
		{"Released", ""},
		{"Legacy From", ""},
		{"Extended Access From", ""},
		{"End of Life", ""},
		{"Access"},
		{"Authorization", "AUTHORIZED"},
		{"Entitlement", "AVAILABLE"},
		{"Agreement", "ERROR: Payment instrument missing"},
		{"Region", "AVAILABLE"},
	})
}

func TestFoundationModelsPage__MovingThroughModelsAsksOnlyForTheOneItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var ids = []string{"a.one-v1", "b.two-v1", "c.three-v1", "d.four-v1"}
		var models []types.FoundationModelSummary
		for _, id := range ids {
			models = append(models, newModel("Test", id, id, types.ModelModalityText, types.ModelModalityText))
		}
		var api = bedrockfake.New(bedrockfake.WithModels(models))
		var view = newModelsTest(t, api)
		loadModels(t, view, len(ids))

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, view.ModelsTable.serviceCtx.App, uitest.MoveThrough[string]{
			Table: view.ModelsTable, Passed: "c.three-v1", StopsOn: "d.four-v1", Loaded: api.AvailabilityAskedFor,
			// The model is shown straight away, before whether it can be used is
			StraightAway: func() {
				if got := view.DetailsTable.GetCellText(1, 1); got != "d.four-v1" {
					t.Errorf("model shown = %q, want d.four-v1", got)
				}
			},
		})
	})
}

func TestFoundationModelsPage__ProfilesOfTheModelAreShownOnceTheirTabIsOpened(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = bedrockfake.New(
			bedrockfake.WithModels([]types.FoundationModelSummary{haiku()}),
			bedrockfake.WithProfiles(testProfiles()),
		)
		var view = newModelsTest(t, api)
		var app = view.ModelsTable.serviceCtx.App
		loadModels(t, view, 1)
		uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool { return api.ProfileLists() == 0 },
			"the profiles not to be loaded before the tab is opened")

		openProfilesTab(t, view)

		awaitProfileRows(t, app, view.ProfilesTable.DetailsTable, "Claude 3 Haiku", []core.TableRow{
			{"System-defined"},
			{"EU Claude 3 Haiku", "eu." + haikuId, "arn:aws:bedrock:us-east-1:123456789012:inference-profile/eu." + haikuId},
			{"US Claude 3 Haiku", "us." + haikuId, "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us." + haikuId},
			{"Application"},
			{"chat-costs", "a1b2c3", "arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/a1b2c3"},
		})
	})
}

func TestFoundationModelsPage__ProfilesAreLoadedOnceForEveryModelAndAgainWithReset(t *testing.T) {
	var api = bedrockfake.New(
		bedrockfake.WithModels([]types.FoundationModelSummary{titan(), haiku()}),
		bedrockfake.WithProfiles(testProfiles()),
	)
	var view = newModelsTest(t, api)
	var app = view.ModelsTable.serviceCtx.App
	loadModels(t, view, 2)
	openProfilesTab(t, view)
	var titanProfile = "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us." + titanId
	awaitProfileRows(t, app, view.ProfilesTable.DetailsTable, "Titan Text Embeddings V2", []core.TableRow{
		{"System-defined"},
		{"US Titan Embeddings", "us." + titanId, titanProfile},
	})

	uitest.OnUI(t, app, func() { view.ModelsTable.Select(2, 0) })

	awaitProfileRows(t, app, view.ProfilesTable.DetailsTable, "Claude 3 Haiku", []core.TableRow{
		{"System-defined"},
		{"EU Claude 3 Haiku", "eu." + haikuId},
		{"US Claude 3 Haiku", "us." + haikuId},
		{"Application"},
		{"chat-costs", "a1b2c3"},
	})
	if lists := api.ProfileLists(); lists != 1 {
		t.Errorf("profiles listed %d times, want once for both models", lists)
	}

	// A new profile is only seen once reset loads them again
	api.SetProfiles(append(testProfiles(), applicationProfile("support-bot", "d4e5f6", haikuId)))
	uitest.OnUI(t, app, func() { app.SetFocus(view.ProfilesTable) })
	uitest.PressRune(t, app, core.APP_KEY_BINDINGS.Reset)

	uitest.Eventually(t, app, func() bool { return view.ProfilesTable.GetCellText(5, 0) == "support-bot" },
		"the new profile to be shown")
}

func TestFoundationModelsPage__ProfilesErrorIsShownAndLoadedAgainWhenTheTabIsReopened(t *testing.T) {
	var api = bedrockfake.New(
		bedrockfake.WithModels([]types.FoundationModelSummary{titan()}),
		bedrockfake.WithProfiles(testProfiles()),
	)
	var view = newModelsTest(t, api)
	var app = view.ModelsTable.serviceCtx.App
	loadModels(t, view, 1)
	uitest.Eventually(t, app, func() bool { return len(api.AvailabilityAskedFor()) > 0 }, "the model to be asked for")
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.ProfilesTable.ErrorMessageCallback = callback })
	api.SetErr(errors.New("ThrottlingException: bedrock:ListInferenceProfiles"))

	openProfilesTab(t, view)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "bedrock:ListInferenceProfiles") {
		t.Errorf("error = %q, want the error of the api", got)
	}

	// The error is reported before the load ends, which is when it can be
	// tried again
	uitest.Eventually(t, app, func() bool { return !view.ProfilesTable.loaded }, "the failed load to end")
	api.SetErr(nil)
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.Eventually(t, app, func() bool { return view.ProfilesTable.GetCellText(1, 1) == "us."+titanId },
		"the profiles to be loaded again")
}

func TestFoundationModelsTable__OpenLoadsTheModelsToSelectOneNotLoaded(t *testing.T) {
	var api = bedrockfake.New(bedrockfake.WithModels([]types.FoundationModelSummary{titan(), haiku()}))
	var view = newModelsTest(t, api)
	var app = view.ModelsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { view.ModelsTable.Open(haikuId) })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(view.ModelsTable.GetSelectedModel().ModelId) == haikuId
	}, "Haiku to be selected")
}

func TestFoundationModelsTable__OpenOfAMissingModelIsAnError(t *testing.T) {
	var api = bedrockfake.New(bedrockfake.WithModels([]types.FoundationModelSummary{titan()}))
	var view = newModelsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.ModelsTable.serviceCtx.App, func() {
		view.ModelsTable.ErrorMessageCallback = callback
		view.ModelsTable.Open("gone.model-v1")
	})

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "gone.model-v1") {
		t.Errorf("error = %q, want it to name the model", got)
	}
}

func newProfilesTest(t *testing.T, api *bedrockfake.Api) *InferenceProfilesPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.BedrockApi](api)
	var view = NewInferenceProfilesPageView(
		NewInferenceProfilesTable(serviceCtx),
		NewInferenceProfileDetailsTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

func TestInferenceProfilesTable__ResetLoadsTheProfiles(t *testing.T) {
	var api = bedrockfake.New(bedrockfake.WithProfiles(testProfiles()))
	var view = newProfilesTest(t, api)
	var app = view.ProfilesTable.serviceCtx.App

	uitest.OnUI(t, app, func() { app.SetFocus(view.ProfilesTable) })
	uitest.PressRune(t, app, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitRows(t, app, view.ProfilesTable, []core.TableRow{
		profileHeadings,
		{"EU Claude 3 Haiku", "eu." + haikuId, "System-defined", "ACTIVE", "2"},
		{"US Claude 3 Haiku", "us." + haikuId, "System-defined", "ACTIVE", "2"},
		{"US Titan Embeddings", "us." + titanId, "System-defined", "ACTIVE", "1"},
		{"chat-costs", "a1b2c3", "Application", "ACTIVE", "1"},
	})
}

func TestInferenceProfilesPage__SelectingAProfileShowsItsModelsByRegion(t *testing.T) {
	var profile = systemProfile("EU Claude 3 Haiku", "eu."+haikuId, haikuId, "eu-west-1", "eu-central-1")
	profile.Models = append(profile.Models, types.InferenceProfileModel{ModelArn: aws.String("arn:aws:bedrock:eu-west-3::custom-model/x")})
	var api = bedrockfake.New(bedrockfake.WithProfiles([]types.InferenceProfileSummary{profile}))
	var view = newProfilesTest(t, api)
	var app = view.ProfilesTable.serviceCtx.App

	view.ProfilesTable.RefreshInferenceProfiles()

	var details = view.DetailsTable.DetailsTable
	uitest.AwaitDetailsRows(t, app, details, []core.TableRow{
		{"Name", "EU Claude 3 Haiku"},
		{"Id", "eu." + haikuId},
		{"ARN", "arn:aws:bedrock:us-east-1:123456789012:inference-profile/eu." + haikuId},
		{"Type", "System-defined"},
		{"Status", "ACTIVE"},
		{"Description", ""},
		{"Created", ""},
		{"Updated", ""},
		{"Models"},
		{"eu-west-1", haikuId},
		{"eu-central-1", haikuId},
		// What isn't a foundation model is shown as it is
		{"Model", "arn:aws:bedrock:eu-west-3::custom-model/x"},
	})
	uitest.OnUI(t, app, func() {
		for row, want := range map[int]string{
			9:  modelArnOf("eu-west-1", haikuId),
			10: modelArnOf("eu-central-1", haikuId),
		} {
			if link, _ := details.GetLink(row); link != want {
				t.Errorf("link of row %d = %q, want %q", row, link, want)
			}
		}
	})
}

func TestIsOpenableBedrockResource(t *testing.T) {
	for _, test := range []struct {
		resource string
		want     bool
	}{
		{modelArnOf("us-west-2", haikuId), true},
		{"arn:aws:bedrock:us-east-1:123456789012:inference-profile/us." + haikuId, true},
		{"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/a1b2c3", true},
		{"arn:aws:bedrock:us-east-1:123456789012:guardrail/abc", false},
		{"arn:aws:bedrock:us-east-1::foundation-model/", false},
	} {
		var resource, err = arn.Parse(test.resource)
		if err != nil {
			t.Fatal(err)
		}
		if got := isOpenableBedrockResource(resource); got != test.want {
			t.Errorf("isOpenableBedrockResource(%q) = %t, want %t", test.resource, got, test.want)
		}
	}
}

func TestHomeView__ResetLoadsTheModels(t *testing.T) {
	var api = bedrockfake.New(bedrockfake.WithModels([]types.FoundationModelSummary{haiku()}))
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewBedrockHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "Claude 3 Haiku")
}

func TestHomeView__OpeningAProfileShowsItOnTheInferenceProfilesPage(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewBedrockHomeView(appCtx, bedrockfake.New(bedrockfake.WithProfiles(testProfiles())))
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener, ok = page.(core.ResourceOpener)
	var resource, _ = arn.Parse("arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/a1b2c3")
	if !ok || !opener.OpensArn(resource) {
		t.Fatal("Bedrock page should open application inference profiles")
	}

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	uitest.AwaitScreenText(t, appCtx.App, screen, "Inference Profile Details ❬chat-costs❭")
}

func TestHomeView__AModelOfAProfileInAnotherRegionOpensTheModelOfTheRegionShown(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewBedrockHomeView(appCtx, bedrockfake.New(bedrockfake.WithModels([]types.FoundationModelSummary{titan(), haiku()})))
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener = page.(core.ResourceOpener)
	var resource, _ = arn.Parse(modelArnOf("us-west-2", haikuId))

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	uitest.AwaitScreenText(t, appCtx.App, screen, "Model Details ❬Claude 3 Haiku❭")
}
