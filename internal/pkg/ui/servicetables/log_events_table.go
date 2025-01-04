package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const logMsgCol = 1

type LogEventsTable struct {
	*core.SelectableTable[string]
	data              []types.OutputLogEvent
	selectedLogGroup  string
	selectedLogStream string
	logger            *log.Logger
	app               *tview.Application
	api               *awsapi.CloudWatchLogsApi
}

func NewLogEventsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogEventsTable {

	var view = &LogEventsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Log Events",
			core.TableRow{
				"Timestamp",
				"Message",
			},
		),
		data:              nil,
		selectedLogGroup:  "",
		selectedLogStream: "",
		logger:            logger,
		app:               app,
		api:               api,
	}

	view.HighlightSearch = true
	view.populateLogEventsTable(false)
	view.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshLogEvents(true)
		case core.APP_KEY_BINDINGS.NextPage:
			view.RefreshLogEvents(false)
		}
		return event
	})

	return view
}

func (inst *LogEventsTable) populateLogEventsTable(reset bool) {
	var tableData []core.TableRow
	var privateData []string
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			time.UnixMilli(aws.ToInt64(row.Timestamp)).Format("2006-01-02 15:04:05.000"),
			aws.ToString(row.Message),
		})
		privateData = append(privateData, aws.ToString(row.Message))
	}

	if !reset {
		inst.ExtendData(tableData, privateData)
		return
	}

	inst.SetData(tableData, privateData, logMsgCol)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *LogEventsTable) RefreshLogEvents(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.data, err = inst.api.ListLogEvents(
			inst.selectedLogGroup,
			inst.selectedLogStream,
			reset,
		)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogEventsTable(reset)
	})
}

func (inst *LogEventsTable) SetSeletedLogGroup(logGroup string) {
	inst.selectedLogGroup = logGroup
}

func (inst *LogEventsTable) SetSeletedLogStream(logStream string) {
	inst.selectedLogStream = logStream
}

func (inst *LogEventsTable) GetFullLogMessage(row int) string {
	var msg = inst.GetCell(row, logMsgCol).Reference
	if row < 1 || msg == nil {
		return ""
	}
	return msg.(string)
}
