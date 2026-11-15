package sns

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/sns/snsfake"
	"aws-tui/internal/ui/uitest"
)

func TestDetailsPage__MovingThroughTopicsLoadsOnlyTheTopicItStopsOn(t *testing.T) {
	var api = snsfake.New(snsfake.WithTopics([]string{
		topicArn("billing"), topicArn("invoices"), topicArn("orders"), topicArn("refunds"),
	}))
	api.SetAttributes(newTopicAttributes())
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var page = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	page.TopicListTable.RefreshTopics(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return page.TopicListTable.GetTable().GetRowCount() == 5 })

	var passed, last, publishTo string
	uitest.OnUI(t, serviceCtx.App, func() {
		for row := 2; row <= 4; row++ {
			page.TopicListTable.Select(row, 0)
			if row == 3 {
				passed = page.TopicListTable.GetSelectedTopic()
			}
		}
		last = page.TopicListTable.GetSelectedTopic()
		publishTo = page.PublishView.selectedTopicArn
	})

	// What is published to follows the selection straight away
	if publishTo != last {
		t.Errorf("publishing to %q, want %q", publishTo, last)
	}
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return slices.Contains(api.Calls(), "attributes "+last)
	}, "the topic stopped on to be loaded")
	uitest.Consistently(t, serviceCtx.App, 2*core.SelectionLoadDelay, func() bool {
		return !slices.Contains(api.Calls(), "attributes "+passed) &&
			!slices.Contains(api.Calls(), "subscriptions "+passed)
	}, "the topic moved through not to be loaded")
}
