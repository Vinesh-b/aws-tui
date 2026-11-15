// Package sqsfake is an in-memory awsapi.SqsApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package sqsfake

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// ErrQueueUrlNotSet is how the fake fails a call for a queue without its URL,
// as the real api does
var ErrQueueUrlNotSet = errors.New("queueUrl not set")

// SentMessageId is the id SendMessage gives the messages it sends
const SentMessageId = "m-1"

// Sent is a message that SendMessage was asked to send
type Sent struct {
	QueueUrl        string
	Body            string
	GroupId         string
	DeduplicationId string
	Attributes      map[string]types.MessageAttributeValue
}

type Api struct {
	awsapi.SqsApi

	mtx        sync.Mutex
	queues     []string
	attributes map[string]string
	tags       map[string]string
	err        error
	sendErr    error
	sent       []Sent
	calls      []string
}

type Option func(*Api)

// WithQueues sets the queue URLs ListQueues gives
func WithQueues(queues []string) Option { return func(f *Api) { f.queues = queues } }

// WithAttributes sets the attributes GetQueueAttributes gives, for any queue
func WithAttributes(attributes map[string]string) Option {
	return func(f *Api) { f.attributes = attributes }
}

// WithTags sets the tags ListQueueTags gives, for any queue
func WithTags(tags map[string]string) Option { return func(f *Api) { f.tags = tags } }

// WithErr makes every call but SendMessage fail
func WithErr(err error) Option { return func(f *Api) { f.err = err } }

// WithSendErr makes SendMessage fail
func WithSendErr(err error) Option { return func(f *Api) { f.sendErr = err } }

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call but SendMessage fail. Safe while
// background loads run.
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// Calls returns every call made so far, formatted as "<method> <queue url>".
func (f *Api) Calls() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

// Sent gives the messages SendMessage was asked to send, oldest first
func (f *Api) Sent() []Sent {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var sent = slices.Clone(f.sent)
	for i := range sent {
		sent[i].Attributes = maps.Clone(sent[i].Attributes)
	}
	return sent
}

// record notes the call, and gives the error it fails with
func (f *Api) record(call string, queueUrl string) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, call+" "+queueUrl)
	if queueUrl == "" {
		return ErrQueueUrlNotSet
	}
	return f.err
}

func (f *Api) ListQueues(ctx context.Context, force bool) ([]string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, "queues")
	return slices.Clone(f.queues), f.err
}

func (f *Api) GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error) {
	if err := f.record("attributes", queueUrl); err != nil {
		return nil, err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return maps.Clone(f.attributes), nil
}

func (f *Api) ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error) {
	if err := f.record("tags", queueUrl); err != nil {
		return nil, err
	}
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return maps.Clone(f.tags), nil
}

func (f *Api) SendMessage(
	ctx context.Context, queueUrl string, body string, groupId string, deduplicationId string,
	attributes map[string]types.MessageAttributeValue,
) (*sqs.SendMessageOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.sent = append(f.sent, Sent{
		QueueUrl: queueUrl, Body: body, GroupId: groupId, DeduplicationId: deduplicationId,
		Attributes: maps.Clone(attributes),
	})
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return &sqs.SendMessageOutput{
		MessageId:      aws.String(SentMessageId),
		SequenceNumber: aws.String("42"),
	}, nil
}
