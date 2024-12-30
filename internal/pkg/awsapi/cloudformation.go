package awsapi

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type CloudFormationApi struct {
	logger               *log.Logger
	config               aws.Config
	client               *cloudformation.Client
	allStacks            map[string]types.StackSummary
	stackEventsPaginator *cloudformation.DescribeStackEventsPaginator
}

func NewCloudFormationApi(
	config aws.Config,
	logger *log.Logger,
) *CloudFormationApi {
	return &CloudFormationApi{
		config:    config,
		logger:    logger,
		client:    cloudformation.NewFromConfig(config),
		allStacks: make(map[string]types.StackSummary),
	}
}

func (inst *CloudFormationApi) ListStacks(force bool) (map[string]types.StackSummary, error) {
	if !force && len(inst.allStacks) > 0 {
		return inst.allStacks, nil
	}

	inst.allStacks = make(map[string]types.StackSummary)

	var paginator = cloudformation.NewListStacksPaginator(
		inst.client, &cloudformation.ListStacksInput{},
	)

	var apiErr error = nil
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			apiErr = err
			break
		}
		for _, stack := range output.StackSummaries {
			inst.allStacks[*stack.StackName] = stack
		}
	}

	return inst.allStacks, apiErr
}

func (inst *CloudFormationApi) FilterByName(name string) map[string]types.StackSummary {

	if len(inst.allStacks) < 1 {
		inst.ListStacks(true)
	}

	var foundStacks = make(map[string]types.StackSummary)

	for _, info := range inst.allStacks {
		found := strings.Contains(*info.StackName, name)
		if found {
			foundStacks[*info.StackName] = info
		}
	}
	return foundStacks
}

func (inst *CloudFormationApi) DescribeStackEvents(stackName string, force bool) ([]types.StackEvent, error) {
	if inst.stackEventsPaginator == nil || force {
		inst.stackEventsPaginator = cloudformation.NewDescribeStackEventsPaginator(
			inst.client, &cloudformation.DescribeStackEventsInput{
				StackName: aws.String(stackName),
			},
		)
	}

	var empty []types.StackEvent
	if !inst.stackEventsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.stackEventsPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.StackEvents, nil
}
