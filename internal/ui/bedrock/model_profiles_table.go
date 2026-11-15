package bedrock

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	"github.com/gdamore/tcell/v2"
)

// ModelProfilesTable shows the inference profiles that send requests to the
// selected model, by name with the id to call, which open on the inference
// profiles page. The profiles are loaded once, when the tab is first opened,
// and only loaded again with reset, as they are the same for every model.
type ModelProfilesTable struct {
	*core.DetailsTable
	model    types.FoundationModelSummary
	profiles []types.InferenceProfileSummary
	// loaded is set when the profiles are asked for, so a load that times
	// out can be tried again with reset
	loaded     bool
	visible    bool
	serviceCtx *core.ServiceContext[awsapi.BedrockApi]
}

func NewModelProfilesTable(
	serviceContext *core.ServiceContext[awsapi.BedrockApi],
) *ModelProfilesTable {
	var table = &ModelProfilesTable{
		DetailsTable: core.NewDetailsTable("Inference Profiles", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateProfilesTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.loaded = false
			table.Load()
			return nil
		}
		return event
	})

	return table
}

func (inst *ModelProfilesTable) populateProfilesTable() {
	var modelId = aws.ToString(inst.model.ModelId)
	var system, application []core.TableRow
	var routing []types.InferenceProfileSummary

	if len(modelId) > 0 {
		for _, profile := range inst.profiles {
			if !routesModel(profile, modelId) {
				continue
			}
			routing = append(routing, profile)
			var row = core.DetailsLink(
				aws.ToString(profile.InferenceProfileName),
				aws.ToString(profile.InferenceProfileId),
				aws.ToString(profile.InferenceProfileArn),
			)
			if profile.Type == types.InferenceProfileTypeApplication {
				application = append(application, row)
			} else {
				system = append(system, row)
			}
		}
	}

	var tableData = append(
		core.DetailsSection("System-defined", system...),
		core.DetailsSection("Application", application...)...,
	)

	inst.SetTitleExtra(aws.ToString(inst.model.ModelName))
	inst.SetRawData(routing)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetModel shows the profiles of the model, from those loaded
func (inst *ModelProfilesTable) SetModel(model types.FoundationModelSummary) {
	inst.model = model
	inst.populateProfilesTable()
}

func (inst *ModelProfilesTable) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the inference profiles, if the tab is open and they aren't
// loaded yet
func (inst *ModelProfilesTable) Load() {
	if !inst.visible || inst.loaded {
		return
	}
	inst.loaded = true

	var profiles []types.InferenceProfileSummary
	var loadFailed = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		profiles, err = inst.serviceCtx.Api.ListInferenceProfiles(ctx)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// What was found before an error is shown, and opening the tab again
		// tries again
		if loadFailed {
			inst.loaded = false
		}
		inst.profiles = profiles
		inst.populateProfilesTable()
	})
}
