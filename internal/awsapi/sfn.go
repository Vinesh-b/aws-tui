package awsapi

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachineApi interface {
	ListStateMachines(ctx context.Context, force bool) ([]types.StateMachineListItem, error)
	DescribeStateMachine(ctx context.Context, stateMachineArn string) (*sfn.DescribeStateMachineOutput, error)
	ListExecutions(
		ctx context.Context,
		stateMachineArn string, start time.Time, end time.Time, reset bool,
	) ([]types.ExecutionListItem, error)
	DescribeExecution(ctx context.Context, executionArn string) (*sfn.DescribeExecutionOutput, error)
	GetExecutionHistory(ctx context.Context, executionArn string) (*sfn.GetExecutionHistoryOutput, error)
}

type stateMachineApi struct {
	baseApi
	nextExecutionsToken     *string
	listExecutionsPaginator *sfn.ListExecutionsPaginator
}

func NewStateMachineApi(
	logger *log.Logger,
) StateMachineApi {
	return &stateMachineApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *stateMachineApi) ListStateMachines(ctx context.Context, force bool) ([]types.StateMachineListItem, error) {
	var paginator = sfn.NewListStateMachinesPaginator(
		GetAwsApiClients().sfn, &sfn.ListStateMachinesInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.StateMachineListItem, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.StateMachines, nil
	})

	sortByKey(result, func(m types.StateMachineListItem) *string { return m.Name })

	return result, inst.logErr(err)
}

func (inst *stateMachineApi) DescribeStateMachine(ctx context.Context, stateMachineArn string) (*sfn.DescribeStateMachineOutput, error) {
	if err := requireArg("state machine ARN", stateMachineArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().sfn

	var output, err = client.DescribeStateMachine(
		ctx,
		&sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(stateMachineArn),
		},
	)

	if err != nil {
		return nil, err
	}

	return output, err
}

func (inst *stateMachineApi) ListExecutions(
	ctx context.Context,
	stateMachineArn string, start time.Time, end time.Time, reset bool,
) ([]types.ExecutionListItem, error) {
	var empty = []types.ExecutionListItem{}

	if err := requireArg("state machine ARN", stateMachineArn); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().sfn

	if inst.listExecutionsPaginator == nil || reset == true {
		inst.listExecutionsPaginator = sfn.NewListExecutionsPaginator(
			client, &sfn.ListExecutionsInput{
				StateMachineArn: aws.String(stateMachineArn),
				MaxResults:      500,
			},
		)
	}

	var result = []types.ExecutionListItem{}
	for inst.listExecutionsPaginator.HasMorePages() {
		var output, err = inst.listExecutionsPaginator.NextPage(ctx)
		if err != nil {
			return empty, inst.logErr(err)
		}

		for _, exec := range output.Executions {
			var date = exec.StartDate

			if date == nil || date.After(end) {
				continue
			}

			if date.Before(start) {
				return result, nil
			}

			result = append(result, exec)
		}
	}

	return result, nil
}

func (inst *stateMachineApi) DescribeExecution(ctx context.Context, executionArn string) (*sfn.DescribeExecutionOutput, error) {
	if err := requireArg("execution ARN", executionArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().sfn

	var response, err = client.DescribeExecution(ctx, &sfn.DescribeExecutionInput{
		ExecutionArn: &executionArn,
	})

	if err != nil {
		return nil, inst.logErr(err)
	}

	return response, nil
}

func (inst *stateMachineApi) GetExecutionHistory(ctx context.Context, executionArn string) (*sfn.GetExecutionHistoryOutput, error) {
	if err := requireArg("execution ARN", executionArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().sfn
	var response, err = client.GetExecutionHistory(ctx, &sfn.GetExecutionHistoryInput{
		ExecutionArn:         aws.String(executionArn),
		IncludeExecutionData: aws.Bool(true),
	})

	if err != nil {
		return nil, inst.logErr(err)
	}

	return response, nil
}
