package awsapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachineApi struct {
	logger                  *log.Logger
	config                  aws.Config
	client                  *sfn.Client
	nextExectionsToken      *string
	listExecutionsPaginator *sfn.ListExecutionsPaginator
}

func NewStateMachineApi(
	config aws.Config,
	logger *log.Logger,
) *StateMachineApi {
	return &StateMachineApi{
		config: config,
		logger: logger,
		client: sfn.NewFromConfig(config),
	}
}

func (inst *StateMachineApi) ListStateMachines(force bool) ([]types.StateMachineListItem, error) {
	var paginator = sfn.NewListStateMachinesPaginator(
		inst.client, &sfn.ListStateMachinesInput{},
	)

	var apiErr error = nil
	var result = []types.StateMachineListItem{}

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			apiErr = err
			break
		}

		result = append(result, output.StateMachines...)
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].Name) < aws.ToString(result[j].Name)
	})

	return result, apiErr
}

func (inst *StateMachineApi) ListExecutions(
	stateMachineArn string, start time.Time, end time.Time, reset bool,
) ([]types.ExecutionListItem, error) {
	var empty = []types.ExecutionListItem{}

	if len(stateMachineArn) == 0 {
		return empty, fmt.Errorf("State machine ARN not set")
	}

	if inst.listExecutionsPaginator == nil || reset == true {
		inst.listExecutionsPaginator = sfn.NewListExecutionsPaginator(
			inst.client, &sfn.ListExecutionsInput{
				StateMachineArn: aws.String(stateMachineArn),
				MaxResults:      500,
			},
		)
	}

	var result = []types.ExecutionListItem{}
	for inst.listExecutionsPaginator.HasMorePages() {
		var output, err = inst.listExecutionsPaginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			return empty, err
		}

		for _, exec := range output.Executions {
			var date = exec.StartDate

			if date.After(end) {
				break
			}

			if date != nil && (date.Equal(start) || date.After(start)) && date.Before(end) {
				result = append(result, exec)
			}
		}
	}

	return result, nil
}

func (inst *StateMachineApi) DescribeExecution(executionArn string) (*sfn.DescribeExecutionOutput, error) {
	if len(executionArn) == 0 {
		return nil, fmt.Errorf("Exeuction ARN not set")
	}

	var response, err = inst.client.DescribeExecution(context.TODO(), &sfn.DescribeExecutionInput{
		ExecutionArn: &executionArn,
	})

	if err != nil {
		inst.logger.Println(err)
		return nil, err
	}

	return response, nil
}

func (inst *StateMachineApi) GetExecutionHistory(executionArn string) (*sfn.GetExecutionHistoryOutput, error) {
	if len(executionArn) == 0 {
		return nil, fmt.Errorf("Exeuction ARN not set")
	}

	var response, err = inst.client.GetExecutionHistory(context.TODO(), &sfn.GetExecutionHistoryInput{
		ExecutionArn:         aws.String(executionArn),
		IncludeExecutionData: aws.Bool(true),
	})

	if err != nil {
		inst.logger.Println(err)
		return nil, err
	}

	return response, nil
}
