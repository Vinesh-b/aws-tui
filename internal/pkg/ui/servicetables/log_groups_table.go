package servicetables

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
	view.SetSelectedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			view.RefreshLogGroups(view.selectedLogGroup)
		}
		return event
	})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			view.RefreshLogGroups(view.GetSearchText())
			view.app.SetFocus(view)
		}
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
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LogGroupsTable) RefreshLogGroups(search string) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		if len(search) > 0 {
			inst.data = inst.api.FilterGroupByName(search)
		} else {
			inst.data, err = inst.api.ListLogGroups(false)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogGroupsTable()
	})
}

func (inst *LogGroupsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		var ref = inst.GetCell(row, 0).Reference
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
