package awsapi

import (
	"aws-tui/internal/pkg/ui/core"
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachineApi struct {
	logger                  *log.Logger
	config                  aws.Config
	client                  *sfn.Client
	allStateMachines        []types.StateMachineListItem
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
		allStateMachines: []types.StateMachineListItem{},
	}
}

func (inst *StateMachineApi) ListStateMachines(force bool) ([]types.StateMachineListItem, error) {
	if len(inst.allStateMachines) > 0 && !force {
		return inst.allStateMachines, nil
	}

	inst.allStateMachines = []types.StateMachineListItem{}

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

		inst.allStateMachines = append(inst.allStateMachines, output.StateMachines...)
	}

	sort.Slice(inst.allStateMachines, func(i, j int) bool {
		return aws.ToString(inst.allStateMachines[i].Name) < aws.ToString(inst.allStateMachines[j].Name)
	})

	return inst.allStateMachines, apiErr
}

func (inst *StateMachineApi) FilterByName(name string) []types.StateMachineListItem {
	return core.FuzzySearch(name, inst.allStateMachines, func(v types.StateMachineListItem) string {
		return aws.ToString(v.Name)
	})
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
