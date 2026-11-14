// Package ssmfake is an in-memory awsapi.SystemsManagerApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package ssmfake

import (
	"context"
	"fmt"
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

// ValueRequest is one version of a parameter, which is 0 for the latest one.
type ValueRequest struct {
	Name    string
	Version int64
}

type Api struct {
	awsapi.SystemsManagerApi

	mtx           sync.Mutex
	parameters    []types.Parameter
	history       []types.ParameterHistory
	values        map[ValueRequest]string
	valueRequests []ValueRequest
	valueErr      error
	err           error
	putErr        error
	createErr     error
	deleteErr     error
	puts          []Put
	creates       []Put
	deleted       []string
}

type Option func(*Api)

func WithParameters(parameters []types.Parameter) Option {
	return func(f *Api) { f.parameters = parameters }
}
func WithHistory(history []types.ParameterHistory) Option {
	return func(f *Api) { f.history = history }
}

// WithValue is the decrypted value of the version of the parameter, which is
// only given by GetParameterValue.
func WithValue(name string, version int64, value string) Option {
	return func(f *Api) { f.values[ValueRequest{Name: name, Version: version}] = value }
}

// WithValueErr is the error of getting a decrypted value.
func WithValueErr(err error) Option { return func(f *Api) { f.valueErr = err } }

// WithErr is the error of listing the parameters and of getting the history.
func WithErr(err error) Option    { return func(f *Api) { f.err = err } }
func WithPutErr(err error) Option { return func(f *Api) { f.putErr = err } }

func WithCreateErr(err error) Option { return func(f *Api) { f.createErr = err } }

func WithDeleteErr(err error) Option { return func(f *Api) { f.deleteErr = err } }

func New(options ...Option) *Api {
	var f = &Api{values: map[ValueRequest]string{}}
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

// Deleted returns the name of every parameter deleted so far.
func (f *Api) Deleted() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.deleted)
}

func (f *Api) Parameters() []types.Parameter {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.parameters
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

// GetParameterValue records the call and gives the value set with WithValue.
func (f *Api) GetParameterValue(ctx context.Context, name string, version int64) (string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var request = ValueRequest{Name: name, Version: version}
	f.valueRequests = append(f.valueRequests, request)
	if f.valueErr != nil {
		return "", f.valueErr
	}
	var value, found = f.values[request]
	if !found {
		return "", fmt.Errorf("no value for version %d of %s", version, name)
	}
	return value, nil
}

// ValueRequests are the GetParameterValue calls so far.
func (f *Api) ValueRequests() []ValueRequest {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.valueRequests)
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

func (f *Api) DeleteParameter(ctx context.Context, name string) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, name)
	return nil
}
