package sfn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

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
	onDetailsLoaded         []func(*sfn.DescribeStateMachineOutput)
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

// AddDetailsLoadedFunc adds a function called with the described state machine
// each time the details are shown, which is nil once they are cleared.
func (inst *SfnDetailsTable) AddDetailsLoadedFunc(handler func(*sfn.DescribeStateMachineOutput)) {
	inst.onDetailsLoaded = append(inst.onDetailsLoaded, handler)
}

func (inst *SfnDetailsTable) populateTable() {
	for _, handler := range inst.onDetailsLoaded {
		handler(inst.data)
	}

	var tableData []core.TableRow
	if inst.data != nil {
		tableData = []core.TableRow{
			{"Name", aws.ToString(inst.data.Name)},
			{"ARN", aws.ToString(inst.data.StateMachineArn)},
			{"Type", string(inst.data.Type)},
			{"Description", aws.ToString(inst.data.Description)},
			{"Status", string(inst.data.Status)},
			{"Created", core.FormatTime(inst.data.CreationDate)},
		}

		var logConfig = inst.data.LoggingConfiguration
		if logConfig != nil {
			var logData = []core.TableRow{
				{"Include Execution Data", fmt.Sprintf("%v", logConfig.IncludeExecutionData)},
				{"Log Level", string(logConfig.Level)},
			}

			for _, logDest := range logConfig.Destinations {
				var group = logDest.CloudWatchLogsLogGroup
				if group != nil {

					var splitArn = strings.Split(aws.ToString(group.LogGroupArn), ":")
					logData = append(logData, core.TableRow{"Log Group", splitArn[6]})
					inst.logGroups = append(inst.logGroups, splitArn[6])
				}
			}

			tableData = append(tableData, core.DetailsSection("Logging", logData...)...)
		}
	}

	if inst.data != nil {
		inst.SetRawData(inst.data)
	} else {
		inst.SetRawData(nil)
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
		inst.data, err = inst.serviceCtx.Api.DescribeStateMachine(inst.serviceCtx.Ctx, inst.selectedStateMachineArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
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
