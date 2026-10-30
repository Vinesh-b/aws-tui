package sns

import (
	"fmt"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

const fifoSuffix = ".fifo"

// deleteTopicKey opens the delete confirmation for the selected topic.
const deleteTopicKey = 'x'

const deleteTopicOverlayId = "DELETE_TOPIC"

type TopicListTable struct {
	*core.SelectableTable[string]
	DeleteView    *core.FloatingConfirmView
	CreateView    *FloatingCreateTopicView
	data          []string
	selectedTopic string
	// toDelete is the topic the delete view was shown for
	toDelete       string
	onTopicDeleted func(topicArn string)
	serviceCtx     *core.ServiceContext[awsapi.SnsApi]
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
		).SetColumnExpansion(0, 1).SetFilterColumns(0),
		CreateView:     NewFloatingCreateTopicView(serviceCtx.AppContext),
		DeleteView:     core.NewFloatingConfirmView(serviceCtx.AppContext, "Delete Topic", "Delete"),
		data:           nil,
		selectedTopic:  "",
		onTopicDeleted: func(string) {},
		serviceCtx:     serviceCtx,
	}

	view.AddRuneToggleOverlay(createTopicOverlayId, view.CreateView, createTopicKey, false)
	view.CreateView.Input.SetOnCreateFunc(view.createTopic)
	view.CreateView.Input.SetOnCloseFunc(func() {
		view.ToggleOverlay(createTopicOverlayId, true)
	})
	view.HelpView.View.AddItem(string(createTopicKey), "Create a topic", nil)

	view.AddRuneToggleOverlay(deleteTopicOverlayId, view.DeleteView, deleteTopicKey, false)
	view.DeleteView.Input.SetOnShowFunc(view.prepareDelete)
	view.DeleteView.Input.SetOnConfirmFunc(func() {
		view.ToggleOverlay(deleteTopicOverlayId, true)
		view.deleteTopic(view.toDelete)
	})
	view.DeleteView.Input.SetOnCancelFunc(func() {
		view.ToggleOverlay(deleteTopicOverlayId, true)
	})
	view.HelpView.View.AddItem(string(deleteTopicKey), "Delete the selected topic", nil)

	view.populateTopicTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

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
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshTopics(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *TopicListTable) GetSelectedTopic() string {
	return inst.selectedTopic
}

// SetOnTopicDeleted sets a function called on the UI goroutine after a topic
// was deleted, before the topics are reloaded.
func (inst *TopicListTable) SetOnTopicDeleted(handler func(topicArn string)) {
	inst.onTopicDeleted = handler
}

// prepareDelete notes the selected topic and asks about it. It is kept so the
// list changing underneath can't change what is deleted.
func (inst *TopicListTable) prepareDelete() {
	var row, _ = inst.GetTable().GetSelection()
	inst.toDelete = ""

	if row < 1 || row >= inst.GetTable().GetRowCount() {
		inst.DeleteView.Input.SetMessage("No topic selected")
		return
	}

	inst.toDelete = inst.GetPrivateData(row, 0)
	inst.DeleteView.Input.SetMessage(fmt.Sprintf(
		"Delete topic %s and all of its subscriptions?\n\nThis can't be undone.", topicName(inst.toDelete),
	))
}

// deleteTopic deletes the topic without blocking the UI, then reloads the
// topics. Failures are shown as errors.
func (inst *TopicListTable) deleteTopic(topicArn string) {
	if topicArn == "" {
		inst.ErrorMessageCallback("No topic selected")
		return
	}

	go func() {
		var err = inst.serviceCtx.Api.DeleteTopic(inst.serviceCtx.Ctx, topicArn)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}

			if inst.selectedTopic == topicArn {
				inst.selectedTopic = ""
			}
			inst.onTopicDeleted(topicArn)
			inst.RefreshTopics(true)
		})
	}()
}

// createTopic creates the topic without blocking the UI, then reloads the
// topics. The result is shown in the create view.
func (inst *TopicListTable) createTopic(name string, fifo bool, contentBasedDeduplication bool) {
	var view = inst.CreateView.Input

	var newName, err = validTopicName(name, fifo)
	if err != nil {
		view.SetStatusMessage(err.Error())
		return
	}
	view.SetStatusMessage("Creating...")

	go func() {
		var topicArn, err = inst.serviceCtx.Api.CreateTopic(
			inst.serviceCtx.Ctx, newName, fifo, contentBasedDeduplication,
		)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				view.SetStatusMessage(err.Error())
				return
			}

			view.SetStatusMessage("Topic created: " + topicArn)
			inst.RefreshTopics(true)
		})
	}()
}
