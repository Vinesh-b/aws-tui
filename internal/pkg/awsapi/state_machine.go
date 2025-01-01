package awsapi

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachineApi struct {
	logger                  *log.Logger
	config                  aws.Config
	client                  *sfn.Client
	allStateMachines        map[string]types.StateMachineListItem
	nextExectionsToken      *string
	listExecutionsPaginator *sfn.ListExecutionsPaginator
}

func NewStateMachineApi(
	config aws.Config,
	logger *log.Logger,
) *StateMachineApi {
	return &StateMachineApi{
		config:           config,
		logger:           logger,
		client:           sfn.NewFromConfig(config),
		allStateMachines: make(map[string]types.StateMachineListItem),
	}
}

func (inst *StateMachineApi) ListStateMachines(force bool) (map[string]types.StateMachineListItem, error) {
	if len(inst.allStateMachines) > 0 && !force {
		return inst.allStateMachines, nil
	}

	inst.allStateMachines = make(map[string]types.StateMachineListItem)

	var paginator = sfn.NewListStateMachinesPaginator(
		inst.client, &sfn.ListStateMachinesInput{},
	)

	var apiErr error = nil
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			apiErr = err
			break
		}

		for _, val := range output.StateMachines {
			inst.allStateMachines[*val.Name] = val
		}
	}
	return inst.allStateMachines, apiErr
}

func (inst *StateMachineApi) FilterByName(name string) map[string]types.StateMachineListItem {

	if len(inst.allStateMachines) < 1 {
		inst.ListStateMachines(true)
	}

	var foundLambdas = make(map[string]types.StateMachineListItem)

	for _, info := range inst.allStateMachines {
		found := strings.Contains(*info.Name, name)
		if found {
			foundLambdas[*info.Name] = info
		}
	}
	return foundLambdas
}

func (inst *StateMachineApi) ListExecutions(stateMachineArn string, reset bool) ([]types.ExecutionListItem, error) {
	var empty = []types.ExecutionListItem{}

	if len(stateMachineArn) == 0 {
		return empty, fmt.Errorf("State machine ARN not set")
	}

	if inst.listExecutionsPaginator == nil || reset == true {
		inst.listExecutionsPaginator = sfn.NewListExecutionsPaginator(
			inst.client, &sfn.ListExecutionsInput{
				StateMachineArn: aws.String(stateMachineArn),
				MaxResults:      100,
			},
		)
	}

	if !inst.listExecutionsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.listExecutionsPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.Executions, nil
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
