// Package snsfake is an in-memory awsapi.SnsApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package snsfake

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// PendingConfirmation is the ARN the api reports for an unconfirmed
// subscription. It must match the constant of the same name in the sns package.
const PendingConfirmation = "PendingConfirmation"

var ErrTopicArnNotSet = errors.New("topicArn not set")

func TopicArn(name string) string {
	return "arn:aws:sns:us-east-1:123456789012:" + name
}

type Api struct {
	awsapi.SnsApi

	mtx           sync.Mutex
	topics        []string
	attributes    map[string]string
	tags          []types.Tag
	subscriptions []types.Subscription
	subscribeArn  string
	err           error
	calls         []string
}

type Option func(*Api)

func WithTopics(topics []string) Option { return func(f *Api) { f.topics = topics } }
func WithAttributes(attributes map[string]string) Option {
	return func(f *Api) { f.attributes = attributes }
}
func WithTags(tags []types.Tag) Option { return func(f *Api) { f.tags = tags } }
func WithSubscriptions(subs []types.Subscription) Option {
	return func(f *Api) { f.subscriptions = subs }
}
func WithSubscribeArn(arn string) Option { return func(f *Api) { f.subscribeArn = arn } }
func WithErr(err error) Option           { return func(f *Api) { f.err = err } }

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

func (f *Api) SetAttributes(attributes map[string]string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.attributes = attributes
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

func (f *Api) record(call string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, call)
}

func (f *Api) getErr() error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.err
}

func (f *Api) ListTopics(ctx context.Context, force bool) ([]string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.topics), f.err
}

func (f *Api) GetTopicAttributes(ctx context.Context, arn string) (map[string]string, error) {
	f.record("attributes " + arn)
	if arn == "" {
		return nil, ErrTopicArnNotSet
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return maps.Clone(f.attributes), f.err
}

func (f *Api) ListTopicTags(ctx context.Context, arn string) ([]types.Tag, error) {
	f.record("tags " + arn)
	if arn == "" {
		return nil, ErrTopicArnNotSet
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.tags), f.err
}

func (f *Api) ListTopicSubscriptions(ctx context.Context, arn string) ([]types.Subscription, error) {
	f.record("subscriptions " + arn)
	if arn == "" {
		return nil, ErrTopicArnNotSet
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.subscriptions), f.err
}

func (f *Api) Subscribe(ctx context.Context, arn string, protocol string, endpoint string) (string, error) {
	f.record("subscribe " + arn + " " + protocol + " " + endpoint)
	if err := f.getErr(); err != nil {
		return "", err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var subscriptionArn = f.subscribeArn
	f.subscriptions = append(f.subscriptions, types.Subscription{
		Protocol:        &protocol,
		Endpoint:        &endpoint,
		SubscriptionArn: &subscriptionArn,
	})
	return subscriptionArn, nil
}

func (f *Api) Unsubscribe(ctx context.Context, arn string) error {
	f.record("unsubscribe " + arn)
	if arn == PendingConfirmation {
		return errors.New("a subscription that is pending confirmation can't be deleted")
	}
	if err := f.getErr(); err != nil {
		return err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.subscriptions = slices.DeleteFunc(f.subscriptions, func(s types.Subscription) bool {
		return aws.ToString(s.SubscriptionArn) == arn
	})
	return nil
}

func (f *Api) DeleteTopic(ctx context.Context, arn string) error {
	f.record("delete " + arn)
	if err := f.getErr(); err != nil {
		return err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.topics = slices.DeleteFunc(f.topics, func(topic string) bool { return topic == arn })
	return nil
}

func (f *Api) CreateTopic(ctx context.Context, name string, fifo bool, dedup bool) (string, error) {
	f.record(fmt.Sprintf("create %s fifo=%t dedup=%t", name, fifo, dedup))
	if err := f.getErr(); err != nil {
		return "", err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.topics = append(f.topics, TopicArn(name))
	return TopicArn(name), nil
}
