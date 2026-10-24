package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type CloudFormationApi struct {
	baseApi
	stackEventsPaginator *cloudformation.DescribeStackEventsPaginator
}

func NewCloudFormationApi(
	logger *log.Logger,
) *CloudFormationApi {
	return &CloudFormationApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *CloudFormationApi) ListStacks(force bool) ([]types.StackSummary, error) {
	var paginator = cloudformation.NewListStacksPaginator(
		GetAwsApiClients().cloudformation, &cloudformation.ListStacksInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.StackSummary, error) {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		return output.StackSummaries, nil
	})

	sortByKey(result, func(s types.StackSummary) *string { return s.StackName })

	return result, inst.logErr(err)
}

func (inst *CloudFormationApi) DescribeStackEvents(stackName string, force bool) ([]types.StackEvent, error) {
	var empty []types.StackEvent

	if err := requireArg("stack name", stackName); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().cloudformation

	if inst.stackEventsPaginator == nil || force {
		inst.stackEventsPaginator = cloudformation.NewDescribeStackEventsPaginator(
			client, &cloudformation.DescribeStackEventsInput{
				StackName: aws.String(stackName),
			},
		)
	}

	if !inst.stackEventsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.stackEventsPaginator.NextPage(context.TODO())
	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.StackEvents, nil
}
