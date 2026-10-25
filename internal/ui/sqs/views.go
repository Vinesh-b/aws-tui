package sqs

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SqsTabName = string

const (
	SqsTabNameDetails SqsTabName = "Details"
	SqsTabNameTags    SqsTabName = "Tags"
	SqsTabNameSend    SqsTabName = "Send Message"
)

type SqsTag struct {
	Key   string
	Value string
}

type SqsDetailsPageView struct {
	*core.ServicePageView
	QueueListTable    *QueueListTable
	QueueDetailsTable *QueueDetailsTable
	QueueTagsTable    *core.TagsTable[SqsTag, awsapi.SqsApi]
	SendMessageView   *SendMessagePageView
	TabView           *core.TabViewHorizontal
	serviceCtx        *core.ServiceContext[awsapi.SqsApi]
}

func NewSqsDetailsPageView(
	queueListTable *QueueListTable,
	queueDetailsTable *QueueDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.SqsApi],
) *SqsDetailsPageView {
	var tagsTable = core.NewTagsTable[SqsTag](serviceCtx).
		SetExtractKeyValFunc(func(t SqsTag) (k string, v string) {
			return t.Key, t.Value
		}).
		SetGetTagsFunc(func() func() ([]SqsTag, error) {
			var queueUrl = queueListTable.GetSelectedQueue()
			return func() ([]SqsTag, error) {
				var tagsMap, err = serviceCtx.Api.ListQueueTags(serviceCtx.Ctx, queueUrl)
				var tags = []SqsTag{}
				for k, v := range tagsMap {
					tags = append(tags, SqsTag{Key: k, Value: v})
				}
				return tags, err
			}
		})

	var sendMessageView = NewSendMessagePageView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SqsTabNameDetails, queueDetailsTable, 0, 1, true).
		AddTab(SqsTabNameTags, tagsTable, 0, 1, true).
		AddTab(SqsTabNameSend, sendMessageView, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		queueListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SqsDetailsPageView{
		ServicePageView:   serviceView,
		QueueListTable:    queueListTable,
		QueueDetailsTable: queueDetailsTable,
		QueueTagsTable:    tagsTable,
		SendMessageView:   sendMessageView,
		TabView:           tabView,
		serviceCtx:        serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	queueDetailsTable.ErrorMessageCallback = errorHandler
	queueListTable.ErrorMessageCallback = errorHandler
	tagsTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{queueListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *SqsDetailsPageView) initInputCapture() {
	inst.QueueListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		if inst.QueueListTable.GetSelectedQueue() == "" {
			return
		}
		inst.QueueDetailsTable.RefreshDetails(inst.QueueListTable.GetSelectedQueue())
		inst.QueueTagsTable.RefreshDetails()
		inst.SendMessageView.SetSelectedQueue(inst.QueueListTable.GetSelectedQueue())
	})

	inst.SendMessageView.SetOnMessageSent(func(queueUrl string) {
		// The message counts of the queue have changed
		if queueUrl == inst.QueueListTable.GetSelectedQueue() {
			inst.QueueDetailsTable.RefreshDetails(queueUrl)
		}
	})
}

func NewSqsHomeView(appCtx *core.AppContext, api awsapi.SqsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x883300))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		sqsDetailsView = NewSqsDetailsPageView(
			NewQueueListTable(serviceCtx),
			NewQueueDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.SQS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Queues", sqsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
