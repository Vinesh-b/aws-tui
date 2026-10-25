package sqs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type fakeSendMessageApi struct {
	fakeSqsApi
	mtx      sync.Mutex
	queueUrl string
	body     string
	groupId  string
	dedupId  string
	calls    int
	sendErr  error
}

func (f *fakeSendMessageApi) SendMessage(
	ctx context.Context, queueUrl string, body string, groupId string, deduplicationId string,
) (*sqs.SendMessageOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls++
	f.queueUrl, f.body, f.groupId, f.dedupId = queueUrl, body, groupId, deduplicationId
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return &sqs.SendMessageOutput{
		MessageId:      aws.String("m-1"),
		SequenceNumber: aws.String("42"),
	}, nil
}

func (f *fakeSendMessageApi) sendCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.calls
}

func newSendMessageView(t *testing.T, api *fakeSendMessageApi) *SendMessagePageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var view = NewSendMessagePageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestSendMessageView__SendsTheBodyToTheSelectedQueue(t *testing.T) {
	var api = &fakeSendMessageApi{}
	var view = newSendMessageView(t, api)
	var sent = make(chan string, 1)
	view.SetOnMessageSent(func(queueUrl string) { sent <- queueUrl })
	view.SetSelectedQueue(queueUrl("orders"))
	view.bodyArea.SetText("hello\nworld", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	if got := <-sent; got != queueUrl("orders") {
		t.Errorf("Unexpected queue %q", got)
	}
	if api.body != "hello\nworld" || api.groupId != "" || api.dedupId != "" {
		t.Errorf("Unexpected message %q / %q / %q", api.body, api.groupId, api.dedupId)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Message ID: m-1")
	})
}

func TestSendMessageView__FifoQueueSendsGroupAndDeduplicationId(t *testing.T) {
	var api = &fakeSendMessageApi{}
	var view = newSendMessageView(t, api)
	var sent = make(chan string, 1)
	view.SetOnMessageSent(func(queueUrl string) { sent <- queueUrl })
	view.SetSelectedQueue(queueUrl("orders.fifo"))
	view.bodyArea.SetText("hello", false)
	view.groupIdInput.SetText("g1")
	view.dedupIdInput.SetText("d1")

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	<-sent
	if api.groupId != "g1" || api.dedupId != "d1" {
		t.Errorf("Unexpected IDs %q / %q", api.groupId, api.dedupId)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Sequence Number: 42")
	})
}

func TestSendMessageView__FifoQueueRequiresAGroupId(t *testing.T) {
	var api = &fakeSendMessageApi{}
	var view = newSendMessageView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.bodyArea.ErrorMessageCallback = callback
	view.SetSelectedQueue(queueUrl("orders.fifo"))
	view.bodyArea.SetText("hello", false)

	uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "Group ID") {
		t.Fatalf("Unexpected error %q", message)
	}
	if api.sendCalls() != 0 {
		t.Fatalf("Expected no api calls but got %d", api.sendCalls())
	}
}

func TestSendMessageView__RejectsMissingQueueOrBodyWithoutCallingTheApi(t *testing.T) {
	for name, setup := range map[string]func(view *SendMessagePageView){
		"no queue": func(view *SendMessagePageView) { view.bodyArea.SetText("hello", false) },
		"no body":  func(view *SendMessagePageView) { view.SetSelectedQueue(queueUrl("orders")) },
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakeSendMessageApi{}
			var view = newSendMessageView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.bodyArea.ErrorMessageCallback = callback
			setup(view)

			uitest.OnUI(t, view.serviceCtx.App, view.SendMessage)

			uitest.AwaitError(t, received)
			if api.sendCalls() != 0 {
				t.Fatalf("Expected no api calls but got %d", api.sendCalls())
			}
		})
	}
}

func TestSendMessageView__ReportsApiError(t *testing.T) {
	var api = &fakeSendMessageApi{sendErr: errors.New("access denied")}
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
	var view = newSendMessageView(t, &fakeSendMessageApi{})
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
	var api = &fakeSendMessageApi{fakeSqsApi: fakeSqsApi{
		queues: []string{queueUrl("orders")}, attributes: newQueueAttributes(),
	}}
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
