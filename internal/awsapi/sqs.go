package awsapi

import (
	"context"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SqsApi interface {
	ListQueues(ctx context.Context, force bool) ([]string, error)
	GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error)
	ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error)
	// SendMessage puts a message on the queue. The group and deduplication IDs
	// are optional, and are only sent when they are not empty.
	SendMessage(
		ctx context.Context, queueUrl string, body string, groupId string, deduplicationId string,
	) (*sqs.SendMessageOutput, error)
}

type sqsApi struct {
	baseApi
}

func NewSqsApi(
	logger *log.Logger,
) SqsApi {
	return &sqsApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *sqsApi) ListQueues(ctx context.Context, force bool) ([]string, error) {
	var paginator = sqs.NewListQueuesPaginator(
		GetAwsApiClients().sqs,
		&sqs.ListQueuesInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]string, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.QueueUrls, nil
	})

	sort.Strings(result)

	return result, inst.logErr(err)
}

func (inst *sqsApi) GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error) {
	if err := requireArg("queueUrl", queueUrl); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().sqs.GetQueueAttributes(ctx,
		&sqs.GetQueueAttributesInput{
			QueueUrl:       &queueUrl,
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameAll},
		},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output.Attributes, nil
}

func (inst *sqsApi) ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error) {
	if err := requireArg("queueUrl", queueUrl); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().sqs.ListQueueTags(ctx,
		&sqs.ListQueueTagsInput{QueueUrl: &queueUrl},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output.Tags, nil
}

func (inst *sqsApi) SendMessage(
	ctx context.Context, queueUrl string, body string, groupId string, deduplicationId string,
) (*sqs.SendMessageOutput, error) {
	if err := requireArg("queueUrl", queueUrl); err != nil {
		return nil, err
	}
	if err := requireArg("message body", body); err != nil {
		return nil, err
	}

	var input = &sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: &body,
	}
	if groupId != "" {
		input.MessageGroupId = &groupId
	}
	if deduplicationId != "" {
		input.MessageDeduplicationId = &deduplicationId
	}

	var output, err = GetAwsApiClients().sqs.SendMessage(ctx, input)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output, nil
}
