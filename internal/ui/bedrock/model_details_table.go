package bedrock

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	"github.com/gdamore/tcell/v2"
)

// ModelDetailsTable shows a foundation model, which the list has all of, then
// whether the account can use it once that is loaded
type ModelDetailsTable struct {
	*core.DetailsTable
	model        types.FoundationModelSummary
	availability *awsapi.ModelAvailability
	serviceCtx   *core.ServiceContext[awsapi.BedrockApi]
}

func NewModelDetailsTable(
	serviceContext *core.ServiceContext[awsapi.BedrockApi],
) *ModelDetailsTable {
	var table = &ModelDetailsTable{
		DetailsTable: core.NewDetailsTable("Model Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshAvailability()
			return nil
		}
		return event
	})

	return table
}

func (inst *ModelDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.model.ModelId != nil {
		var model = inst.model
		var lifecycle = model.ModelLifecycle
		if lifecycle == nil {
			lifecycle = &types.FoundationModelLifecycle{}
		}

		tableData = []core.TableRow{
			{"Name", aws.ToString(model.ModelName)},
			{"Model Id", aws.ToString(model.ModelId)},
			{"ARN", aws.ToString(model.ModelArn)},
			{"Provider", aws.ToString(model.ProviderName)},
			{"Input", core.FormatList(model.InputModalities)},
			{"Output", core.FormatList(model.OutputModalities)},
			{"Streaming", formatBool(model.ResponseStreamingSupported)},
			{"Inference Types", core.FormatList(model.InferenceTypesSupported)},
			{"Customizations", core.FormatList(model.CustomizationsSupported)},
		}
		tableData = append(tableData, core.DetailsSection("Lifecycle",
			core.TableRow{"Status", string(lifecycle.Status)},
			core.TableRow{"Released", core.FormatTime(lifecycle.StartOfLifeTime)},
			core.TableRow{"Legacy From", core.FormatTime(lifecycle.LegacyTime)},
			core.TableRow{"Extended Access From", core.FormatTime(lifecycle.PublicExtendedAccessTime)},
			core.TableRow{"End of Life", core.FormatTime(lifecycle.EndOfLifeTime)},
		)...)
		rawData = model

		if availability := inst.availability; availability != nil {
			tableData = append(tableData, core.DetailsSection("Access",
				core.TableRow{"Authorization", string(availability.Authorization)},
				core.TableRow{"Entitlement", string(availability.Entitlement)},
				core.TableRow{"Agreement", agreementStatus(*availability)},
				core.TableRow{"Region", string(availability.Region)},
			)...)
			rawData = struct {
				Model        types.FoundationModelSummary
				Availability awsapi.ModelAvailability
			}{model, *availability}
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.model.ModelName))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetModel shows the model, until whether it can be used is loaded
func (inst *ModelDetailsTable) SetModel(model types.FoundationModelSummary) {
	inst.model = model
	inst.availability = nil
	inst.populateDetailsTable()
}

// RefreshAvailability loads whether the account can use the model shown
func (inst *ModelDetailsTable) RefreshAvailability() {
	var modelId = aws.ToString(inst.model.ModelId)
	if len(modelId) == 0 {
		return
	}

	var availability *awsapi.ModelAvailability
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		availability, err = inst.serviceCtx.Api.GetFoundationModelAvailability(ctx, modelId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another model was selected while this one loaded
		if aws.ToString(inst.model.ModelId) != modelId || availability == nil {
			return
		}
		inst.availability = availability
		inst.populateDetailsTable()
	})
}
