// Package xrayfake is an in-memory awsapi.XRayApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package xrayfake

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

type Api struct {
	awsapi.XRayApi

	mtx sync.Mutex
	// pages are the pages of the trace summaries, newest first, of any span
	// without pages of its own in spanPages and any filter without pages of
	// its own in filterPages, which come before those of the span
	pages       [][]types.TraceSummary
	spanPages   map[time.Duration][][]types.TraceSummary
	filterPages map[string][][]types.TraceSummary
	// current are the pages being gone through, which a reset picks
	current  [][]types.TraceSummary
	nextPage int
	// holds keep the loads of a span waiting until they are released
	holds map[time.Duration]chan struct{}
	// windowsAskedFor are the times and filters of the loads of trace
	// summaries, in order
	windowsAskedFor []Window
	traces          map[string]types.Trace
	rules           []types.SamplingRuleRecord
	services        []types.Service
	spanServices    map[time.Duration][]types.Service
	// graphWindowsAskedFor are the times of the loads of the service graph
	graphWindowsAskedFor []Window
	err                  error
	// tracesAskedFor is the IDs of the traces that were loaded, in order
	tracesAskedFor []string
}

// Window is the times traces were asked for between, and the filter they
// were asked for with
type Window struct {
	Start, End time.Time
	Filter     string
}

type Option func(*Api)

// WithTracePages gives the pages of the trace summaries, newest first, for
// every span without pages of its own
func WithTracePages(pages ...[]types.TraceSummary) Option {
	return func(f *Api) { f.pages = pages }
}

// WithTracePagesSince gives the pages of the trace summaries of the windows
// that are as long as the span
func WithTracePagesSince(span time.Duration, pages ...[]types.TraceSummary) Option {
	return func(f *Api) { f.spanPages[span] = pages }
}

// WithTracePagesFiltered gives the pages of the trace summaries of the
// filter expression, whatever the span
func WithTracePagesFiltered(filter string, pages ...[]types.TraceSummary) Option {
	return func(f *Api) { f.filterPages[filter] = pages }
}

// WithTrace gives a trace with its segments, by its ID
func WithTrace(trace types.Trace) Option {
	return func(f *Api) { f.traces[aws.ToString(trace.Id)] = trace }
}

func WithSamplingRules(rules []types.SamplingRuleRecord) Option {
	return func(f *Api) { f.rules = rules }
}

// WithServiceGraph gives the services of the service graph of any span
// without services of its own
func WithServiceGraph(services []types.Service) Option {
	return func(f *Api) { f.services = services }
}

// WithServiceGraphSince gives the services of the service graph of the
// windows that are as long as the span
func WithServiceGraphSince(span time.Duration, services []types.Service) Option {
	return func(f *Api) { f.spanServices[span] = services }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{
		traces:       map[string]types.Trace{},
		spanPages:    map[time.Duration][][]types.TraceSummary{},
		filterPages:  map[string][][]types.TraceSummary{},
		spanServices: map[time.Duration][]types.Service{},
		holds:        map[time.Duration]chan struct{}{},
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

// HoldSpan keeps the loads of the trace summaries and of the service graph of
// the windows that are as long as the span waiting from now until release is
// called
func (f *Api) HoldSpan(span time.Duration) (release func()) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var hold = make(chan struct{})
	f.holds[span] = hold
	return sync.OnceFunc(func() { close(hold) })
}

// WindowsAskedFor are the times and filters of the loads of trace summaries, in order
func (f *Api) WindowsAskedFor() []Window {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.windowsAskedFor)
}

// SpansAskedFor are how long the windows of the loads of trace summaries are,
// in order
func (f *Api) SpansAskedFor() []time.Duration {
	var spans []time.Duration
	for _, window := range f.WindowsAskedFor() {
		spans = append(spans, window.End.Sub(window.Start))
	}
	return spans
}

// FiltersAskedFor are the filters of the loads of trace summaries, in order
func (f *Api) FiltersAskedFor() []string {
	var filters []string
	for _, window := range f.WindowsAskedFor() {
		filters = append(filters, window.Filter)
	}
	return filters
}

func (f *Api) GetTraceSummaries(
	ctx context.Context, start, end time.Time, filter string, reset bool,
) ([]types.TraceSummary, error) {
	var span = end.Sub(start)
	f.mtx.Lock()
	f.windowsAskedFor = append(f.windowsAskedFor, Window{Start: start, End: end, Filter: filter})
	var hold = f.holds[span]
	f.mtx.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return []types.TraceSummary{}, ctx.Err()
		}
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return []types.TraceSummary{}, f.err
	}
	if reset || f.current == nil {
		f.current = f.pages
		if pages, ok := f.spanPages[span]; ok {
			f.current = pages
		}
		if pages, ok := f.filterPages[filter]; ok {
			f.current = pages
		}
		f.nextPage = 0
	}
	if f.nextPage >= len(f.current) {
		return []types.TraceSummary{}, nil
	}
	var page = slices.Clone(f.current[f.nextPage])
	f.nextPage++
	return page, nil
}

// TracesAskedFor is the IDs of the traces that were loaded, in order
func (f *Api) TracesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.tracesAskedFor)
}

func (f *Api) GetTrace(ctx context.Context, traceId string) (types.Trace, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.tracesAskedFor = append(f.tracesAskedFor, traceId)
	if f.err != nil {
		return types.Trace{}, f.err
	}
	var trace, ok = f.traces[traceId]
	if !ok {
		return types.Trace{}, errors.New("trace " + traceId + " not found")
	}
	trace.Segments = slices.Clone(trace.Segments)
	return trace, nil
}

func (f *Api) GetSamplingRules(ctx context.Context, force bool) ([]types.SamplingRuleRecord, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.rules), f.err
}

// GraphWindowsAskedFor are the times of the loads of the service graph, in
// order
func (f *Api) GraphWindowsAskedFor() []Window {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.graphWindowsAskedFor)
}

func (f *Api) GetServiceGraph(ctx context.Context, start, end time.Time) ([]types.Service, error) {
	var span = end.Sub(start)
	f.mtx.Lock()
	f.graphWindowsAskedFor = append(f.graphWindowsAskedFor, Window{Start: start, End: end})
	var hold = f.holds[span]
	f.mtx.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return []types.Service{}, ctx.Err()
		}
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return []types.Service{}, f.err
	}
	if services, ok := f.spanServices[span]; ok {
		return slices.Clone(services), nil
	}
	return slices.Clone(f.services), nil
}
