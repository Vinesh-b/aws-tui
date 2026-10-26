package sqs

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func TestHomeView__ResetKeyLoadsQueuesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeSqsApi{queues: []string{queueUrl("orders-queue")}, attributes: newQueueAttributes()}
	var screen = uitest.ShowPage(t, appCtx.App, NewSqsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-queue")
}

func TestHomeView__LoadedQueueShowsItsAttributes(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeSqsApi{
		queues:     []string{queueUrl("orders-queue")},
		attributes: newQueueAttributes(),
	}
	var screen = uitest.ShowPage(t, appCtx.App, NewSqsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "arn:aws:sqs:us-east-1:123456789012:orders")
}

// strictSqsApi fails like the real api does when no queue URL is given.
type strictSqsApi struct {
	fakeSqsApi
	mtx   sync.Mutex
	calls []string
}

func (f *strictSqsApi) record(call string, queueUrl string) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls = append(f.calls, call+" "+queueUrl)
	if queueUrl == "" {
		return errors.New("queueUrl not set")
	}
	return nil
}

func (f *strictSqsApi) GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error) {
	return f.attributes, f.record("attributes", queueUrl)
}

func (f *strictSqsApi) ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error) {
	return f.tags, f.record("tags", queueUrl)
}

func (f *strictSqsApi) recorded() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.calls)
}

func TestDetailsPageView__FirstLoadDoesNotLoadDetailsWithoutAQueueUrl(t *testing.T) {
	for name, queues := range map[string][]string{
		"queues":    {queueUrl("orders-queue")},
		"no queues": nil,
	} {
		t.Run(name, func(t *testing.T) {
			var api = &strictSqsApi{fakeSqsApi: fakeSqsApi{queues: queues, attributes: newQueueAttributes()}}
			var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
			var view = NewSqsDetailsPageView(
				NewQueueListTable(serviceCtx), NewQueueDetailsTable(serviceCtx), serviceCtx,
			)
			var callback, received = uitest.CaptureErrors()
			view.QueueDetailsTable.ErrorMessageCallback = callback
			uitest.RunApp(t, serviceCtx.App)

			view.QueueListTable.RefreshQueues(true)
			uitest.Eventually(t, serviceCtx.App, func() bool {
				return len(queues) == 0 || view.QueueListTable.GetSelectedQueue() != ""
			})
			time.Sleep(200 * time.Millisecond)

			select {
			case message := <-received:
				t.Fatalf("Unexpected error on first load: %q", message)
			default:
			}
			if slices.Contains(api.recorded(), "attributes ") || slices.Contains(api.recorded(), "tags ") {
				t.Fatalf("Details were loaded without a queue URL: %v", api.recorded())
			}
		})
	}
}
