package sns

import (
	"aws-tui/internal/ui/sns/snsfake"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

func newPopulatedTopicListTable(names ...string) *TopicListTable {
	var table = NewTopicListTable(uitest.NewServiceContext[awsapi.SnsApi]())
	var arns []string
	for _, name := range names {
		arns = append(arns, topicArn(name))
	}
	table.populateTopicTable(arns)
	return table
}

func TestTopicListTable__ShowsHeadingsAndOneRowPerTopic(t *testing.T) {
	var table = newPopulatedTopicListTable("orders", "billing.fifo")

	uitest.AssertTableRows(t, table, []core.TableRow{
		topicHeadings,
		{"orders", "Standard"},
		{"billing.fifo", "FIFO"},
	})
}

func TestTopicListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	uitest.AssertTableRows(t, newPopulatedTopicListTable(), []core.TableRow{topicHeadings})
}

func TestTopicListTable__SelectingRowTracksSelectedTopicArn(t *testing.T) {
	var table = newPopulatedTopicListTable("orders", "billing")

	table.Select(2, 0)

	if got := table.GetSelectedTopic(); got != topicArn("billing") {
		t.Errorf("selected topic = %q, want %q", got, topicArn("billing"))
	}
}

func TestTopicListTable__RefreshLoadsTopicsFromApi(t *testing.T) {
	var api = snsfake.New(snsfake.WithTopics([]string{topicArn("orders")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewTopicListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshTopics(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{topicHeadings, {"orders", "Standard"}})
}

func TestTopicListTable__RefreshReportsApiError(t *testing.T) {
	var api = snsfake.New(snsfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewTopicListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshTopics(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestTopicListTable__SearchShowsMatchingTopics(t *testing.T) {
	var api = snsfake.New(snsfake.WithTopics([]string{topicArn("orders"), topicArn("billing")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewTopicListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshTopics(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("bill") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{topicHeadings, {"billing", "Standard"}})
}

func TestTopicDetailsTable__EmptyTopicShowsBlankValues(t *testing.T) {
	var table = NewTopicDetailsTable(uitest.NewServiceContext[awsapi.SnsApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", ""},
		{"Type", "Standard"},
		{"ARN", ""},
		{"Display Name", ""},
		{"Owner", ""},
		{"Encryption", "None"},
		{"Delivery Policy", ""},
		{"Subscriptions"},
		{"Confirmed", ""},
		{"Pending", ""},
		{"Deleted", ""},
	})
}

func TestTopicDetailsTable__PopulatesFromAttributes(t *testing.T) {
	var table = NewTopicDetailsTable(uitest.NewServiceContext[awsapi.SnsApi]())
	table.topicArn = topicArn("orders.fifo")
	table.attributes = newTopicAttributes()
	table.attributes["ContentBasedDeduplication"] = "true"

	table.populateTopicDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders.fifo"},
		{"Type", "FIFO"},
		{"ARN", topicArn("orders.fifo")},
		{"Display Name", "Orders"},
		{"Owner", "123456789012"},
		{"Encryption", "KMS: alias/aws/sns"},
		{"Delivery Policy", `{"http":{}}`},
		{"Content Based Deduplication", "true"},
		{"Subscriptions"},
		{"Confirmed", "3"},
		{"Pending", "1"},
		{"Deleted", "0"},
	})
}

func TestTopicDetailsTable__RefreshLoadsAttributesFromApi(t *testing.T) {
	var api = snsfake.New(snsfake.WithAttributes(newTopicAttributes()))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewTopicDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshDetails(topicArn("orders"))

	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(8, 1) == "3" })
}

func TestTopicDetailsTable__RefreshReportsApiError(t *testing.T) {
	var api = snsfake.New(snsfake.WithErr(errors.New("topic does not exist")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewTopicDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshDetails(topicArn("orders"))

	if got := uitest.AwaitError(t, received); got != "topic does not exist" {
		t.Errorf("message = %q, want %q", got, "topic does not exist")
	}
}

func TestHomeView__ResetKeyLoadsTopicsAndTheirDetails(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = snsfake.New(snsfake.WithTopics([]string{topicArn("orders-topic")}), snsfake.WithAttributes(newTopicAttributes()))
	var screen = uitest.ShowPage(t, appCtx.App, NewSnsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-topic")
	uitest.AwaitScreenText(t, appCtx.App, screen, "alias/aws/sns")
}

func TestDetailsPageView__TagsAreLoadedForTheSelectedTopic(t *testing.T) {
	var api = snsfake.New(snsfake.WithTopics([]string{topicArn("orders")}), snsfake.WithAttributes(newTopicAttributes()), snsfake.WithTags([]types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var view = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	view.TopicListTable.RefreshTopics(true)

	uitest.AwaitDetailsTag(t, serviceCtx.App, view.TopicDetailsTable.DetailsTable, "env", "prod")
}

func TestDetailsPageView__FirstLoadDoesNotLoadDetailsWithoutATopicArn(t *testing.T) {
	for name, topics := range map[string][]string{
		"topics":    {topicArn("orders")},
		"no topics": nil,
	} {
		t.Run(name, func(t *testing.T) {
			var api = snsfake.New(snsfake.WithTopics(topics), snsfake.WithAttributes(newTopicAttributes()))
			var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
			var view = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
			var callback, received = uitest.CaptureErrors()
			view.TopicDetailsTable.ErrorMessageCallback = callback
			uitest.RunApp(t, serviceCtx.App)

			view.TopicListTable.RefreshTopics(true)
			uitest.Eventually(t, serviceCtx.App, func() bool {
				return len(topics) == 0 || view.TopicListTable.GetSelectedTopic() != ""
			})

			select {
			case message := <-received:
				t.Fatalf("Unexpected error on first load: %q", message)
			case <-time.After(200 * time.Millisecond):
			}
			if slices.ContainsFunc(api.Calls(), func(call string) bool { return strings.HasSuffix(call, " ") }) {
				t.Fatalf("Details were loaded without a topic ARN: %v", api.Calls())
			}
		})
	}
}

func TestTopicName__IsTheLastPartOfTheArn(t *testing.T) {
	if got := topicName(topicArn("orders")); got != "orders" {
		t.Errorf("topicName = %q, want orders", got)
	}
	if got := topicName("plain"); got != "plain" {
		t.Errorf("topicName = %q, want plain", got)
	}
}
