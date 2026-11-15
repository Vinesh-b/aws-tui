package awsapi

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

// The kinds of resource of Bedrock ARNs the pages show, which start their
// resource before a "/"
const (
	BedrockFoundationModel             = "foundation-model"
	BedrockInferenceProfile            = "inference-profile"
	BedrockApplicationInferenceProfile = "application-inference-profile"
)

// BedrockArn is what a Bedrock ARN names, whose resource is "<kind>/<id>",
// such as "foundation-model/anthropic.claude-3-haiku-20240307-v1:0"
type BedrockArn struct {
	Region string
	Kind   string
	Id     string
}

// ParseBedrockArn is what the Bedrock ARN names
func ParseBedrockArn(resourceArn string) (BedrockArn, error) {
	var parsed, err = arn.Parse(resourceArn)
	if err != nil {
		return BedrockArn{}, fmt.Errorf("invalid Bedrock ARN %q: %w", resourceArn, err)
	}

	var kind, id, found = strings.Cut(parsed.Resource, "/")
	if parsed.Service != "bedrock" || !found || len(kind) == 0 || len(id) == 0 || strings.Contains(id, "/") {
		return BedrockArn{}, fmt.Errorf("invalid Bedrock ARN %q", resourceArn)
	}
	return BedrockArn{Region: parsed.Region, Kind: kind, Id: id}, nil
}

// ModelAvailability is whether the account can use a foundation model
type ModelAvailability struct {
	ModelId       string
	Authorization types.AuthorizationStatus
	Entitlement   types.EntitlementAvailability
	Agreement     types.AgreementStatus
	// AgreementError is why the agreement is in error, if it is
	AgreementError string
	Region         types.RegionAvailability
}

// BedrockApi browses the foundation models and inference profiles of the
// region
type BedrockApi interface {
	// ListFoundationModels returns the foundation models, sorted by provider
	// then name
	ListFoundationModels(ctx context.Context) ([]types.FoundationModelSummary, error)
	// GetFoundationModelAvailability returns whether the account can use the
	// model of the id
	GetFoundationModelAvailability(ctx context.Context, modelId string) (*ModelAvailability, error)
	// ListInferenceProfiles returns the inference profiles Bedrock defines,
	// then those of the account, each sorted by name. Profiles found before
	// an error are returned with it.
	ListInferenceProfiles(ctx context.Context) ([]types.InferenceProfileSummary, error)
}

type bedrockApi struct {
	baseApi
}

func NewBedrockApi(
	clients *AwsApiClients,
	logger *log.Logger,
) BedrockApi {
	return &bedrockApi{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

func (inst *bedrockApi) ListFoundationModels(ctx context.Context) ([]types.FoundationModelSummary, error) {
	// The models all come in one call, which has no pages
	var output, err = inst.clients.bedrock.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return []types.FoundationModelSummary{}, inst.logErr(err)
	}

	var models = output.ModelSummaries
	slices.SortStableFunc(models, func(a, b types.FoundationModelSummary) int {
		return cmp.Or(
			cmp.Compare(aws.ToString(a.ProviderName), aws.ToString(b.ProviderName)),
			cmp.Compare(aws.ToString(a.ModelName), aws.ToString(b.ModelName)),
			cmp.Compare(aws.ToString(a.ModelId), aws.ToString(b.ModelId)),
		)
	})
	return models, nil
}

func (inst *bedrockApi) GetFoundationModelAvailability(ctx context.Context, modelId string) (*ModelAvailability, error) {
	if err := requireArg("model id", modelId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.bedrock.GetFoundationModelAvailability(ctx,
		&bedrock.GetFoundationModelAvailabilityInput{ModelId: aws.String(modelId)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	var availability = &ModelAvailability{
		ModelId:       aws.ToString(output.ModelId),
		Authorization: output.AuthorizationStatus,
		Entitlement:   output.EntitlementAvailability,
		Region:        output.RegionAvailability,
	}
	if agreement := output.AgreementAvailability; agreement != nil {
		availability.Agreement = agreement.Status
		availability.AgreementError = aws.ToString(agreement.ErrorMessage)
	}
	return availability, nil
}

func (inst *bedrockApi) ListInferenceProfiles(ctx context.Context) ([]types.InferenceProfileSummary, error) {
	// Each type is asked for, so the result doesn't depend on which the API
	// gives when none is
	var result = []types.InferenceProfileSummary{}
	for _, profileType := range []types.InferenceProfileType{
		types.InferenceProfileTypeSystemDefined,
		types.InferenceProfileTypeApplication,
	} {
		var paginator = bedrock.NewListInferenceProfilesPaginator(
			inst.clients.bedrock, &bedrock.ListInferenceProfilesInput{TypeEquals: profileType},
		)
		var profiles, err = collectPaginator(paginator.HasMorePages, func() ([]types.InferenceProfileSummary, error) {
			var output, err = paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			return output.InferenceProfileSummaries, nil
		})
		sortByKey(profiles, func(profile types.InferenceProfileSummary) *string { return profile.InferenceProfileName })
		result = append(result, profiles...)
		if err != nil {
			return result, inst.logErr(err)
		}
	}
	return result, nil
}
