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
	ListTopicSubscriptions(ctx context.Context, topicArn string) ([]types.Subscription, error)
	// Publish sends a message to the topic. The subject, group and
	// deduplication IDs and message attributes are optional, and are only sent
	// when they are not empty.
	Publish(
		ctx context.Context, topicArn string, message string, subject string,
		groupId string, deduplicationId string, attributes map[string]types.MessageAttributeValue,
	) (*sns.PublishOutput, error)
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

func (inst *snsApi) Publish(
	ctx context.Context, topicArn string, message string, subject string,
	groupId string, deduplicationId string, attributes map[string]types.MessageAttributeValue,
) (*sns.PublishOutput, error) {
	if err := requireArg("topicArn", topicArn); err != nil {
		return nil, err
	}
	if err := requireArg("message", message); err != nil {
		return nil, err
	}

	var input = &sns.PublishInput{
		TopicArn: &topicArn,
		Message:  &message,
	}
	if subject != "" {
		input.Subject = &subject
	}
	if groupId != "" {
		input.MessageGroupId = &groupId
	}
	if deduplicationId != "" {
		input.MessageDeduplicationId = &deduplicationId
	}
	if len(attributes) > 0 {
		input.MessageAttributes = attributes
	}

	var output, err = GetAwsApiClients().sns.Publish(ctx, input)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output, nil
}

func (inst *snsApi) ListTopicSubscriptions(ctx context.Context, topicArn string) ([]types.Subscription, error) {
	if err := requireArg("topicArn", topicArn); err != nil {
		return nil, err
	}

	var paginator = sns.NewListSubscriptionsByTopicPaginator(
		GetAwsApiClients().sns,
		&sns.ListSubscriptionsByTopicInput{TopicArn: &topicArn},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.Subscription, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Subscriptions, nil
	})

	return result, inst.logErr(err)
}
