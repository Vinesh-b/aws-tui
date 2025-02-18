package servicetables

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type SfnDetailsTable struct {
	*core.DetailsTable
	data                    *sfn.DescribeStateMachineOutput
	selectedStateMachineArn string
	logGroups               []string
	logGroupsChan           chan []string
	serviceCtx              *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnDetailsTable(
	serviceContext *core.ServiceContext[awsapi.StateMachineApi],
) *SfnDetailsTable {
	var table = &SfnDetailsTable{
		DetailsTable:            core.NewDetailsTable("State Machine Details", serviceContext.AppContext),
		data:                    nil,
		logGroups:               []string{},
		logGroupsChan:           make(chan []string),
		selectedStateMachineArn: "",
		serviceCtx:              serviceContext,
	}

	table.populateTable()

	return table
}

func (inst *SfnDetailsTable) populateTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = []core.TableRow{
			{"Name", aws.ToString(inst.data.Name)},
			{"ARN", aws.ToString(inst.data.StateMachineArn)},
			{"Type", string(inst.data.Type)},
			{"Description", aws.ToString(inst.data.Description)},
			{"Status", string(inst.data.Status)},
			{"Created Date", aws.ToTime(inst.data.CreationDate).Format(time.DateTime)},
		}

		var logConfig = inst.data.LoggingConfiguration
		if logConfig != nil {
			var logData = []core.TableRow{
				{"Logging Config", ""},
				{"Include Execution Data", fmt.Sprintf("%v", logConfig.IncludeExecutionData)},
				{"Log Level", string(logConfig.Level)},
			}

			logData = append(logData, core.TableRow{"Log Groups", ""})
			for _, logDest := range logConfig.Destinations {
				var group = logDest.CloudWatchLogsLogGroup
				if group != nil {

					var splitArn = strings.Split(aws.ToString(group.LogGroupArn), ":")
					logData = append(logData, core.TableRow{"", splitArn[6]})
					inst.logGroups = append(inst.logGroups, splitArn[6])
				}
			}

			tableData = append(tableData, logData...)
		}
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *SfnDetailsTable) ClearDetails() {
	inst.data = nil
	inst.logGroups = nil
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable()
	})
}

func (inst *SfnDetailsTable) RefreshDetails(stateMachine types.StateMachineListItem) {
ChanFlushLoop:
	for {
		select {
		case _, ok := <-inst.logGroupsChan:
			if !ok {
				break ChanFlushLoop
			}
		default:
			break ChanFlushLoop
		}
	}

	inst.selectedStateMachineArn = aws.ToString(stateMachine.StateMachineArn)
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.data, err = inst.serviceCtx.Api.DescribeStateMachine(inst.selectedStateMachineArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
			return
		}

		if inst.data == nil {
			return
		}

		var logConfig = inst.data.LoggingConfiguration
		inst.logGroups = nil
		if logConfig != nil {
			for _, logDest := range logConfig.Destinations {
				var group = logDest.CloudWatchLogsLogGroup
				if group != nil {
					var splitArn = strings.Split(aws.ToString(group.LogGroupArn), ":")
					inst.logGroups = append(inst.logGroups, splitArn[6])
					go func() {
						inst.logGroupsChan <- inst.logGroups
					}()
				}
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable()
	})
}

func (inst *SfnDetailsTable) GetSelectedSmLogGroup() string {
	var logGroups []string
	var timeoutCtx, cancelFunc = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	for {
		select {
		case logGroups = <-inst.logGroupsChan:
			if len(logGroups) > 0 {
				return logGroups[0]
			}
			return ""
		case <-timeoutCtx.Done():
			inst.ErrorMessageCallback("Timed out requesting state machine log groups")
			return ""
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}
