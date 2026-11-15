package sqs

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/sqs/sqsfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func newSendMessageView(t *testing.T, api *sqsfake.Api) *SendMessagePageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var view = NewSendMessagePageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestSendMessageView__SendsTheBodyToTheSelectedQueue(t *testing.T) {
	var api = sqsfake.New()
	var view = newSendMessageView(t, api)
	var sent = make(chan string, 1)
	view.SetOnMessageSent(func(queueUrl string) { sent <- queueUrl })
	view.SetSelectedQueue(queueUrl("orders"))
	view.bodyArea.SetText("hello\nworld", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	if got := <-sent; got != queueUrl("orders") {
		t.Errorf("Unexpected queue %q", got)
	}
	if got := api.Sent(); len(got) != 1 || got[0].QueueUrl != queueUrl("orders") ||
		got[0].Body != "hello\nworld" || got[0].GroupId != "" || got[0].DeduplicationId != "" {
		t.Errorf("sent = %+v, want hello\\nworld to the orders queue, without IDs", got)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Message ID: "+sqsfake.SentMessageId)
	})
}

func TestSendMessageView__FifoQueueSendsGroupAndDeduplicationId(t *testing.T) {
	var api = sqsfake.New()
	var view = newSendMessageView(t, api)
	var sent = make(chan string, 1)
	view.SetOnMessageSent(func(queueUrl string) { sent <- queueUrl })
	view.SetSelectedQueue(queueUrl("orders.fifo"))
	view.bodyArea.SetText("hello", false)
	view.groupIdInput.SetText("g1")
	view.dedupIdInput.SetText("d1")

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	<-sent
	if got := api.Sent(); len(got) != 1 || got[0].GroupId != "g1" || got[0].DeduplicationId != "d1" {
		t.Errorf("sent = %+v, want one with the group g1 and the deduplication id d1", got)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Sequence Number: 42")
	})
}

func TestSendMessageView__FifoQueueRequiresAGroupId(t *testing.T) {
	var api = sqsfake.New()
	var view = newSendMessageView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.bodyArea.ErrorMessageCallback = callback
	view.SetSelectedQueue(queueUrl("orders.fifo"))
	view.bodyArea.SetText("hello", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "Group ID") {
		t.Fatalf("Unexpected error %q", message)
	}
	if got := api.Sent(); len(got) != 0 {
		t.Fatalf("Expected no api calls but got %+v", got)
	}
}

func TestSendMessageView__RejectsMissingQueueOrBodyWithoutCallingTheApi(t *testing.T) {
	for name, setup := range map[string]func(view *SendMessagePageView){
		"no queue": func(view *SendMessagePageView) { view.bodyArea.SetText("hello", false) },
		"no body":  func(view *SendMessagePageView) { view.SetSelectedQueue(queueUrl("orders")) },
	} {
		t.Run(name, func(t *testing.T) {
			var api = sqsfake.New()
			var view = newSendMessageView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.bodyArea.ErrorMessageCallback = callback
			setup(view)

			uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

			uitest.AwaitError(t, received)
			if got := api.Sent(); len(got) != 0 {
				t.Fatalf("Expected no api calls but got %+v", got)
			}
		})
	}
}

func TestSendMessageView__ReportsApiError(t *testing.T) {
	var api = sqsfake.New(sqsfake.WithSendErr(errors.New("access denied")))
	var view = newSendMessageView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnMessageSent(func(string) { t.Error("sent callback must not run on failure") })
	view.SetSelectedQueue(queueUrl("orders"))
	view.bodyArea.SetText("hello", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestSendMessageView__FormatIndentsJsonAndLeavesOtherTextAlone(t *testing.T) {
	var view = newSendMessageView(t, sqsfake.New())
	var callback, received = uitest.CaptureErrors()
	view.bodyArea.ErrorMessageCallback = callback

	view.bodyArea.SetText(`{"big":12345678901234567890}`, false)
	uitest.OnUI(t, view.serviceCtx.App, view.FormatBody)
	if text := view.bodyArea.GetText(); text != "{\n  \"big\": 12345678901234567890\n}" {
		t.Errorf("Unexpected formatted text %q", text)
	}

	view.bodyArea.SetText("plain text", false)
	uitest.OnUI(t, view.serviceCtx.App, view.FormatBody)
	uitest.AwaitError(t, received)
	if text := view.bodyArea.GetText(); text != "plain text" {
		t.Errorf("Expected the body to be untouched but got %q", text)
	}
}

func TestDetailsPageView__SelectingAQueueSelectsItForSending(t *testing.T) {
	var api = sqsfake.New(sqsfake.WithQueues([]string{queueUrl("orders")}), sqsfake.WithAttributes(newQueueAttributes()))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var view = NewSqsDetailsPageView(
		NewQueueListTable(serviceCtx), NewQueueDetailsTable(serviceCtx), serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)

	view.QueueListTable.RefreshQueues(true)

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return view.SendMessageView.selectedQueueUrl == queueUrl("orders")
	})
	if view.TabView.GetTab(SqsTabNameSend) == nil {
		t.Fatalf("Expected a %q tab", SqsTabNameSend)
	}
}

func TestSendMessageView__SendsMessageAttributes(t *testing.T) {
	var api = sqsfake.New()
	var view = newSendMessageView(t, api)
	var sent = make(chan string, 1)
	view.SetOnMessageSent(func(queueUrl string) { sent <- queueUrl })
	view.SetSelectedQueue(queueUrl("orders"))
	view.bodyArea.SetText("hello", false)
	view.attributesArea.SetText("Source=tui\nCount:Number=3", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	<-sent
	var got = api.Sent()
	if len(got) != 1 {
		t.Fatalf("sent = %+v, want one message", got)
	}
	var attrs = got[0].Attributes
	if len(attrs) != 2 ||
		aws.ToString(attrs["Source"].StringValue) != "tui" ||
		aws.ToString(attrs["Count"].DataType) != "Number" {
		t.Errorf("Unexpected attributes %v", attrs)
	}
}

func TestSendMessageView__InvalidAttributesAreReportedWithoutCallingTheApi(t *testing.T) {
	var api = sqsfake.New()
	var view = newSendMessageView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.bodyArea.ErrorMessageCallback = callback
	view.SetSelectedQueue(queueUrl("orders"))
	view.bodyArea.SetText("hello", false)
	view.attributesArea.SetText("Count:Number=abc", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "line 1") {
		t.Fatalf("Unexpected error %q", message)
	}
	if got := api.Sent(); len(got) != 0 {
		t.Fatalf("Expected no api calls but got %+v", got)
	}
}
