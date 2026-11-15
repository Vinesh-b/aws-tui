// Package lambdafake is an in-memory awsapi.LambdaApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package lambdafake

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type Api struct {
	awsapi.LambdaApi

	mtx       sync.Mutex
	functions []types.FunctionConfiguration
	err       error

	// mappings, versions, aliases, tags and policies are by lambda ARN, and
	// delays hold them back
	mappings  map[string][]types.EventSourceMappingConfiguration
	versions  map[string][]types.FunctionConfiguration
	aliases   map[string][]types.AliasConfiguration
	tags      map[string]map[string]string
	tagsErr   error
	policies  map[string]string
	policyErr error
	delays    map[string]time.Duration

	calls []string
}

type Option func(*Api)

// WithFunctions sets the lambdas ListLambdas gives
func WithFunctions(functions []types.FunctionConfiguration) Option {
	return func(f *Api) { f.functions = functions }
}

// WithErr makes ListLambdas, ListEventSourceMappings, ListVersions and
// ListAliases fail
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

// WithMappings sets the event source mappings of each lambda, by ARN
func WithMappings(mappings map[string][]types.EventSourceMappingConfiguration) Option {
	return func(f *Api) { f.mappings = maps.Clone(mappings) }
}

// WithVersions sets the versions of each lambda, by ARN
func WithVersions(versions map[string][]types.FunctionConfiguration) Option {
	return func(f *Api) { f.versions = maps.Clone(versions) }
}

// WithAliases sets the aliases of each lambda, by ARN
func WithAliases(aliases map[string][]types.AliasConfiguration) Option {
	return func(f *Api) { f.aliases = maps.Clone(aliases) }
}

// WithTags sets the tags of each lambda, by ARN
func WithTags(tags map[string]map[string]string) Option {
	return func(f *Api) { f.tags = maps.Clone(tags) }
}

// WithTagsErr makes ListTags fail
func WithTagsErr(err error) Option { return func(f *Api) { f.tagsErr = err } }

// WithPolicies sets the resource policy of each lambda, by ARN
func WithPolicies(policies map[string]string) Option {
	return func(f *Api) { f.policies = maps.Clone(policies) }
}

// WithPolicyErr makes GetPolicy fail
func WithPolicyErr(err error) Option { return func(f *Api) { f.policyErr = err } }

// WithDelays holds back every call for each lambda named by ARN, by the time
// given, as a slow api would. Use it in uitest.RunInBubble, where the wait
// takes no real time.
func WithDelays(delays map[string]time.Duration) Option {
	return func(f *Api) { f.delays = maps.Clone(delays) }
}

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call of ListLambdas, ListEventSourceMappings,
// ListVersions and ListAliases fail. Safe while background loads run.
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// Calls returns every call made so far, formatted as "<method> <lambda arn>",
// or just "ListLambdas".
func (f *Api) Calls() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

// CallsWithPrefix returns the calls whose text starts with prefix, e.g. "ListTags ".
func (f *Api) CallsWithPrefix(prefix string) []string {
	return slices.DeleteFunc(f.Calls(), func(call string) bool { return !strings.HasPrefix(call, prefix) })
}

// call notes the call for the lambda, and then holds it back by the delay of
// the lambda
func (f *Api) call(method string, lambdaArn string) {
	f.mtx.Lock()
	f.calls = append(f.calls, method+" "+lambdaArn)
	var delay = f.delays[lambdaArn]
	f.mtx.Unlock()
	time.Sleep(delay) // sleep-ok: simulated api latency
}

func (f *Api) ListLambdas(ctx context.Context, force bool) ([]types.FunctionConfiguration, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "ListLambdas")
	return slices.Clone(f.functions), f.err
}

func (f *Api) ListEventSourceMappings(
	ctx context.Context, lambdaArn string,
) ([]types.EventSourceMappingConfiguration, error) {
	f.call("ListEventSourceMappings", lambdaArn)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.mappings[lambdaArn]), f.err
}

func (f *Api) GetPolicy(ctx context.Context, lambdaArn string) (string, error) {
	f.call("GetPolicy", lambdaArn)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.policies[lambdaArn], f.policyErr
}

func (f *Api) ListTags(ctx context.Context, lambdaArn string) (map[string]string, error) {
	f.call("ListTags", lambdaArn)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return maps.Clone(f.tags[lambdaArn]), f.tagsErr
}

func (f *Api) ListVersions(ctx context.Context, lambdaArn string) ([]types.FunctionConfiguration, error) {
	f.call("ListVersions", lambdaArn)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.versions[lambdaArn]), f.err
}

func (f *Api) ListAliases(ctx context.Context, lambdaArn string) ([]types.AliasConfiguration, error) {
	f.call("ListAliases", lambdaArn)
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.aliases[lambdaArn]), f.err
}
