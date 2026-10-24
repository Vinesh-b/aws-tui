package sqs

import (
	"path"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/gdamore/tcell/v2"
)

const fifoSuffix = ".fifo"

type QueueListTable struct {
	*core.SelectableTable[string]
	data          []string
	filtered      []string
	selectedQueue string
	serviceCtx    *core.ServiceContext[awsapi.SqsApi]
}

func NewQueueListTable(
	serviceCtx *core.ServiceContext[awsapi.SqsApi],
) *QueueListTable {

	var view = &QueueListTable{
		SelectableTable: core.NewSelectableTable[string](
			"Queues",
			core.TableRow{
				"Name",
				"Type",
			},
			serviceCtx.AppContext,
		),
		data:          nil,
		selectedQueue: "",
		serviceCtx:    serviceCtx,
	}

	view.populateQueueTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func queueName(queueUrl string) string {
	return path.Base(queueUrl)
}

func queueType(queueUrl string) string {
	if strings.HasSuffix(queueUrl, fifoSuffix) {
		return "FIFO"
	}
	return "Standard"
}

func (inst *QueueListTable) populateQueueTable(data []string) {
	var tableData []core.TableRow
	for _, queueUrl := range data {
		tableData = append(tableData, core.TableRow{
			queueName(queueUrl),
			queueType(queueUrl),
		})
	}

	inst.SetData(tableData, data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *QueueListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(name, inst.data, queueName)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateQueueTable(inst.filtered)
	})
}

func (inst *QueueListTable) RefreshQueues(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var queues []string

	dataLoader.AsyncLoadData(func() {
		var err error
		queues, err = inst.serviceCtx.Api.ListQueues(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = queues
		inst.populateQueueTable(inst.data)
	})
}

func (inst *QueueListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedQueue = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *QueueListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedQueue = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *QueueListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshQueues(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *QueueListTable) GetSelectedQueue() string {
	return inst.selectedQueue
}
