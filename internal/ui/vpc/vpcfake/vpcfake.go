// Package vpcfake is an in-memory awsapi.Ec2Api for the VPC tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package vpcfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Api struct {
	awsapi.Ec2Api

	mtx         sync.Mutex
	networkAcls []types.NetworkAcl
	err         error
	aclListings int
}

type Option func(*Api)

func WithNetworkAcls(acls []types.NetworkAcl) Option {
	return func(f *Api) { f.networkAcls = slices.Clone(acls) }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetNetworkAcls(acls []types.NetworkAcl) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.networkAcls = slices.Clone(acls)
}

func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// NetworkAclListings is how many times the network ACLs were listed.
func (f *Api) NetworkAclListings() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.aclListings
}

func (f *Api) ListNetworkAcls(ctx context.Context, force bool) ([]types.NetworkAcl, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.aclListings++
	return slices.Clone(f.networkAcls), f.err
}
