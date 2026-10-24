package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBridgeApi interface {
	ListEventBuses(ctx context.Context, force bool) ([]types.EventBus, error)
	DescribeEventBus(ctx context.Context, force bool, busArn string) (eventbridge.DescribeEventBusOutput, error)
	ListRules(ctx context.Context, force bool, busArn string) ([]types.Rule, error)
	ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error)
}

type eventBridgeApi struct {
	baseApi
	allEventBuses []types.EventBus
	allBusRules   []types.Rule
}

func NewEventBridgeApi(
	logger *log.Logger,
) EventBridgeApi {
	return &eventBridgeApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *eventBridgeApi) ListEventBuses(ctx context.Context, force bool) ([]types.EventBus, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.EventBus, *string, error) {
		var output, err = GetAwsApiClients().eventbridge.ListEventBuses(ctx,
			&eventbridge.ListEventBusesInput{
				Limit:     aws.Int32(50),
				NextToken: token,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.EventBuses, output.NextToken, nil
	})

	sortByKey(result, func(b types.EventBus) *string { return b.Name })

	inst.allEventBuses = result

	return result, inst.logErr(err)
}

func (inst *eventBridgeApi) DescribeEventBus(ctx context.Context, force bool, busArn string) (eventbridge.DescribeEventBusOutput, error) {
	var empty = eventbridge.DescribeEventBusOutput{}
	var client = GetAwsApiClients().eventbridge

	var output, err = client.DescribeEventBus(ctx,
		&eventbridge.DescribeEventBusInput{
			Name: aws.String(busArn),
		},
	)

	if err != nil {
		return empty, inst.logErr(err)
	}

	return *output, err
}

func (inst *eventBridgeApi) ListRules(ctx context.Context, force bool, busArn string) ([]types.Rule, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.Rule, *string, error) {
		var output, err = GetAwsApiClients().eventbridge.ListRules(ctx,
			&eventbridge.ListRulesInput{
				EventBusName: &busArn,
				Limit:        aws.Int32(50),
				NextToken:    token,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Rules, output.NextToken, nil
	})

	sortByKey(result, func(r types.Rule) *string { return r.Name })

	inst.allBusRules = result

	return result, inst.logErr(err)
}

func (inst *eventBridgeApi) ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error) {
	var client = GetAwsApiClients().eventbridge

	var output, err = client.ListTagsForResource(ctx,
		&eventbridge.ListTagsForResourceInput{
			ResourceARN: aws.String(resourceArn),
		},
	)

	if err != nil {
		return nil, inst.logErr(err)
	}

	return output.Tags, nil
}
