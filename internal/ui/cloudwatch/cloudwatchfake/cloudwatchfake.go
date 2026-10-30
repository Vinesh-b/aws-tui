// Package cloudwatchfake is an in-memory awsapi.CloudWatchLogsApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set and Queue methods afterwards. UI code calls the
// fake from background goroutines, so fields must never be touched directly.
package cloudwatchfake

import (
	"context"
	"slices"
	"strings"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type Api struct {
	awsapi.CloudWatchLogsApi

	mtx       sync.Mutex
	events    []types.OutputLogEvent
	followed  [][]types.OutputLogEvent
	followErr error
	calls     []string
}

type Option func(*Api)

// WithLogEvents sets the events ListLogEvents gives
func WithLogEvents(events ...types.OutputLogEvent) Option {
	return func(f *Api) { f.events = events }
}

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

// QueueFollowedEvents makes a call of FollowLogEvents give these events, after
// those queued before. Calls with none queued give none.
func (f *Api) QueueFollowedEvents(events ...types.OutputLogEvent) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.followed = append(f.followed, events)
}

// SetFollowErr makes every following call of FollowLogEvents fail
func (f *Api) SetFollowErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.followErr = err
}

// Calls returns every call made so far, formatted as "<method> <args...>".
func (f *Api) Calls() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

// CallsWithPrefix returns the calls whose text starts with prefix, e.g. "follow ".
func (f *Api) CallsWithPrefix(prefix string) []string {
	return slices.DeleteFunc(f.Calls(), func(call string) bool { return !strings.HasPrefix(call, prefix) })
}

func (f *Api) ListLogEvents(
	ctx context.Context, logGroupName string, logStreamName string, reset bool,
) ([]types.OutputLogEvent, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "list "+logGroupName+" "+logStreamName)
	return slices.Clone(f.events), nil
}

func (f *Api) FollowLogEvents(
	ctx context.Context, logGroupName string, logStreamName string,
) ([]types.OutputLogEvent, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "follow "+logGroupName+" "+logStreamName)
	if f.followErr != nil {
		return nil, f.followErr
	}
	if len(f.followed) == 0 {
		return nil, nil
	}
	var events = f.followed[0]
	f.followed = f.followed[1:]
	return events, nil
}
