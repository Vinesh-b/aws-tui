package sns

import (
	"aws-tui/internal/ui/sns/snsfake"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type fakePublishApi struct {
	*snsfake.Api
	pmtx       sync.Mutex
	topicArn   string
	message    string
	subject    string
	groupId    string
	dedupId    string
	attributes map[string]types.MessageAttributeValue
	count      int
	publishErr error
}

func (f *fakePublishApi) Publish(
	ctx context.Context, topicArn string, message string, subject string,
	groupId string, deduplicationId string, attributes map[string]types.MessageAttributeValue,
) (*sns.PublishOutput, error) {
	f.pmtx.Lock()
	defer f.pmtx.Unlock()
	f.count++
	f.topicArn, f.message, f.subject, f.groupId, f.dedupId, f.attributes =
		topicArn, message, subject, groupId, deduplicationId, attributes
	if f.publishErr != nil {
		return nil, f.publishErr
	}
	return &sns.PublishOutput{MessageId: aws.String("m-1"), SequenceNumber: aws.String("42")}, nil
}

func (f *fakePublishApi) publishCalls() int {
	f.pmtx.Lock()
	defer f.pmtx.Unlock()
	return f.count
}

func newPublishView(t *testing.T, api *fakePublishApi) *PublishPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var view = NewPublishPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestPublishView__PublishesTheMessageToTheSelectedTopic(t *testing.T) {
	var api = &fakePublishApi{Api: snsfake.New()}
	var view = newPublishView(t, api)
	view.SetSelectedTopic(topicArn("orders"))
	view.messageArea.SetText("hello\nworld", false)
	view.subjectInput.SetText("Greeting")
	view.attributesArea.SetText("Source=tui\nCount:Number=3", false)

	uitest.OnUI(t, view.serviceCtx.App, view.Publish)

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Message ID: m-1")
	})
	if api.topicArn != topicArn("orders") || api.message != "hello\nworld" || api.subject != "Greeting" {
		t.Errorf("Unexpected publish %q %q %q", api.topicArn, api.message, api.subject)
	}
	if api.groupId != "" || api.dedupId != "" {
		t.Errorf("Unexpected IDs %q / %q", api.groupId, api.dedupId)
	}
	if len(api.attributes) != 2 ||
		aws.ToString(api.attributes["Source"].StringValue) != "tui" ||
		aws.ToString(api.attributes["Count"].DataType) != "Number" {
		t.Errorf("Unexpected attributes %v", api.attributes)
	}
}

func TestPublishView__FifoTopicSendsGroupAndDeduplicationId(t *testing.T) {
	var api = &fakePublishApi{Api: snsfake.New()}
	var view = newPublishView(t, api)
	view.SetSelectedTopic(topicArn("orders.fifo"))
	view.messageArea.SetText("hello", false)
	view.groupIdInput.SetText("g1")
	view.dedupIdInput.SetText("d1")

	uitest.OnUI(t, view.serviceCtx.App, view.Publish)

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Sequence Number: 42")
	})
	if api.groupId != "g1" || api.dedupId != "d1" {
		t.Errorf("Unexpected IDs %q / %q", api.groupId, api.dedupId)
	}
}

func TestPublishView__RejectsInvalidInputWithoutCallingTheApi(t *testing.T) {
	for name, tc := range map[string]struct {
		setup func(view *PublishPageView)
		want  string
	}{
		"no topic":   {func(view *PublishPageView) { view.SetSelectedTopic("") }, "No topic"},
		"no message": {func(view *PublishPageView) { view.messageArea.SetText("", false) }, "Message is empty"},
		"fifo without group": {func(view *PublishPageView) {
			view.SetSelectedTopic(topicArn("orders.fifo"))
		}, "Group ID"},
		"bad attributes": {func(view *PublishPageView) {
			view.attributesArea.SetText("Count:Number=abc", false)
		}, "line 1"},
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakePublishApi{Api: snsfake.New()}
			var view = newPublishView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.messageArea.ErrorMessageCallback = callback
			view.SetSelectedTopic(topicArn("orders"))
			view.messageArea.SetText("hello", false)
			tc.setup(view)

			uitest.OnUI(t, view.serviceCtx.App, view.Publish)

			if message := uitest.AwaitError(t, received); !strings.Contains(message, tc.want) {
				t.Fatalf("Unexpected error %q, want one containing %q", message, tc.want)
			}
			if api.publishCalls() != 0 {
				t.Fatalf("Expected no api calls but got %d", api.publishCalls())
			}
		})
	}
}

func TestPublishView__ReportsApiError(t *testing.T) {
	var api = &fakePublishApi{Api: snsfake.New(), publishErr: errors.New("access denied")}
	var view = newPublishView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetSelectedTopic(topicArn("orders"))
	view.messageArea.SetText("hello", false)

	uitest.OnUI(t, view.serviceCtx.App, view.Publish)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestPublishView__FormatIndentsJsonAndLeavesOtherTextAlone(t *testing.T) {
	var view = newPublishView(t, &fakePublishApi{Api: snsfake.New()})
	var callback, received = uitest.CaptureErrors()
	view.messageArea.ErrorMessageCallback = callback

	view.messageArea.SetText(`{"big":12345678901234567890}`, false)
	uitest.OnUI(t, view.serviceCtx.App, view.FormatMessage)
	if text := view.messageArea.GetText(); text != "{\n  \"big\": 12345678901234567890\n}" {
		t.Errorf("Unexpected formatted text %q", text)
	}

	view.messageArea.SetText("plain text", false)
	uitest.OnUI(t, view.serviceCtx.App, view.FormatMessage)
	uitest.AwaitError(t, received)
	if text := view.messageArea.GetText(); text != "plain text" {
		t.Errorf("Expected the message to be untouched but got %q", text)
	}
}

func TestDetailsPageView__SelectingATopicSelectsItForPublishing(t *testing.T) {
	var api = &fakePublishApi{Api: snsfake.New(
		snsfake.WithTopics([]string{topicArn("orders")}), snsfake.WithAttributes(newTopicAttributes()),
	)}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var view = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	view.TopicListTable.RefreshTopics(true)

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return view.PublishView.selectedTopicArn == topicArn("orders")
	})
	if view.TabView.GetTab(SnsTabNamePublish) == nil {
		t.Fatalf("Expected a %q tab", SnsTabNamePublish)
	}
}
