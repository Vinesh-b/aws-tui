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
