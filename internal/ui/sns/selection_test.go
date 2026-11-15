package sns

import (
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/sns/snsfake"
	"aws-tui/internal/ui/uitest"
)

func TestDetailsPage__MovingThroughTopicsLoadsOnlyTheTopicItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = snsfake.New(snsfake.WithTopics([]string{
			topicArn("billing"), topicArn("invoices"), topicArn("orders"), topicArn("refunds"),
		}))
		api.SetAttributes(newTopicAttributes())
		var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
		var page = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
		uitest.RunApp(t, serviceCtx.App)
		page.TopicListTable.RefreshTopics(true)
		uitest.Eventually(t, serviceCtx.App, func() bool { return page.TopicListTable.GetTable().GetRowCount() == 5 })

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, serviceCtx.App, uitest.MoveThrough[string]{
			Table: page.TopicListTable, Passed: topicArn("orders"), StopsOn: topicArn("refunds"),
			// Both the attributes and the subscriptions of a topic are loaded
			Loaded: func() []string {
				var got []string
				for _, call := range api.Calls() {
					if _, arn, ok := strings.Cut(call, " "); ok &&
						(strings.HasPrefix(call, "attributes ") || strings.HasPrefix(call, "subscriptions ")) {
						got = append(got, arn)
					}
				}
				return got
			},
			// What is published to follows the selection straight away
			StraightAway: func() {
				if got := page.PublishView.selectedTopicArn; got != topicArn("refunds") {
					t.Errorf("publishing to %q, want %q", got, topicArn("refunds"))
				}
			},
		})
	})
}
