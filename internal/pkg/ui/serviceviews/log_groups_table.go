package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogGroupsTable struct {
	*core.SelectableTable[string]
	data             []types.LogGroup
	selectedLogGroup string
	logger           *log.Logger
	app              *tview.Application
	api              *awsapi.CloudWatchLogsApi
}

func NewLogGroupsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogGroupsTable {

	var view = &LogGroupsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Log Groups",
			core.TableRow{
				"Name",
			},
		),
		data:             nil,
		selectedLogGroup: "",
		logger:           logger,
		app:              app,
		api:              api,
	}

	view.populateLogGroupsTable()
	view.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			view.RefreshLogGroups(view.selectedLogGroup)
		}
		return event
	})

	return view
}

func (inst *LogGroupsTable) populateLogGroupsTable() {
	var tableData []core.TableRow
	var privateData []string

	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.LogGroupName),
		})
		privateData = append(privateData, aws.ToString(row.LogGroupName))
	}

	inst.SetData(tableData)
	inst.SetPrivateData(privateData, 0)
	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(0, 0)
	inst.Table.ScrollToBeginning()
}

func (inst *LogGroupsTable) RefreshLogGroups(search string) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterGroupByName(search)
		} else {
			inst.data = inst.api.ListLogGroups(false)
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateLogGroupsTable()
	})
}

func (inst *LogGroupsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.Table.SetSelectedFunc(func(row, column int) {
        var ref = inst.Table.GetCell(row, 0).Reference
		if row < 1 || ref == nil {
			return
		}

		inst.selectedLogGroup = ref.(string)
		handler(row, column)
	})
}

func (inst *LogGroupsTable) GetSeletedLogGroup() string {
	return inst.selectedLogGroup
}
