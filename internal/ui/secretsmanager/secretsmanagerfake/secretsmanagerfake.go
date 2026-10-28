// Package secretsmanagerfake is an in-memory awsapi.SecretsManagerApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package secretsmanagerfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// Put is a value that was stored for a secret.
type Put struct {
	SecretId string
	Value    string
}

type Api struct {
	awsapi.SecretsManagerApi

	mtx     sync.Mutex
	secrets []types.SecretListEntry
	value   string
	err     error
	putErr  error
	puts    []Put
}

type Option func(*Api)

func WithSecrets(secrets []types.SecretListEntry) Option {
	return func(f *Api) { f.secrets = secrets }
}
func WithValue(value string) Option { return func(f *Api) { f.value = value } }

// WithErr is the error of listing the secrets and of getting a value.
func WithErr(err error) Option    { return func(f *Api) { f.err = err } }
func WithPutErr(err error) Option { return func(f *Api) { f.putErr = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetSecrets(secrets []types.SecretListEntry) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.secrets = secrets
}

func (f *Api) SetValue(value string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.value = value
}

// Puts returns every value stored so far.
func (f *Api) Puts() []Put {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.puts)
}

func (f *Api) ListSecrets(ctx context.Context, reset bool) ([]types.SecretListEntry, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.secrets, f.err
}

func (f *Api) GetSecretValue(ctx context.Context, secretId string) (*secretsmanager.GetSecretValueOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(f.value)}, nil
}

func (f *Api) PutSecretValue(ctx context.Context, secretId string, value string) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.putErr != nil {
		return f.putErr
	}
	f.puts = append(f.puts, Put{SecretId: secretId, Value: value})
	return nil
}
