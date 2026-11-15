// Package bedrockfake is an in-memory awsapi.BedrockApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package bedrockfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

type Api struct {
	awsapi.BedrockApi

	mtx    sync.Mutex
	models []types.FoundationModelSummary
	// availability is what GetFoundationModelAvailability gives, by model id
	availability map[string]awsapi.ModelAvailability
	profiles     []types.InferenceProfileSummary
	err          error
	// availabilityAskedFor is the ids GetFoundationModelAvailability was
	// called with, in order
	availabilityAskedFor []string
	profileLists         int
}

type Option func(*Api)

func WithModels(models []types.FoundationModelSummary) Option {
	return func(f *Api) { f.models = cloneModels(models) }
}

// WithAvailability gives what GetFoundationModelAvailability returns for the
// model id
func WithAvailability(modelId string, availability awsapi.ModelAvailability) Option {
	return func(f *Api) { f.availability[modelId] = availability }
}

func WithProfiles(profiles []types.InferenceProfileSummary) Option {
	return func(f *Api) { f.profiles = cloneProfiles(profiles) }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{availability: map[string]awsapi.ModelAvailability{}}
	for _, option := range options {
		option(f)
	}
	return f
}

// cloneModels copies the models with their lists and lifecycle, so the test
// and the UI never share them
func cloneModels(models []types.FoundationModelSummary) []types.FoundationModelSummary {
	var result = slices.Clone(models)
	for i := range result {
		result[i].InputModalities = slices.Clone(result[i].InputModalities)
		result[i].OutputModalities = slices.Clone(result[i].OutputModalities)
		result[i].InferenceTypesSupported = slices.Clone(result[i].InferenceTypesSupported)
		result[i].CustomizationsSupported = slices.Clone(result[i].CustomizationsSupported)
		if lifecycle := result[i].ModelLifecycle; lifecycle != nil {
			var copied = *lifecycle
			result[i].ModelLifecycle = &copied
		}
	}
	return result
}

// cloneProfiles copies the profiles with their models, so the test and the UI
// never share them
func cloneProfiles(profiles []types.InferenceProfileSummary) []types.InferenceProfileSummary {
	var result = slices.Clone(profiles)
	for i := range result {
		result[i].Models = slices.Clone(result[i].Models)
	}
	return result
}

// SetErr makes every following call fail
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// SetProfiles changes what ListInferenceProfiles returns
func (f *Api) SetProfiles(profiles []types.InferenceProfileSummary) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.profiles = cloneProfiles(profiles)
}

func (f *Api) ListFoundationModels(ctx context.Context) ([]types.FoundationModelSummary, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return []types.FoundationModelSummary{}, f.err
	}
	return cloneModels(f.models), nil
}

// GetFoundationModelAvailability gives the availability of the model id, or
// one with only the id when it wasn't given
func (f *Api) GetFoundationModelAvailability(ctx context.Context, modelId string) (*awsapi.ModelAvailability, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.availabilityAskedFor = append(f.availabilityAskedFor, modelId)
	if f.err != nil {
		return nil, f.err
	}
	var availability, ok = f.availability[modelId]
	if !ok {
		availability = awsapi.ModelAvailability{ModelId: modelId}
	}
	return &availability, nil
}

// AvailabilityAskedFor is the ids GetFoundationModelAvailability was called
// with, in order
func (f *Api) AvailabilityAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.availabilityAskedFor)
}

func (f *Api) ListInferenceProfiles(ctx context.Context) ([]types.InferenceProfileSummary, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.profileLists++
	if f.err != nil {
		return []types.InferenceProfileSummary{}, f.err
	}
	return cloneProfiles(f.profiles), nil
}

// ProfileLists is how many times ListInferenceProfiles was called
func (f *Api) ProfileLists() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.profileLists
}
