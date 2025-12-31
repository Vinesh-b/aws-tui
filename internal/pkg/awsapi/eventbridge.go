package awsapi

import (
	"context"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBridgeApi struct {
	logger        *log.Logger
	config        aws.Config
	client        *eventbridge.Client
	allEventBuses []types.EventBus
	allBusRules   []types.Rule
}

func NewEventBridgeApi(
	config aws.Config,
	logger *log.Logger,
) *EventBridgeApi {
	return &EventBridgeApi{
		config: config,
		logger: logger,
		client: eventbridge.NewFromConfig(config),
	}
}

func (inst *EventBridgeApi) ListEventBuses(force bool) ([]types.EventBus, error) {
	var nextToken *string = nil
	var namePrefix *string = nil
	var apiError error = nil
	var result = []types.EventBus{}

	for {
		var output, err = inst.client.ListEventBuses(context.TODO(),
			&eventbridge.ListEventBusesInput{
				Limit:      aws.Int32(50),
				NamePrefix: namePrefix,
				NextToken:  nextToken,
			},
		)

		if err != nil {
			apiError = err
			break
		}

		result = append(result, output.EventBuses...)

		if output.NextToken == nil {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].Name) < aws.ToString(result[j].Name)
	})

	inst.allEventBuses = result

	return result, apiError
}

func (inst *EventBridgeApi) DescribeEventBus(force bool, busArn string) (eventbridge.DescribeEventBusOutput, error) {
	var empty = eventbridge.DescribeEventBusOutput{}

	var output, err = inst.client.DescribeEventBus(context.TODO(),
		&eventbridge.DescribeEventBusInput{
			Name: aws.String(busArn),
		},
	)

	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return *output, err
}

func (inst *EventBridgeApi) ListRules(force bool, busArn string) ([]types.Rule, error) {
	var nextToken *string = nil
	var namePrefix *string = nil
	var apiError error = nil
	var result = []types.Rule{}

	for {
		var output, err = inst.client.ListRules(context.TODO(),
			&eventbridge.ListRulesInput{EventBusName: &busArn,
				Limit:      aws.Int32(50),
				NamePrefix: namePrefix,
				NextToken:  nextToken,
			},
		)

		if err != nil {
			apiError = err
			inst.logger.Println(err)
			break
		}

		result = append(result, output.Rules...)

		if output.NextToken == nil {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].Name) < aws.ToString(result[j].Name)
	})

	inst.allBusRules = result

	return result, apiError
}

func (inst *EventBridgeApi) ListTags(force bool, resourceArn string) ([]types.Tag, error) {
	var apiError error = nil

	var output, err = inst.client.ListTagsForResource(context.TODO(),
		&eventbridge.ListTagsForResourceInput{
			ResourceARN: aws.String(resourceArn),
		},
	)

	if err != nil {
		inst.logger.Println(err)
		return nil, err
	}

	return output.Tags, apiError
}
