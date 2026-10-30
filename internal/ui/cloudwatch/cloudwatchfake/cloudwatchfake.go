// Package cloudwatchfake is an in-memory awsapi.CloudWatchLogsApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set and Queue methods afterwards. UI code calls the
// fake from background goroutines, so fields must never be touched directly.
package cloudwatchfake

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type Api struct {
	awsapi.CloudWatchLogsApi

	mtx         sync.Mutex
	events      []types.OutputLogEvent
	streams     []types.LogStream
	moreEvents  bool
	moreStreams bool
	followed    [][]types.OutputLogEvent
	followErr   error
	calls       []string
	unmask      bool
}

type Option func(*Api)

// WithLogEvents sets the events ListLogEvents gives
func WithLogEvents(events ...types.OutputLogEvent) Option {
	return func(f *Api) { f.events = events }
}

// WithLogStreams sets the streams ListLogStreams gives
func WithLogStreams(streams ...types.LogStream) Option {
	return func(f *Api) { f.streams = streams }
}

// WithMoreLogEvents and WithMoreLogStreams make the api say it has more pages
func WithMoreLogEvents(more bool) Option  { return func(f *Api) { f.moreEvents = more } }
func WithMoreLogStreams(more bool) Option { return func(f *Api) { f.moreStreams = more } }

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

// SetMoreLogEvents changes whether the api says it has more events
func (f *Api) SetMoreLogEvents(more bool) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.moreEvents = more
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

func (f *Api) SetUnmaskLogEvents(unmask bool) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.unmask = unmask
	f.calls = append(f.calls, "unmask "+strconv.FormatBool(unmask))
}

func (f *Api) HasMoreLogEvents() bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.moreEvents
}

func (f *Api) HasMoreLogStreams() bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.moreStreams
}

func (f *Api) ListLogStreams(
	ctx context.Context, logGroupName string, searchPrefix string, reset bool,
) ([]types.LogStream, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "streams "+logGroupName)
	return slices.Clone(f.streams), nil
}

func (f *Api) ListLogEvents(
	ctx context.Context, logGroupName string, logStreamName string, startTime time.Time, reset bool,
) ([]types.OutputLogEvent, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var call = "list " + logGroupName + " " + logStreamName
	if !startTime.IsZero() {
		call += " from " + startTime.UTC().Format(time.RFC3339)
	}
	f.calls = append(f.calls, call)
	return slices.Clone(f.events), nil
}

// ListFilteredLogEvents gives the events that were set with WithLogEvents and
// contain the filter pattern, which is a plain text match in the fake.
func (f *Api) ListFilteredLogEvents(
	ctx context.Context, logGroupName string, logStreamName string, filterPattern string,
	startTime time.Time, reset bool,
) ([]types.OutputLogEvent, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var call = "filter " + logGroupName + " " + logStreamName + " " + filterPattern
	if !startTime.IsZero() {
		call += " from " + startTime.UTC().Format(time.RFC3339)
	}
	f.calls = append(f.calls, call)

	var matching []types.OutputLogEvent
	for _, event := range f.events {
		if strings.Contains(aws.ToString(event.Message), filterPattern) {
			matching = append(matching, event)
		}
	}
	return matching, nil
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
