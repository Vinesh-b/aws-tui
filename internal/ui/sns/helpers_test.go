package sns

import (
	"context"
	"errors"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

var errTopicArnNotSet = errors.New("topicArn not set")

var topicHeadings = []string{"Name", "Type"}

func topicArn(name string) string {
	return "arn:aws:sns:us-east-1:123456789012:" + name
}

func newTopicAttributes() map[string]string {
	return map[string]string{
		"DisplayName":             "Orders",
		"Owner":                   "123456789012",
		"SubscriptionsConfirmed":  "3",
		"SubscriptionsPending":    "1",
		"SubscriptionsDeleted":    "0",
		"KmsMasterKeyId":          "alias/aws/sns",
		"EffectiveDeliveryPolicy": `{"http":{}}`,
	}
}

type fakeSnsApi struct {
	awsapi.SnsApi
	topics        []string
	attributes    map[string]string
	tags          []types.Tag
	subscriptions []types.Subscription
	subscribeArn  string
	err           error

	mtx   sync.Mutex
	calls []string
}

func (f *fakeSnsApi) record(call string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, call)
}

func (f *fakeSnsApi) ListTopics(ctx context.Context, force bool) ([]string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.topics), f.err
}

func (f *fakeSnsApi) GetTopicAttributes(ctx context.Context, arn string) (map[string]string, error) {
	f.record("attributes " + arn)
	if arn == "" {
		return nil, errTopicArnNotSet
	}
	return f.attributes, f.err
}

func (f *fakeSnsApi) ListTopicTags(ctx context.Context, arn string) ([]types.Tag, error) {
	f.record("tags " + arn)
	if arn == "" {
		return nil, errTopicArnNotSet
	}
	return f.tags, f.err
}

func (f *fakeSnsApi) ListTopicSubscriptions(ctx context.Context, arn string) ([]types.Subscription, error) {
	f.record("subscriptions " + arn)
	if arn == "" {
		return nil, errTopicArnNotSet
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return append([]types.Subscription(nil), f.subscriptions...), f.err
}

func (f *fakeSnsApi) Subscribe(ctx context.Context, arn string, protocol string, endpoint string) (string, error) {
	f.record("subscribe " + arn + " " + protocol + " " + endpoint)
	if f.err != nil {
		return "", f.err
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

func (f *fakeSnsApi) Unsubscribe(ctx context.Context, arn string) error {
	f.record("unsubscribe " + arn)
	if arn == pendingConfirmation {
		return errors.New("a subscription that is pending confirmation can't be deleted")
	}
	if f.err != nil {
		return f.err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.subscriptions = slices.DeleteFunc(f.subscriptions, func(s types.Subscription) bool {
		return aws.ToString(s.SubscriptionArn) == arn
	})
	return nil
}

func (f *fakeSnsApi) DeleteTopic(ctx context.Context, arn string) error {
	f.record("delete " + arn)
	if f.err != nil {
		return f.err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.topics = slices.DeleteFunc(f.topics, func(topic string) bool { return topic == arn })
	return nil
}

func (f *fakeSnsApi) recorded() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return append([]string(nil), f.calls...)
}
