// Package ssmfake is an in-memory awsapi.SystemsManagerApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package ssmfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// Put is a value that was stored for a parameter, or a parameter that was
// created.
type Put struct {
	Name  string
	Value string
	Type  types.ParameterType
}

type Api struct {
	awsapi.SystemsManagerApi

	mtx        sync.Mutex
	parameters []types.Parameter
	history    []types.ParameterHistory
	err        error
	putErr     error
	createErr  error
	puts       []Put
	creates    []Put
}

type Option func(*Api)

func WithParameters(parameters []types.Parameter) Option {
	return func(f *Api) { f.parameters = parameters }
}
func WithHistory(history []types.ParameterHistory) Option {
	return func(f *Api) { f.history = history }
}

// WithErr is the error of listing the parameters and of getting the history.
func WithErr(err error) Option    { return func(f *Api) { f.err = err } }
func WithPutErr(err error) Option { return func(f *Api) { f.putErr = err } }

func WithCreateErr(err error) Option { return func(f *Api) { f.createErr = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetParameters(parameters []types.Parameter) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.parameters = parameters
}

// Puts returns every value stored so far.
func (f *Api) Puts() []Put {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.puts)
}

// Creates returns every parameter created so far.
func (f *Api) Creates() []Put {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.creates)
}

func (f *Api) GetParametersByPath(
	ctx context.Context, path string, reset bool,
) ([]types.Parameter, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.parameters, f.err
}

func (f *Api) GetParameterHistory(
	ctx context.Context, name string, reset bool,
) ([]types.ParameterHistory, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.history, f.err
}

func (f *Api) PutParameter(
	ctx context.Context, name string, value string, paramType types.ParameterType,
) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.putErr != nil {
		return f.putErr
	}
	f.puts = append(f.puts, Put{Name: name, Value: value, Type: paramType})
	return nil
}

func (f *Api) CreateParameter(
	ctx context.Context, name string, value string, paramType types.ParameterType,
) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.creates = append(f.creates, Put{Name: name, Value: value, Type: paramType})
	return nil
}
