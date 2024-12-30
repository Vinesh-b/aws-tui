package serviceviews

import (
	"fmt"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LambdaDetailsTable struct {
	*core.DetailsTable
	data *types.FunctionConfiguration

	selectedLambda string
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.LambdaApi
}

func NewLambdaDetailsTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsTable {
	var table = &LambdaDetailsTable{
		DetailsTable:   core.NewDetailsTable("Lambda Details"),
		data:           nil,
		selectedLambda: "",
		logger:         logger,
		app:            app,
		api:            api,
	}

	table.populateLambdaDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshDetails(table.selectedLambda, true)
		}
		return event
	})

	return table
}

func (inst *LambdaDetailsTable) populateLambdaDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = []core.TableRow{
			{"Description", aws.ToString(inst.data.Description)},
			{"Arn", aws.ToString(inst.data.FunctionArn)},
			{"Version", aws.ToString(inst.data.Version)},
			{"MemorySize", fmt.Sprintf("%d", *inst.data.MemorySize)},
			{"Runtime", string(inst.data.Runtime)},
			{"Arch", fmt.Sprintf("%v", inst.data.Architectures)},
			{"Timeout", fmt.Sprintf("%d", *inst.data.Timeout)},
			{"LoggingGroup", aws.ToString(inst.data.LoggingConfig.LogGroup)},
			{"AppLogLevel", string(inst.data.LoggingConfig.ApplicationLogLevel)},
			{"State", string(inst.data.State)},
			{"LastModified", aws.ToString(inst.data.LastModified)},
		}
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaDetailsTable) RefreshDetails(lambdaName string, force bool) {
	inst.selectedLambda = lambdaName
	var data map[string]types.FunctionConfiguration
	var resultChannel = make(chan struct{})

	go func() {
		var err error
		data, err = inst.api.ListLambdas(force)

		if err != nil {
			inst.ErrorMessageHandler(err.Error())
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		var val, ok = data[lambdaName]
		if ok {
			inst.data = &val
		}
		inst.populateLambdaDetailsTable()
	})
}
