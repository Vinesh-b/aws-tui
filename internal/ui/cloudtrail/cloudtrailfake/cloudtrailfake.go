// Package cloudtrailfake is an in-memory awsapi.CloudTrailApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package cloudtrailfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
)

type Api struct {
	awsapi.CloudTrailApi

	mtx sync.Mutex
	// pages are the pages of the event history, newest first
	pages    [][]types.Event
	nextPage int
	trails   []types.Trail
	statuses map[string]*cloudtrail.GetTrailStatusOutput
	err      error
	// statusesAskedFor is the ARNs of the trails whose status was loaded, in
	// order
	statusesAskedFor []string
}

type Option func(*Api)

// WithEventPages gives the pages of the event history, newest first
func WithEventPages(pages ...[]types.Event) Option { return func(f *Api) { f.pages = pages } }
func WithTrails(trails []types.Trail) Option       { return func(f *Api) { f.trails = trails } }

// WithStatus gives the status of a trail by its ARN
func WithStatus(trailArn string, status *cloudtrail.GetTrailStatusOutput) Option {
	return func(f *Api) { f.statuses[trailArn] = status }
}
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{statuses: map[string]*cloudtrail.GetTrailStatusOutput{}}
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

func (f *Api) LookupEvents(ctx context.Context, reset bool) ([]types.Event, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return []types.Event{}, f.err
	}
	if reset {
		f.nextPage = 0
	}
	if f.nextPage >= len(f.pages) {
		return []types.Event{}, nil
	}
	var page = slices.Clone(f.pages[f.nextPage])
	f.nextPage++
	return page, nil
}

func (f *Api) DescribeTrails(ctx context.Context, force bool) ([]types.Trail, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.trails), f.err
}

// StatusesAskedFor is the ARNs of the trails whose status was loaded, in order
func (f *Api) StatusesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.statusesAskedFor)
}

func (f *Api) GetTrailStatus(ctx context.Context, trailArn string) (*cloudtrail.GetTrailStatusOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.statusesAskedFor = append(f.statusesAskedFor, trailArn)
	if f.err != nil {
		return nil, f.err
	}
	if status, ok := f.statuses[trailArn]; ok {
		var copied = *status
		return &copied, nil
	}
	return &cloudtrail.GetTrailStatusOutput{IsLogging: aws.Bool(false)}, nil
}
