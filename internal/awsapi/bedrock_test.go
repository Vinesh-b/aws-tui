package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"aws-tui/internal/safetest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

func bedrockResponse(body string) *http.Response {
	return response(200, "application/json", body)
}

func TestBedrockListFoundationModelsSortedByProviderThenName(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		urls.Record(requestUrl(req))
		return bedrockResponse(`{"modelSummaries":[` +
			`{"modelId":"meta.llama3-8b-instruct-v1:0","modelName":"Llama 3 8B Instruct","providerName":"Meta"},` +
			`{"modelId":"anthropic.claude-3-haiku-20240307-v1:0","modelName":"Claude 3 Haiku","providerName":"Anthropic"},` +
			`{"modelId":"amazon.titan-embed-text-v2:0","modelName":"Titan Text Embeddings V2","providerName":"Amazon"}]}`)
	})

	var models, err = NewBedrockApi(fake.clients, testLogger()).ListFoundationModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var providers []string
	for _, model := range models {
		providers = append(providers, aws.ToString(model.ProviderName))
	}
	if strings.Join(providers, " ") != "Amazon Anthropic Meta" {
		t.Errorf("models are of %v, want them sorted by provider", providers)
	}
	if got := urls.Calls(); len(got) != 1 || got[0] != "/foundation-models" {
		t.Errorf("requests = %v, want the models with no filter", got)
	}
}

func TestBedrockGetFoundationModelAvailability(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		urls.Record(requestUrl(req))
		return bedrockResponse(`{"modelId":"anthropic.claude-3-haiku-20240307-v1:0",` +
			`"authorizationStatus":"AUTHORIZED","entitlementAvailability":"AVAILABLE",` +
			`"regionAvailability":"AVAILABLE",` +
			`"agreementAvailability":{"status":"ERROR","errorMessage":"Payment instrument missing"}}`)
	})

	var availability, err = NewBedrockApi(fake.clients, testLogger()).
		GetFoundationModelAvailability(context.Background(), "anthropic.claude-3-haiku-20240307-v1:0")
	if err != nil {
		t.Fatal(err)
	}
	var want = ModelAvailability{
		ModelId:        "anthropic.claude-3-haiku-20240307-v1:0",
		Authorization:  types.AuthorizationStatusAuthorized,
		Entitlement:    types.EntitlementAvailabilityAvailable,
		Agreement:      types.AgreementStatusError,
		AgreementError: "Payment instrument missing",
		Region:         types.RegionAvailabilityAvailable,
	}
	if *availability != want {
		t.Errorf("availability = %+v, want %+v", *availability, want)
	}
	// The ":" of the model id is escaped in the path
	if got := urls.Calls(); len(got) != 1 ||
		got[0] != "/foundation-model-availability/anthropic.claude-3-haiku-20240307-v1%3A0" {
		t.Errorf("requests = %v, want the availability of the model", got)
	}
}

func TestBedrockListInferenceProfilesAsksForEachTypeAndFollowsTheToken(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		var url = requestUrl(req)
		urls.Record(url)
		switch {
		case strings.Contains(url, "type=APPLICATION"):
			return bedrockResponse(`{"inferenceProfileSummaries":[{"inferenceProfileName":"chat-costs","type":"APPLICATION"}]}`)
		case strings.Contains(url, "nextToken=t1"):
			return bedrockResponse(`{"inferenceProfileSummaries":[{"inferenceProfileName":"EU Claude 3 Haiku","type":"SYSTEM_DEFINED"}]}`)
		}
		return bedrockResponse(`{"inferenceProfileSummaries":[{"inferenceProfileName":"US Claude 3 Haiku","type":"SYSTEM_DEFINED"}],"nextToken":"t1"}`)
	})

	var profiles, err = NewBedrockApi(fake.clients, testLogger()).ListInferenceProfiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, profile := range profiles {
		names = append(names, aws.ToString(profile.InferenceProfileName))
	}
	if strings.Join(names, ", ") != "EU Claude 3 Haiku, US Claude 3 Haiku, chat-costs" {
		t.Errorf("profiles = %v, want those of Bedrock sorted, then those of the account", names)
	}
	if got := urls.Calls(); len(got) != 3 || !strings.Contains(got[0], "type=SYSTEM_DEFINED") {
		t.Errorf("requests = %v, want two pages of the system defined profiles then the application ones", got)
	}
}

func TestBedrockListInferenceProfilesGivesThoseFoundBeforeAnError(t *testing.T) {
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		if strings.Contains(requestUrl(req), "type=APPLICATION") {
			return response(403, "application/json", `{"message":"not allowed to list application profiles"}`)
		}
		return bedrockResponse(`{"inferenceProfileSummaries":[{"inferenceProfileName":"US Claude 3 Haiku"}]}`)
	})

	var profiles, err = NewBedrockApi(fake.clients, testLogger()).ListInferenceProfiles(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("err = %v, want the error of the application profiles", err)
	}
	if len(profiles) != 1 || aws.ToString(profiles[0].InferenceProfileName) != "US Claude 3 Haiku" {
		t.Errorf("profiles = %v, want the system defined one found before the error", profiles)
	}
}

func TestParseBedrockArn(t *testing.T) {
	for _, test := range []struct {
		arn     string
		want    BedrockArn
		wantErr bool
	}{
		{"arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-haiku-20240307-v1:0",
			BedrockArn{"us-east-1", BedrockFoundationModel, "anthropic.claude-3-haiku-20240307-v1:0"}, false},
		{"arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-3-haiku-20240307-v1:0",
			BedrockArn{"us-east-1", BedrockInferenceProfile, "us.anthropic.claude-3-haiku-20240307-v1:0"}, false},
		{"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/a1b2c3",
			BedrockArn{"us-east-1", BedrockApplicationInferenceProfile, "a1b2c3"}, false},
		{"arn:aws:bedrock:us-east-1::foundation-model/", BedrockArn{}, true},
		{"arn:aws:bedrock:us-east-1::foundation-model", BedrockArn{}, true},
		{"arn:aws:sagemaker:us-east-1:123456789012:model/a", BedrockArn{}, true},
		{"not an arn", BedrockArn{}, true},
	} {
		var got, err = ParseBedrockArn(test.arn)
		if (err != nil) != test.wantErr || got != test.want {
			t.Errorf("ParseBedrockArn(%q) = %+v, %v, want %+v, error %t", test.arn, got, err, test.want, test.wantErr)
		}
	}
}
