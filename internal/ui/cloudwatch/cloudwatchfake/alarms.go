package cloudwatchfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// AlarmsApi is an in-memory awsapi.CloudWatchAlarmsApi
type AlarmsApi struct {
	awsapi.CloudWatchAlarmsApi

	mtx     sync.Mutex
	alarms  []cwTypes.MetricAlarm
	history []cwTypes.AlarmHistoryItem
	err     error
}

type AlarmsOption func(*AlarmsApi)

// WithAlarms sets the alarms ListAlarms gives
func WithAlarms(alarms []cwTypes.MetricAlarm) AlarmsOption {
	return func(f *AlarmsApi) { f.alarms = alarms }
}

// WithAlarmHistory sets the history ListAlarmHistory gives, for any alarm
func WithAlarmHistory(history []cwTypes.AlarmHistoryItem) AlarmsOption {
	return func(f *AlarmsApi) { f.history = history }
}

// WithAlarmsErr makes every call fail
func WithAlarmsErr(err error) AlarmsOption { return func(f *AlarmsApi) { f.err = err } }

func NewAlarms(options ...AlarmsOption) *AlarmsApi {
	var f = &AlarmsApi{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call fail. Safe while background loads run.
func (f *AlarmsApi) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

func (f *AlarmsApi) ListAlarms(ctx context.Context, force bool) ([]cwTypes.MetricAlarm, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.alarms), f.err
}

func (f *AlarmsApi) ListAlarmHistory(
	ctx context.Context, name string, force bool,
) ([]cwTypes.AlarmHistoryItem, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.history), f.err
}
