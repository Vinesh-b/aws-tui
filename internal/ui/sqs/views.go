package sqs

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SqsTabName = string

const (
	SqsTabNameDetails SqsTabName = "Details"
	SqsTabNameSend    SqsTabName = "Send Message"
)

type SqsDetailsPageView struct {
	*core.ServicePageView
	QueueListTable    *QueueListTable
	QueueDetailsTable *QueueDetailsTable
	SendMessageView   *SendMessagePageView
	TabView           *core.TabViewHorizontal
	serviceCtx        *core.ServiceContext[awsapi.SqsApi]
}

func NewSqsDetailsPageView(
	queueListTable *QueueListTable,
	queueDetailsTable *QueueDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.SqsApi],
) *SqsDetailsPageView {
	var sendMessageView = NewSendMessagePageView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SqsTabNameDetails, queueDetailsTable, 0, 1, true).
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
		SendMessageView:   sendMessageView,
		TabView:           tabView,
		serviceCtx:        serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	queueDetailsTable.ErrorMessageCallback = errorHandler
	queueListTable.ErrorMessageCallback = errorHandler

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

	serviceRootView.SetResourceOpener("sqs", func(resource arn.ARN) {
		serviceRootView.ChangePage(0, nil)
		sqsDetailsView.QueueListTable.OpenQueue(resource.Resource)
	})

	return serviceRootView
}
