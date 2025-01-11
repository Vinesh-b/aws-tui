package servicetables

import (
	"log"
	"sort"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/rivo/tview"
)

type LambdaTagsTable struct {
	*core.DetailsTable
	data   types.FunctionConfiguration
	tags   map[string]string
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.LambdaApi
}

func NewLambdaTagsTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaTagsTable {
	var table = &LambdaTagsTable{
		DetailsTable: core.NewDetailsTable("Tags"),
		data:         types.FunctionConfiguration{},
		logger:       logger,
		app:          app,
		api:          api,
	}

	table.populateLambdaTagsTable()

	return table
}

func (inst *LambdaTagsTable) populateLambdaTagsTable() {
	var tableData []core.TableRow
	for k, v := range inst.tags {
		tableData = append(tableData, core.TableRow{k, v})
	}

	sort.Slice(tableData, func(i int, j int) bool {
		return tableData[i][0] < tableData[j][0]
	})

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaTagsTable) ClearDetails() {
	inst.tags = nil
	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaTagsTable()
	})
}

func (inst *LambdaTagsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.tags, err = inst.api.ListTags(aws.ToString(inst.data.FunctionArn))
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaTagsTable()
	})
}
