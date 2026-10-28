// Package ec2fake is an in-memory awsapi.Ec2Api for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package ec2fake

import (
	"context"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Api struct {
	awsapi.Ec2Api

	mtx       sync.Mutex
	instances []types.Instance
	err       error
	listed    int
}

type Option func(*Api)

func WithInstances(instances []types.Instance) Option {
	return func(f *Api) { f.instances = instances }
}
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetInstances(instances []types.Instance) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.instances = instances
}

// InstanceListings is how many times the instances were listed.
func (f *Api) InstanceListings() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.listed
}

func (f *Api) ListInstances(ctx context.Context, force bool) ([]types.Instance, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.listed++
	return f.instances, f.err
}
