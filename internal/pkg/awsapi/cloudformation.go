package awsapi

import (
	"aws-tui/internal/pkg/ui/core"
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type CloudFormationApi struct {
	logger               *log.Logger
	config               aws.Config
	client               *cloudformation.Client
	allStacks            []types.StackSummary
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
		allStacks: []types.StackSummary{},
	}
}

func (inst *CloudFormationApi) ListStacks(force bool) ([]types.StackSummary, error) {
	if !force && len(inst.allStacks) > 0 {
		return inst.allStacks, nil
	}

	inst.allStacks = []types.StackSummary{}

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
		inst.allStacks = append(inst.allStacks, output.StackSummaries...)
	}

	sort.Slice(inst.allStacks, func(i, j int) bool {
		return aws.ToString(inst.allStacks[i].StackName) < aws.ToString(inst.allStacks[j].StackName)
	})

	return inst.allStacks, apiErr
}

func (inst *CloudFormationApi) FilterByName(name string) []types.StackSummary {
	if len(inst.allStacks) == 0 {
        return nil
	}

	var foundIdxs = core.FuzzySearch(name, inst.allStacks, func(v types.StackSummary) string {
		return aws.ToString(v.StackName)
	})

	var foundStacks = []types.StackSummary{}

	for _, matchIdx := range foundIdxs {
		var stack = inst.allStacks[matchIdx]
		foundStacks = append(foundStacks, stack)
	}
	return foundStacks
}

func (inst *CloudFormationApi) DescribeStackEvents(stackName string, force bool) ([]types.StackEvent, error) {
	var empty []types.StackEvent

	if len(stackName) == 0 {
		return empty, fmt.Errorf("Stack name not set")
	}

	if inst.stackEventsPaginator == nil || force {
		inst.stackEventsPaginator = cloudformation.NewDescribeStackEventsPaginator(
			inst.client, &cloudformation.DescribeStackEventsInput{
				StackName: aws.String(stackName),
			},
		)
	}

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
