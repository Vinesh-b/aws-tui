package sns

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SnsTabName = string

const (
	SnsTabNameDetails       SnsTabName = "Details"
	SnsTabNameSubscriptions SnsTabName = "Subscriptions"
	SnsTabNameTags          SnsTabName = "Tags"
	SnsTabNamePublish       SnsTabName = "Publish"
)

type SnsDetailsPageView struct {
	*core.ServicePageView
	TopicListTable     *TopicListTable
	TopicDetailsTable  *TopicDetailsTable
	TopicTagsTable     *core.TagsTable[types.Tag, awsapi.SnsApi]
	SubscriptionsTable *SubscriptionsTable
	PublishView        *PublishPageView
	TabView            *core.TabViewHorizontal
	serviceCtx         *core.ServiceContext[awsapi.SnsApi]
}

func NewSnsDetailsPageView(
	topicListTable *TopicListTable,
	topicDetailsTable *TopicDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.SnsApi],
) *SnsDetailsPageView {
	var tagsTable = core.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() func() ([]types.Tag, error) {
			var topicArn = topicListTable.GetSelectedTopic()
			return func() ([]types.Tag, error) {
				return serviceCtx.Api.ListTopicTags(serviceCtx.Ctx, topicArn)
			}
		})

	var subscriptionsTable = NewSubscriptionsTable(serviceCtx)
	var publishView = NewPublishPageView(serviceCtx)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SnsTabNameDetails, topicDetailsTable, 0, 1, true).
		AddTab(SnsTabNameSubscriptions, subscriptionsTable, 0, 1, true).
		AddTab(SnsTabNameTags, tagsTable, 0, 1, true).
		AddTab(SnsTabNamePublish, publishView, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		topicListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SnsDetailsPageView{
		ServicePageView:    serviceView,
		TopicListTable:     topicListTable,
		TopicDetailsTable:  topicDetailsTable,
		TopicTagsTable:     tagsTable,
		SubscriptionsTable: subscriptionsTable,
		PublishView:        publishView,
		TabView:            tabView,
		serviceCtx:         serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	topicDetailsTable.ErrorMessageCallback = errorHandler
	topicListTable.ErrorMessageCallback = errorHandler
	tagsTable.ErrorMessageCallback = errorHandler
	subscriptionsTable.ErrorMessageCallback = errorHandler

	// The views of a deleted topic are emptied, the reloaded list then shows
	// whatever is selected next
	topicListTable.SetOnTopicDeleted(func(topicArn string) {
		if topicArn != view.PublishView.selectedTopicArn {
			return
		}
		topicDetailsTable.ClearDetails()
		tagsTable.ClearDetails()
		subscriptionsTable.RefreshSubscriptions("")
		publishView.SetSelectedTopic("")
	})

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{topicListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *SnsDetailsPageView) initInputCapture() {
	inst.TopicListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		var topicArn = inst.TopicListTable.GetSelectedTopic()
		if topicArn == "" {
			return
		}
		inst.TopicDetailsTable.RefreshDetails(topicArn)
		inst.TopicTagsTable.RefreshDetails()
		inst.SubscriptionsTable.RefreshSubscriptions(topicArn)
		inst.PublishView.SetSelectedTopic(topicArn)
	})
}

func NewSnsHomeView(appCtx *core.AppContext, api awsapi.SnsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC3366))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		snsDetailsView = NewSnsDetailsPageView(
			NewTopicListTable(serviceCtx),
			NewTopicDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.SNS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Topics", snsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
