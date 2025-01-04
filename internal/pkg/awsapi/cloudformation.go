package awsapi

import (
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
	stackEventsPaginator *cloudformation.DescribeStackEventsPaginator
}

func NewCloudFormationApi(
	config aws.Config,
	logger *log.Logger,
) *CloudFormationApi {
	return &CloudFormationApi{
		config: config,
		logger: logger,
		client: cloudformation.NewFromConfig(config),
	}
}

func (inst *CloudFormationApi) ListStacks(force bool) ([]types.StackSummary, error) {
	var paginator = cloudformation.NewListStacksPaginator(
		inst.client, &cloudformation.ListStacksInput{},
	)

	var apiErr error = nil
	var result = []types.StackSummary{}

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			apiErr = err
			break
		}
		result = append(result, output.StackSummaries...)
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].StackName) < aws.ToString(result[j].StackName)
	})

	return result, apiErr
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
