package sfn

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

var baseTime = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

func at(seconds int) *time.Time {
	return aws.Time(baseTime.Add(time.Duration(seconds) * time.Second))
}

func newStateMachine(name string, machineType types.StateMachineType) types.StateMachineListItem {
	return types.StateMachineListItem{
		Name:            aws.String(name),
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:" + name),
		Type:            machineType,
		CreationDate:    at(0),
	}
}

func newExecution(name string, status types.ExecutionStatus, startSec, stopSec int) ExecutionItem {
	return ExecutionItem{
		ExecutionListItem: &types.ExecutionListItem{
			Name:      aws.String(name),
			Status:    status,
			StartDate: at(startSec),
			StopDate:  at(stopSec),
		},
	}
}
