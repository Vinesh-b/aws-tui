package statemachine

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachineApi struct {
	logger             *log.Logger
	config             aws.Config
	client             *sfn.Client
	allStateMachines   map[string]types.StateMachineListItem
	nextExectionsToken *string
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

func (inst *StateMachineApi) ListStateMachines(force bool) map[string]types.StateMachineListItem {
	if len(inst.allStateMachines) > 0 && !force {
		return inst.allStateMachines
	}

	inst.allStateMachines = make(map[string]types.StateMachineListItem)

	var paginator = sfn.NewListStateMachinesPaginator(
		inst.client, &sfn.ListStateMachinesInput{},
	)

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		for _, val := range output.StateMachines {
			inst.allStateMachines[*val.Name] = val
		}
	}
	return inst.allStateMachines
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

func (inst *StateMachineApi) ListExecutions(name string, nextToken *string) ([]types.ExecutionListItem, *string) {
	var stateMachine = inst.allStateMachines[name]

	var paginator = sfn.NewListExecutionsPaginator(
		inst.client, &sfn.ListExecutionsInput{
			StateMachineArn: stateMachine.StateMachineArn,
			NextToken:       nextToken,
		},
	)

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		return output.Executions, output.NextToken
	}
	return nil, nil
}
