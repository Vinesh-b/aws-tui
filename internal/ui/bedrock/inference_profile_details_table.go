package bedrock

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

// InferenceProfileDetailsTable shows the inference profile selected, which
// the list has all of, with the model it sends requests to in each region.
// The models open on the foundation models page.
type InferenceProfileDetailsTable struct {
	*core.DetailsTable
}

func NewInferenceProfileDetailsTable(
	serviceContext *core.ServiceContext[awsapi.BedrockApi],
) *InferenceProfileDetailsTable {
	var table = &InferenceProfileDetailsTable{
		DetailsTable: core.NewDetailsTable("Inference Profile Details", serviceContext.AppContext),
	}
	table.SetProfile(types.InferenceProfileSummary{})
	return table
}

// SetProfile shows the inference profile
func (inst *InferenceProfileDetailsTable) SetProfile(profile types.InferenceProfileSummary) {
	var tableData []core.TableRow
	var rawData any

	if profile.InferenceProfileId != nil {
		tableData = []core.TableRow{
			{"Name", aws.ToString(profile.InferenceProfileName)},
			{"Id", aws.ToString(profile.InferenceProfileId)},
			{"ARN", aws.ToString(profile.InferenceProfileArn)},
			{"Type", profileTypeName(profile.Type)},
			{"Status", string(profile.Status)},
			{"Description", aws.ToString(profile.Description)},
			{"Created", core.FormatTime(profile.CreatedAt)},
			{"Updated", core.FormatTime(profile.UpdatedAt)},
		}

		var models []core.TableRow
		for _, model := range profile.Models {
			var modelArn = aws.ToString(model.ModelArn)
			if region, modelId, ok := profileModel(modelArn); ok {
				models = append(models, core.DetailsLink(region, modelId, modelArn))
			} else {
				models = append(models, core.TableRow{"Model", modelArn})
			}
		}
		tableData = append(tableData, core.DetailsSection("Models", models...)...)
		rawData = profile
	}

	inst.SetTitleExtra(aws.ToString(profile.InferenceProfileName))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}
