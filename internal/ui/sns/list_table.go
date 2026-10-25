package sns

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/gdamore/tcell/v2"
)

const fifoSuffix = ".fifo"

type TopicListTable struct {
	*core.SelectableTable[string]
	data          []string
	filtered      []string
	selectedTopic string
	serviceCtx    *core.ServiceContext[awsapi.SnsApi]
}

func NewTopicListTable(
	serviceCtx *core.ServiceContext[awsapi.SnsApi],
) *TopicListTable {

	var view = &TopicListTable{
		SelectableTable: core.NewSelectableTable[string](
			"Topics",
			core.TableRow{
				"Name",
				"Type",
			},
			serviceCtx.AppContext,
		),
		data:          nil,
		selectedTopic: "",
		serviceCtx:    serviceCtx,
	}

	view.populateTopicTable(view.data)
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

// topicName is the last part of the topic ARN
func topicName(topicArn string) string {
	if idx := strings.LastIndex(topicArn, ":"); idx >= 0 {
		return topicArn[idx+1:]
	}
	return topicArn
}

func topicType(topicArn string) string {
	if strings.HasSuffix(topicArn, fifoSuffix) {
		return "FIFO"
	}
	return "Standard"
}

func (inst *TopicListTable) populateTopicTable(data []string) {
	var tableData []core.TableRow
	for _, topicArn := range data {
		tableData = append(tableData, core.TableRow{
			topicName(topicArn),
			topicType(topicArn),
		})
	}

	inst.SetData(tableData, data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *TopicListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(name, inst.data, topicName)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTopicTable(inst.filtered)
	})
}

func (inst *TopicListTable) RefreshTopics(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var topics []string

	dataLoader.AsyncLoadData(func() {
		var err error
		topics, err = inst.serviceCtx.Api.ListTopics(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = topics
		inst.populateTopicTable(inst.data)
	})
}

func (inst *TopicListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedTopic = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *TopicListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedTopic = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *TopicListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshTopics(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *TopicListTable) GetSelectedTopic() string {
	return inst.selectedTopic
}
