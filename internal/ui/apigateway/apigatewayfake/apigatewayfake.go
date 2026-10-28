// Package apigatewayfake is an in-memory awsapi.ApiGatewayApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package apigatewayfake

import (
	"context"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"

	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

type Api struct {
	awsapi.ApiGatewayApi

	mtx    sync.Mutex
	apis   []types.RestApi
	err    error
	listed int

	resources        map[string][]types.Resource
	resourceListings int

	invokeOutput *apigateway.TestInvokeMethodOutput
	invocations  []Invocation
}

// Invocation is a call of TestInvokeMethod.
type Invocation struct {
	RestApiId           string
	ResourceId          string
	HttpMethod          string
	PathWithQueryString string
	Body                string
	Headers             map[string]string
}

type Option func(*Api)

func WithApis(apis []types.RestApi) Option {
	return func(f *Api) { f.apis = apis }
}

// WithResources sets the resources of the API with the id.
func WithResources(restApiId string, resources []types.Resource) Option {
	return func(f *Api) { f.setResources(restApiId, resources) }
}

// WithInvokeOutput sets what testing a method returns.
func WithInvokeOutput(output *apigateway.TestInvokeMethodOutput) Option {
	return func(f *Api) { f.invokeOutput = output }
}
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetApis(apis []types.RestApi) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.apis = apis
}

// ApiListings is how many times the APIs were listed.
func (f *Api) ApiListings() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.listed
}

func (f *Api) ListRestApis(ctx context.Context, force bool) ([]types.RestApi, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.listed++
	return f.apis, f.err
}

func (f *Api) setResources(restApiId string, resources []types.Resource) {
	if f.resources == nil {
		f.resources = map[string][]types.Resource{}
	}
	f.resources[restApiId] = resources
}

func (f *Api) SetResources(restApiId string, resources []types.Resource) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.setResources(restApiId, resources)
}

// ResourceListings is how many times the resources of an API were listed.
func (f *Api) ResourceListings() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.resourceListings
}

func (f *Api) ListResources(ctx context.Context, force bool, restApiId string) ([]types.Resource, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.resourceListings++
	return f.resources[restApiId], f.err
}

// Invocations are the calls of TestInvokeMethod, in order.
func (f *Api) Invocations() []Invocation {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return append([]Invocation(nil), f.invocations...)
}

func (f *Api) TestInvokeMethod(
	ctx context.Context,
	restApiId string,
	resourceId string,
	httpMethod string,
	pathWithQueryString string,
	body string,
	headers map[string]string,
) (*apigateway.TestInvokeMethodOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.invocations = append(f.invocations, Invocation{
		restApiId, resourceId, httpMethod, pathWithQueryString, body, headers,
	})
	return f.invokeOutput, f.err
}
