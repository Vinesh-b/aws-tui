// Package sfnfake is an in-memory awsapi.StateMachineApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package sfnfake

import (
	"context"
	"slices"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

// StartedExecutionArn is the ARN StartExecution gives the executions it starts
const StartedExecutionArn = "arn:execution:run-1"

// Started is an execution that StartExecution was asked to start
type Started struct {
	StateMachineArn string
	Input           string
}

type Api struct {
	awsapi.StateMachineApi

	mtx        sync.Mutex
	machines   []types.StateMachineListItem
	executions []types.ExecutionListItem
	execution  *sfn.DescribeExecutionOutput
	history    *sfn.GetExecutionHistoryOutput
	err        error
	startErr   error
	started    []Started
}

type Option func(*Api)

// WithStateMachines sets the state machines ListStateMachines gives
func WithStateMachines(machines []types.StateMachineListItem) Option {
	return func(f *Api) { f.machines = machines }
}

// WithExecutions sets the executions ListExecutions gives, for any state machine
func WithExecutions(executions []types.ExecutionListItem) Option {
	return func(f *Api) { f.executions = executions }
}

// WithExecution sets what DescribeExecution gives, for any execution
func WithExecution(execution *sfn.DescribeExecutionOutput) Option {
	return func(f *Api) { f.execution = execution }
}

// WithHistory sets what GetExecutionHistory gives, for any execution
func WithHistory(history *sfn.GetExecutionHistoryOutput) Option {
	return func(f *Api) { f.history = history }
}

// WithErr makes every call but StartExecution fail
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

// WithStartErr makes StartExecution fail
func WithStartErr(err error) Option { return func(f *Api) { f.startErr = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call but StartExecution fail. Safe while
// background loads run.
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// Started gives the executions StartExecution was asked to start, oldest first
func (f *Api) Started() []Started {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.started)
}

func (f *Api) ListStateMachines(ctx context.Context, force bool) ([]types.StateMachineListItem, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.machines), f.err
}

func (f *Api) ListExecutions(
	ctx context.Context, stateMachineArn string, start time.Time, end time.Time, reset bool,
) ([]types.ExecutionListItem, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.executions), f.err
}

func (f *Api) DescribeExecution(ctx context.Context, executionArn string) (*sfn.DescribeExecutionOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.execution == nil {
		return nil, f.err
	}
	var execution = *f.execution
	return &execution, f.err
}

func (f *Api) GetExecutionHistory(ctx context.Context, executionArn string) (*sfn.GetExecutionHistoryOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.history == nil {
		return nil, f.err
	}
	var history = *f.history
	history.Events = slices.Clone(history.Events)
	return &history, f.err
}

func (f *Api) StartExecution(
	ctx context.Context, stateMachineArn string, name string, input string,
) (*sfn.StartExecutionOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.started = append(f.started, Started{StateMachineArn: stateMachineArn, Input: input})
	if f.startErr != nil {
		return nil, f.startErr
	}
	return &sfn.StartExecutionOutput{
		ExecutionArn: aws.String(StartedExecutionArn),
		StartDate:    aws.Time(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)),
	}, nil
}
