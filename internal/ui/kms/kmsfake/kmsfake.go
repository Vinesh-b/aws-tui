// Package kmsfake is an in-memory awsapi.KmsApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package kmsfake

import (
	"context"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type Api struct {
	awsapi.KmsApi

	mtx      sync.Mutex
	keys     []types.KeyListEntry
	aliases  []types.AliasListEntry
	metadata map[string]*types.KeyMetadata
	tags     []types.Tag
	policy   string
	err      error
	listed   int
}

type Option func(*Api)

func WithKeys(keys []types.KeyListEntry) Option { return func(f *Api) { f.keys = keys } }
func WithAliases(aliases []types.AliasListEntry) Option {
	return func(f *Api) { f.aliases = aliases }
}

// WithMetadata gives the description of a key by its ID.
func WithMetadata(keyId string, metadata *types.KeyMetadata) Option {
	return func(f *Api) { f.metadata[keyId] = metadata }
}
func WithTags(tags []types.Tag) Option { return func(f *Api) { f.tags = tags } }
func WithPolicy(policy string) Option  { return func(f *Api) { f.policy = policy } }
func WithErr(err error) Option         { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{metadata: map[string]*types.KeyMetadata{}}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetKeys(keys []types.KeyListEntry) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.keys = keys
}

func (f *Api) Keys() []types.KeyListEntry {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.keys
}

// KeyListings is how many times the keys were listed.
func (f *Api) KeyListings() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.listed
}

func (f *Api) ListKeys(ctx context.Context, force bool) ([]types.KeyListEntry, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.listed++
	return f.keys, f.err
}

func (f *Api) ListAliases(ctx context.Context, force bool) ([]types.AliasListEntry, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.aliases, f.err
}

func (f *Api) DescribeKey(ctx context.Context, keyId string) (*types.KeyMetadata, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.metadata[keyId], f.err
}

func (f *Api) ListTags(ctx context.Context, force bool, keyId string) ([]types.Tag, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.tags, f.err
}

func (f *Api) GetKeyPolicy(ctx context.Context, keyId string) (string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.policy, f.err
}
