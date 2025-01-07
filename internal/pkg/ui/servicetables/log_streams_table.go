package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"log"
	"slices"
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
            app,
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
	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			view.SetLogStreamSearchPrefix(view.GetSearchText())
			view.RefreshStreams(true)
			view.app.SetFocus(view)
		}
	})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshStreams(true)
		case core.APP_KEY_BINDINGS.NextPage:
			view.RefreshStreams(false)
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
		inst.ExtendData(tableData, nil)
		return
	}

	inst.SetData(tableData, nil, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.ScrollToBeginning()
}

func (inst *LogStreamsTable) RefreshStreams(force bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.data, err = inst.api.ListLogStreams(
			inst.selectedLogGroup,
			inst.searchStreamPrefix,
			force,
		)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogStreamsTable(!force)
	})
}

func (inst *LogStreamsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}

		inst.selectedLogStream = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *LogStreamsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}

		inst.selectedLogStream = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *LogStreamsTable) GetSeletedLogStream() string {
	return inst.selectedLogStream
}

func (inst *LogStreamsTable) GetLogStreamDetail() types.LogStream {
	var idx = slices.IndexFunc(inst.data, func(d types.LogStream) bool {
		return aws.ToString(d.LogStreamName) == inst.selectedLogStream
	})
	if idx == -1 {
		return types.LogStream{}
	}

	return inst.data[idx]
}

func (inst *LogStreamsTable) GetSeletedLogGroup() string {
	return inst.selectedLogGroup
}

func (inst *LogStreamsTable) SetSeletedLogGroup(logGroup string) {
	inst.selectedLogGroup = logGroup
	inst.SetTitleExtra(logGroup)
}

func (inst *LogStreamsTable) SetLogStreamSearchPrefix(prefix string) {
	inst.searchStreamPrefix = prefix
}
