package bedrock

import (
	"strconv"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

func formatBool(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}

func lifecycleStatus(model types.FoundationModelSummary) string {
	if model.ModelLifecycle == nil {
		return ""
	}
	return string(model.ModelLifecycle.Status)
}

func profileTypeName(profileType types.InferenceProfileType) string {
	switch profileType {
	case types.InferenceProfileTypeSystemDefined:
		return "System-defined"
	case types.InferenceProfileTypeApplication:
		return "Application"
	}
	return string(profileType)
}

// profileModel is the model a model ARN of an inference profile names, with
// the region it is in, or the ARN when it isn't one of a foundation model
func profileModel(modelArn string) (region string, modelId string, ok bool) {
	var named, err = awsapi.ParseBedrockArn(modelArn)
	if err != nil || named.Kind != awsapi.BedrockFoundationModel {
		return "", modelArn, false
	}
	return named.Region, named.Id, true
}

// routesModel is whether the inference profile sends requests to the model of
// the id, in any of its regions
func routesModel(profile types.InferenceProfileSummary, modelId string) bool {
	for _, model := range profile.Models {
		if _, id, ok := profileModel(aws.ToString(model.ModelArn)); ok && id == modelId {
			return true
		}
	}
	return false
}

// agreementStatus is the status of the model's agreement, with why it is in
// error when it is
func agreementStatus(availability awsapi.ModelAvailability) string {
	if len(availability.AgreementError) > 0 {
		return string(availability.Agreement) + ": " + availability.AgreementError
	}
	return string(availability.Agreement)
}
