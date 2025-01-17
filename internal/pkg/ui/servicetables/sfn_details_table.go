package servicetables

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/rivo/tview"
)

type StateMachineDetailsTable struct {
	*core.DetailsTable
	data                    *sfn.DescribeStateMachineOutput
	selectedStateMachineArn string
	logGroups               []string
	logGroupsChan           chan []string
	logger                  *log.Logger
	app                     *tview.Application
	api                     *awsapi.StateMachineApi
}

func NewStateMachineDetailsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineDetailsTable {
	var table = &StateMachineDetailsTable{
		DetailsTable:            core.NewDetailsTable("State Machine Details"),
		data:                    nil,
		logGroups:               []string{},
		logGroupsChan:           make(chan []string),
		selectedStateMachineArn: "",
		logger:                  logger,
		app:                     app,
		api:                     api,
	}

	table.populateStateMachineDetailsTableTable()

	return table
}

func (inst *StateMachineDetailsTable) populateStateMachineDetailsTableTable() {
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

func (inst *StateMachineDetailsTable) ClearDetails() {
	inst.data = nil
	inst.logGroups = nil
	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStateMachineDetailsTableTable()
	})
}

func (inst *StateMachineDetailsTable) RefreshDetails(stateMachineArn string) {
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

	inst.selectedStateMachineArn = stateMachineArn
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.data, err = inst.api.DescribeStateMachine(stateMachineArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
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
		inst.populateStateMachineDetailsTableTable()
	})
}

func (inst *StateMachineDetailsTable) GetSelectedSmLogGroup() string {
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
