// Package waffake is an in-memory awsapi.WafApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package waffake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

type Api struct {
	awsapi.WafApi

	mtx     sync.Mutex
	webAcls []types.WebACLSummary
	// webAclDetails are what GetWebACL gives, by ARN
	webAclDetails map[string]types.WebACL
	// resources are what ListWebACLResources gives, by web ACL ARN
	resources map[string][]awsapi.WebACLResource
	ipSets    []types.IPSetSummary
	// ipSetDetails are what GetIPSet gives, by ARN
	ipSetDetails map[string]types.IPSet
	err          error
	// webAclsAskedFor is the ARNs GetWebACL was called with, in order
	webAclsAskedFor []string
	// resourcesAskedFor is the ARNs ListWebACLResources was called with, in
	// order
	resourcesAskedFor []string
	// ipSetsAskedFor is the ARNs GetIPSet was called with, in order
	ipSetsAskedFor []string
}

type Option func(*Api)

func WithWebACLs(acls []types.WebACLSummary) Option {
	return func(f *Api) { f.webAcls = slices.Clone(acls) }
}

// WithWebACL gives what GetWebACL returns for the ARN
func WithWebACL(webAclArn string, acl types.WebACL) Option {
	return func(f *Api) { f.webAclDetails[webAclArn] = acl }
}

// WithResources gives what ListWebACLResources returns for the web ACL ARN
func WithResources(webAclArn string, resources []awsapi.WebACLResource) Option {
	return func(f *Api) { f.resources[webAclArn] = slices.Clone(resources) }
}

func WithIPSets(sets []types.IPSetSummary) Option {
	return func(f *Api) { f.ipSets = slices.Clone(sets) }
}

// WithIPSet gives what GetIPSet returns for the ARN
func WithIPSet(ipSetArn string, set types.IPSet) Option {
	return func(f *Api) { f.ipSetDetails[ipSetArn] = set }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{
		webAclDetails: map[string]types.WebACL{},
		resources:     map[string][]awsapi.WebACLResource{},
		ipSetDetails:  map[string]types.IPSet{},
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call fail
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

func (f *Api) SetWebACLs(acls []types.WebACLSummary) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.webAcls = slices.Clone(acls)
}

func (f *Api) ListWebACLs(ctx context.Context) ([]types.WebACLSummary, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.webAcls), nil
}

// GetWebACL gives the web ACL of the ARN, or one without rules when it wasn't
// given. Its rules are copied so the test and the UI never share them.
func (f *Api) GetWebACL(ctx context.Context, webAclArn string) (*types.WebACL, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.webAclsAskedFor = append(f.webAclsAskedFor, webAclArn)
	if f.err != nil {
		return nil, f.err
	}
	var acl, ok = f.webAclDetails[webAclArn]
	if !ok {
		acl = types.WebACL{ARN: aws.String(webAclArn)}
	}
	acl.Rules = slices.Clone(acl.Rules)
	acl.TokenDomains = slices.Clone(acl.TokenDomains)
	return &acl, nil
}

// WebACLsAskedFor is the ARNs GetWebACL was called with, in order
func (f *Api) WebACLsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.webAclsAskedFor)
}

func (f *Api) ListWebACLResources(ctx context.Context, webAclArn string) ([]awsapi.WebACLResource, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.resourcesAskedFor = append(f.resourcesAskedFor, webAclArn)
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.resources[webAclArn]), nil
}

// ResourcesAskedFor is the ARNs ListWebACLResources was called with, in order
func (f *Api) ResourcesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.resourcesAskedFor)
}

func (f *Api) ListIPSets(ctx context.Context) ([]types.IPSetSummary, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.ipSets), nil
}

// GetIPSet gives the IP set of the ARN, or one without addresses when it
// wasn't given. Its addresses are copied so the test and the UI never share
// them.
func (f *Api) GetIPSet(ctx context.Context, ipSetArn string) (*types.IPSet, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.ipSetsAskedFor = append(f.ipSetsAskedFor, ipSetArn)
	if f.err != nil {
		return nil, f.err
	}
	var set, ok = f.ipSetDetails[ipSetArn]
	if !ok {
		set = types.IPSet{ARN: aws.String(ipSetArn)}
	}
	set.Addresses = slices.Clone(set.Addresses)
	return &set, nil
}

// IPSetsAskedFor is the ARNs GetIPSet was called with, in order
func (f *Api) IPSetsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.ipSetsAskedFor)
}
