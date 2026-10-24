package awsapi

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachineApi struct {
	baseApi
	nextExectionsToken      *string
	listExecutionsPaginator *sfn.ListExecutionsPaginator
}

func NewStateMachineApi(
	logger *log.Logger,
) *StateMachineApi {
	return &StateMachineApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *StateMachineApi) ListStateMachines(force bool) ([]types.StateMachineListItem, error) {
	var paginator = sfn.NewListStateMachinesPaginator(
		GetAwsApiClients().sfn, &sfn.ListStateMachinesInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.StateMachineListItem, error) {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		return output.StateMachines, nil
	})

	sortByKey(result, func(m types.StateMachineListItem) *string { return m.Name })

	return result, inst.logErr(err)
}

func (inst *StateMachineApi) DescribeStateMachine(stateMachineArn string) (*sfn.DescribeStateMachineOutput, error) {
	if err := requireArg("state machine ARN", stateMachineArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().sfn

	var output, err = client.DescribeStateMachine(
		context.TODO(),
		&sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(stateMachineArn),
		},
	)

	if err != nil {
		return nil, err
	}

	return output, err
}

func (inst *StateMachineApi) ListExecutions(
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
		var output, err = inst.listExecutionsPaginator.NextPage(context.TODO())
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

func (inst *StateMachineApi) DescribeExecution(executionArn string) (*sfn.DescribeExecutionOutput, error) {
	if err := requireArg("execution ARN", executionArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().sfn

	var response, err = client.DescribeExecution(context.TODO(), &sfn.DescribeExecutionInput{
		ExecutionArn: &executionArn,
	})

	if err != nil {
		return nil, inst.logErr(err)
	}

	return response, nil
}

func (inst *StateMachineApi) GetExecutionHistory(executionArn string) (*sfn.GetExecutionHistoryOutput, error) {
	if err := requireArg("execution ARN", executionArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().sfn
	var response, err = client.GetExecutionHistory(context.TODO(), &sfn.GetExecutionHistoryInput{
		ExecutionArn:         aws.String(executionArn),
		IncludeExecutionData: aws.Bool(true),
	})

	if err != nil {
		return nil, inst.logErr(err)
	}

	return response, nil
}
