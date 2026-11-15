package cloudwatchfake

import (
	"context"
	"slices"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// DataRequest is what GetMetricData was asked for
type DataRequest struct {
	MetricName string
	Stat       string
	Period     int32
	Start      time.Time
	End        time.Time
}

// MetricsApi is an in-memory awsapi.CloudWatchMetricsApi
type MetricsApi struct {
	awsapi.CloudWatchMetricsApi

	mtx     sync.Mutex
	metrics []cwTypes.Metric
	err     error
	data    cwTypes.MetricDataResult
	dataErr error

	requests []DataRequest
}

type MetricsOption func(*MetricsApi)

// WithMetrics sets the metrics ListMetrics gives
func WithMetrics(metrics []cwTypes.Metric) MetricsOption {
	return func(f *MetricsApi) { f.metrics = metrics }
}

// WithMetricsErr makes ListMetrics fail
func WithMetricsErr(err error) MetricsOption { return func(f *MetricsApi) { f.err = err } }

// WithMetricData sets the data GetMetricData gives, for any metric
func WithMetricData(data cwTypes.MetricDataResult) MetricsOption {
	return func(f *MetricsApi) { f.data = data }
}

// WithMetricDataErr makes GetMetricData fail
func WithMetricDataErr(err error) MetricsOption { return func(f *MetricsApi) { f.dataErr = err } }

func NewMetrics(options ...MetricsOption) *MetricsApi {
	var f = &MetricsApi{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetDataErr makes every following call of GetMetricData fail. Safe while
// background loads run.
func (f *MetricsApi) SetDataErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.dataErr = err
}

// DataRequests gives what GetMetricData was asked for, oldest first
func (f *MetricsApi) DataRequests() []DataRequest {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.requests)
}

// Charted gives the names of the metrics whose data was asked for, oldest
// first
func (f *MetricsApi) Charted() []string {
	var names []string
	for _, request := range f.DataRequests() {
		names = append(names, request.MetricName)
	}
	return names
}

func (f *MetricsApi) GetMetricData(
	ctx context.Context, metric cwTypes.Metric, stat string, periodSeconds int32, start time.Time, end time.Time,
) (cwTypes.MetricDataResult, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.requests = append(f.requests, DataRequest{
		MetricName: aws.ToString(metric.MetricName), Stat: stat, Period: periodSeconds, Start: start, End: end,
	})
	var data = f.data
	data.Timestamps = slices.Clone(data.Timestamps)
	data.Values = slices.Clone(data.Values)
	return data, f.dataErr
}

func (f *MetricsApi) ListMetrics(
	ctx context.Context, dims []cwTypes.DimensionFilter, namespace string, metricName string, force bool,
) ([]cwTypes.Metric, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.metrics), f.err
}
