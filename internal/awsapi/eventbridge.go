package awsapi

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBridgeApi interface {
	ListEventBuses(ctx context.Context, force bool) ([]types.EventBus, error)
	DescribeEventBus(ctx context.Context, force bool, busArn string) (eventbridge.DescribeEventBusOutput, error)
	ListRules(ctx context.Context, force bool, busArn string) ([]types.Rule, error)
	ListTargetsByRule(ctx context.Context, force bool, busArn string, ruleName string) ([]types.Target, error)
	ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error)
	// PutEvent sends a single event to the bus, given by name or ARN, and
	// returns its event ID. The detail is a JSON object, and the resources are
	// optional ARNs the event is about.
	PutEvent(
		ctx context.Context, busName string, source string, detailType string, detail string, resources []string,
	) (string, error)
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

func (inst *eventBridgeApi) ListTargetsByRule(
	ctx context.Context, force bool, busArn string, ruleName string,
) ([]types.Target, error) {
	if err := requireArg("ruleName", ruleName); err != nil {
		return nil, err
	}

	var result, err = collectTokenPages(func(token *string) ([]types.Target, *string, error) {
		var output, err = GetAwsApiClients().eventbridge.ListTargetsByRule(ctx,
			&eventbridge.ListTargetsByRuleInput{
				Rule:         &ruleName,
				EventBusName: &busArn,
				Limit:        aws.Int32(100),
				NextToken:    token,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Targets, output.NextToken, nil
	})

	return result, inst.logErr(err)
}

func (inst *eventBridgeApi) PutEvent(
	ctx context.Context, busName string, source string, detailType string, detail string, resources []string,
) (string, error) {
	if err := requireArg("event bus", busName); err != nil {
		return "", err
	}
	if err := requireArg("event source", source); err != nil {
		return "", err
	}
	if err := requireArg("event detail type", detailType); err != nil {
		return "", err
	}
	if err := requireArg("event detail", detail); err != nil {
		return "", err
	}

	var output, err = GetAwsApiClients().eventbridge.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{{
			EventBusName: aws.String(busName),
			Source:       aws.String(source),
			DetailType:   aws.String(detailType),
			Detail:       aws.String(detail),
			Resources:    resources,
		}},
	})
	if err != nil {
		return "", inst.logErr(err)
	}

	// The call succeeds even when an entry is rejected, so check the entry
	if len(output.Entries) == 0 || output.FailedEntryCount > 0 {
		var failure = fmt.Errorf("event was not accepted")
		if len(output.Entries) > 0 {
			var entry = output.Entries[0]
			failure = fmt.Errorf("event was not accepted: %s: %s",
				aws.ToString(entry.ErrorCode), aws.ToString(entry.ErrorMessage))
		}
		return "", inst.logErr(failure)
	}

	return aws.ToString(output.Entries[0].EventId), nil
}
