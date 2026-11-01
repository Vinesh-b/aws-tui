// Package schedulerfake is an in-memory awsapi.SchedulerApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package schedulerfake

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

var ErrNotFound = errors.New("schedule not found")

func ScheduleArn(group string, name string) string {
	return "arn:aws:scheduler:us-east-1:123456789012:schedule/" + group + "/" + name
}

type Api struct {
	awsapi.SchedulerApi

	mtx       sync.Mutex
	schedules []scheduler.GetScheduleOutput
	err       error
	getErr    error
	calls     []string
	creates   []scheduler.CreateScheduleInput
	updates   []scheduler.UpdateScheduleInput
}

type Option func(*Api)

// WithSchedules stores the schedules. Only the group, name, state, target and
// dates are listed, the rest is only returned by GetSchedule.
func WithSchedules(schedules ...scheduler.GetScheduleOutput) Option {
	return func(f *Api) { f.schedules = slices.Clone(schedules) }
}

// WithErr makes every call fail.
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

// SetGetErr makes only the following GetSchedule calls fail.
func (f *Api) SetGetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.getErr = err
}

// Calls returns every call made so far, formatted as "<method> <args...>".
func (f *Api) Calls() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

// CallsWithPrefix returns the calls whose text starts with prefix, e.g. "delete ".
func (f *Api) CallsWithPrefix(prefix string) []string {
	return slices.DeleteFunc(f.Calls(), func(call string) bool { return !strings.HasPrefix(call, prefix) })
}

// Creates returns the inputs CreateSchedule was called with.
func (f *Api) Creates() []scheduler.CreateScheduleInput {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.creates)
}

// Updates returns the inputs UpdateSchedule was called with.
func (f *Api) Updates() []scheduler.UpdateScheduleInput {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.updates)
}

func (f *Api) record(call string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, call)
}

// index is the position of the schedule, or -1. The lock must be held.
func (f *Api) index(group string, name string) int {
	return slices.IndexFunc(f.schedules, func(s scheduler.GetScheduleOutput) bool {
		return aws.ToString(s.GroupName) == group && aws.ToString(s.Name) == name
	})
}

func groupOrDefault(group *string) string {
	if aws.ToString(group) == "" {
		return "default"
	}
	return *group
}

func (f *Api) ListSchedules(ctx context.Context) ([]types.ScheduleSummary, error) {
	f.record("list")
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}

	var result = make([]types.ScheduleSummary, 0, len(f.schedules))
	for _, s := range f.schedules {
		var summary = types.ScheduleSummary{
			Arn:                  s.Arn,
			Name:                 s.Name,
			GroupName:            s.GroupName,
			State:                s.State,
			CreationDate:         s.CreationDate,
			LastModificationDate: s.LastModificationDate,
		}
		if s.Target != nil {
			summary.Target = &types.TargetSummary{Arn: s.Target.Arn}
		}
		result = append(result, summary)
	}
	slices.SortStableFunc(result, func(a, b types.ScheduleSummary) int {
		return cmp.Or(
			cmp.Compare(aws.ToString(a.GroupName), aws.ToString(b.GroupName)),
			cmp.Compare(aws.ToString(a.Name), aws.ToString(b.Name)),
		)
	})
	return result, nil
}

func (f *Api) GetSchedule(ctx context.Context, group string, name string) (scheduler.GetScheduleOutput, error) {
	f.record("get " + group + "/" + name)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if err := cmp.Or(f.err, f.getErr); err != nil {
		return scheduler.GetScheduleOutput{}, err
	}
	var idx = f.index(group, name)
	if idx < 0 {
		return scheduler.GetScheduleOutput{}, ErrNotFound
	}
	return cloneSchedule(f.schedules[idx]), nil
}

// cloneSchedule copies the parts of the schedule that are pointers to structs,
// so callers can't change what is stored
func cloneSchedule(s scheduler.GetScheduleOutput) scheduler.GetScheduleOutput {
	if s.Target != nil {
		var target = *s.Target
		if target.RetryPolicy != nil {
			var retry = *target.RetryPolicy
			target.RetryPolicy = &retry
		}
		if target.DeadLetterConfig != nil {
			var dlq = *target.DeadLetterConfig
			target.DeadLetterConfig = &dlq
		}
		s.Target = &target
	}
	if s.FlexibleTimeWindow != nil {
		var window = *s.FlexibleTimeWindow
		s.FlexibleTimeWindow = &window
	}
	return s
}

func (f *Api) CreateSchedule(ctx context.Context, input scheduler.CreateScheduleInput) (string, error) {
	var group = groupOrDefault(input.GroupName)
	var name = aws.ToString(input.Name)
	f.record("create " + group + "/" + name)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.creates = append(f.creates, input)
	if f.err != nil {
		return "", f.err
	}
	if f.index(group, name) >= 0 {
		return "", fmt.Errorf("schedule %s already exists", name)
	}

	var arn = ScheduleArn(group, name)
	f.schedules = append(f.schedules, scheduler.GetScheduleOutput{
		Arn:                        &arn,
		Name:                       &name,
		GroupName:                  &group,
		Description:                input.Description,
		ScheduleExpression:         input.ScheduleExpression,
		ScheduleExpressionTimezone: input.ScheduleExpressionTimezone,
		FlexibleTimeWindow:         input.FlexibleTimeWindow,
		State:                      cmp.Or(input.State, types.ScheduleStateEnabled),
		Target:                     input.Target,
	})
	return arn, nil
}

func (f *Api) UpdateSchedule(ctx context.Context, input scheduler.UpdateScheduleInput) (string, error) {
	var group = groupOrDefault(input.GroupName)
	var name = aws.ToString(input.Name)
	f.record("update " + group + "/" + name)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.updates = append(f.updates, input)
	if f.err != nil {
		return "", f.err
	}
	var idx = f.index(group, name)
	if idx < 0 {
		return "", ErrNotFound
	}

	var stored = &f.schedules[idx]
	stored.Description = input.Description
	stored.ScheduleExpression = input.ScheduleExpression
	stored.ScheduleExpressionTimezone = input.ScheduleExpressionTimezone
	stored.FlexibleTimeWindow = input.FlexibleTimeWindow
	stored.State = cmp.Or(input.State, types.ScheduleStateEnabled)
	stored.Target = input.Target
	stored.StartDate = input.StartDate
	stored.EndDate = input.EndDate
	stored.KmsKeyArn = input.KmsKeyArn
	stored.ActionAfterCompletion = input.ActionAfterCompletion
	return aws.ToString(stored.Arn), nil
}

func (f *Api) DeleteSchedule(ctx context.Context, group string, name string) error {
	f.record("delete " + group + "/" + name)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return f.err
	}
	var idx = f.index(group, name)
	if idx < 0 {
		return ErrNotFound
	}
	f.schedules = slices.Delete(f.schedules, idx, idx+1)
	return nil
}
