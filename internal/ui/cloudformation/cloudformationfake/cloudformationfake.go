// Package cloudformationfake is an in-memory awsapi.CloudFormationApi for
// tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package cloudformationfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type Api struct {
	awsapi.CloudFormationApi

	mtx    sync.Mutex
	stacks []types.StackSummary
	events []types.StackEvent
	err    error
	calls  []string
}

type Option func(*Api)

// WithStacks sets the stacks ListStacks gives
func WithStacks(stacks []types.StackSummary) Option { return func(f *Api) { f.stacks = stacks } }

// WithEvents sets the events DescribeStackEvents gives, for any stack
func WithEvents(events []types.StackEvent) Option { return func(f *Api) { f.events = events } }

// WithErr makes every call fail
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call fail. Safe while background loads run.
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// Calls returns every call made so far, formatted as "<method> <args...>".
func (f *Api) Calls() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

func (f *Api) ListStacks(ctx context.Context, force bool) ([]types.StackSummary, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "stacks")
	return slices.Clone(f.stacks), f.err
}

func (f *Api) DescribeStackEvents(ctx context.Context, stackName string, force bool) ([]types.StackEvent, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "events "+stackName)
	return slices.Clone(f.events), f.err
}
