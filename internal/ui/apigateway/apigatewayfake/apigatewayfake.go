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

	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

type Api struct {
	awsapi.ApiGatewayApi

	mtx    sync.Mutex
	apis   []types.RestApi
	err    error
	listed int
}

type Option func(*Api)

func WithApis(apis []types.RestApi) Option {
	return func(f *Api) { f.apis = apis }
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
