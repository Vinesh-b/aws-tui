package servicetables

import (
	"fmt"
	"log"
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
					logData = append(logData, core.TableRow{"", aws.ToString(group.LogGroupArn)})
					inst.logGroups = append(inst.logGroups, aws.ToString(group.LogGroupArn))
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
	inst.selectedStateMachineArn = stateMachineArn
	inst.logGroups = nil

	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.data, err = inst.api.DescribeStateMachine(stateMachineArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStateMachineDetailsTableTable()
	})
}

func (inst *StateMachineDetailsTable) GetSelectedSmLogGroup() string {
	if inst.logGroups == nil {
		return ""
	}
	return inst.logGroups[0]
}
