package serviceviews

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogStreamsTable struct {
	*core.SelectableTable[any]
	selectedLogStream  string
	selectedLogGroup   string
	searchStreamPrefix string
	data               []types.LogStream
	logger             *log.Logger
	app                *tview.Application
	api                *awsapi.CloudWatchLogsApi
}

func NewLogStreamsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogStreamsTable {

	var view = &LogStreamsTable{
		SelectableTable: core.NewSelectableTable[any](
			"LogStreams",
			core.TableRow{
				"Name",
				"LastEventTimestamp",
			},
		),
		selectedLogStream:  "",
		selectedLogGroup:   "",
		searchStreamPrefix: "",
		data:               nil,
		logger:             logger,
		app:                app,
		api:                api,
	}

	view.populateLogStreamsTable(false)
	view.SetSelectedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			view.RefreshStreams(true)
		}
		return event
	})

	return view
}

func (inst *LogStreamsTable) populateLogStreamsTable(extend bool) {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.LogStreamName),
			time.UnixMilli(aws.ToInt64(row.LastEventTimestamp)).Format(time.DateTime),
		})
	}

	if extend {
		inst.ExtendData(tableData)
		return
	}

	inst.SetData(tableData)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LogStreamsTable) RefreshStreams(force bool) {
	var resultChannel = make(chan struct{})
	go func() {
		var err error = nil
		inst.data, err = inst.api.ListLogStreams(
			inst.selectedLogGroup,
			inst.searchStreamPrefix,
			force,
		)
		if err != nil {
			inst.ErrorMessageHandler(err.Error())
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		inst.populateLogStreamsTable(!force)
	})
}

func (inst *LogStreamsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 0 {
			return
		}

		inst.selectedLogStream = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *LogStreamsTable) GetSeletedLogStream() string {
	return inst.selectedLogStream
}

func (inst *LogStreamsTable) GetSeletedLogGroup() string {
	return inst.selectedLogGroup
}

func (inst *LogStreamsTable) SetSeletedLogGroup(logGroup string) {
	inst.selectedLogGroup = logGroup
}

func (inst *LogStreamsTable) SetLogStreamSearchPrefix(prefix string) {
	inst.searchStreamPrefix = prefix
}
