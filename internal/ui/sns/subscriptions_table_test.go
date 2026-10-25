package sns

import (
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

var subscriptionHeadings = core.TableRow{"Protocol", "Endpoint", "Status", "Subscription Arn"}

func subscription(protocol string, endpoint string, arn string) types.Subscription {
	return types.Subscription{
		Protocol:        aws.String(protocol),
		Endpoint:        aws.String(endpoint),
		SubscriptionArn: aws.String(arn),
	}
}

func TestSubscriptionsTable__EmptyShowsOnlyHeadings(t *testing.T) {
	var table = NewSubscriptionsTable(uitest.NewServiceContext[awsapi.SnsApi]())

	uitest.AssertTableRows(t, table, []core.TableRow{subscriptionHeadings})
}

func TestSubscriptionsTable__ShowsSortedSubscriptionsWithStatus(t *testing.T) {
	var table = NewSubscriptionsTable(uitest.NewServiceContext[awsapi.SnsApi]())

	table.populateSubscriptionsTable([]types.Subscription{
		subscription("sqs", "arn:aws:sqs:us-east-1:1:queue", "arn:sub:3"),
		subscription("email", "b@example.com", pendingConfirmation),
		subscription("email", "a@example.com", "arn:sub:1"),
	})

	uitest.AssertTableRows(t, table, []core.TableRow{
		subscriptionHeadings,
		{"email", "a@example.com", "Confirmed", "arn:sub:1"},
		{"email", "b@example.com", "Pending", pendingConfirmation},
		{"sqs", "arn:aws:sqs:us-east-1:1:queue", "Confirmed", "arn:sub:3"},
	})
}

func TestSubscriptionsTable__RefreshLoadsSubscriptionsFromApi(t *testing.T) {
	var api = &fakeSnsApi{subscriptions: []types.Subscription{subscription("email", "a@example.com", "arn:sub:1")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewSubscriptionsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshSubscriptions(topicArn("orders"))

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		subscriptionHeadings,
		{"email", "a@example.com", "Confirmed", "arn:sub:1"},
	})
}

func TestSubscriptionsTable__RefreshWithoutTopicClearsTheTable(t *testing.T) {
	var api = &fakeSnsApi{subscriptions: []types.Subscription{subscription("email", "a@example.com", "arn:sub:1")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewSubscriptionsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshSubscriptions(topicArn("orders"))
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 2 })

	table.RefreshSubscriptions("")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{subscriptionHeadings})
}

func TestSubscriptionsTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeSnsApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewSubscriptionsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshSubscriptions(topicArn("orders"))

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestDetailsPageView__SubscriptionsAreLoadedForTheSelectedTopic(t *testing.T) {
	var api = &fakeSnsApi{
		topics: []string{topicArn("orders")}, attributes: newTopicAttributes(),
		subscriptions: []types.Subscription{subscription("email", "a@example.com", "arn:sub:1")},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var view = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	view.TopicListTable.RefreshTopics(true)

	uitest.Eventually(t, serviceCtx.App, func() bool { return view.SubscriptionsTable.GetCellText(1, 1) == "a@example.com" })
	if view.TabView.GetTab(SnsTabNameSubscriptions) == nil {
		t.Fatalf("Expected a %q tab", SnsTabNameSubscriptions)
	}
}
