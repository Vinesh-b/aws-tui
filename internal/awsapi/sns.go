package awsapi

import (
	"context"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type SnsApi interface {
	ListTopics(ctx context.Context, force bool) ([]string, error)
	GetTopicAttributes(ctx context.Context, topicArn string) (map[string]string, error)
	ListTopicTags(ctx context.Context, topicArn string) ([]types.Tag, error)
}

type snsApi struct {
	baseApi
}

func NewSnsApi(
	logger *log.Logger,
) SnsApi {
	return &snsApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *snsApi) ListTopics(ctx context.Context, force bool) ([]string, error) {
	var paginator = sns.NewListTopicsPaginator(
		GetAwsApiClients().sns,
		&sns.ListTopicsInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]string, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var arns = make([]string, 0, len(output.Topics))
		for _, topic := range output.Topics {
			arns = append(arns, aws.ToString(topic.TopicArn))
		}
		return arns, nil
	})

	sort.Strings(result)

	return result, inst.logErr(err)
}

func (inst *snsApi) GetTopicAttributes(ctx context.Context, topicArn string) (map[string]string, error) {
	if err := requireArg("topicArn", topicArn); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().sns.GetTopicAttributes(ctx,
		&sns.GetTopicAttributesInput{TopicArn: &topicArn},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output.Attributes, nil
}

func (inst *snsApi) ListTopicTags(ctx context.Context, topicArn string) ([]types.Tag, error) {
	if err := requireArg("topicArn", topicArn); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().sns.ListTagsForResource(ctx,
		&sns.ListTagsForResourceInput{ResourceArn: &topicArn},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output.Tags, nil
}
