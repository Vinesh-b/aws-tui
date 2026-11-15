package sqs

import (
	"slices"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/sqs/sqsfake"
	"aws-tui/internal/ui/uitest"
)

func TestHomeView__ResetKeyLoadsQueuesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = sqsfake.New(sqsfake.WithQueues([]string{queueUrl("orders-queue")}), sqsfake.WithAttributes(newQueueAttributes()))
	var screen = uitest.ShowPage(t, appCtx.App, NewSqsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-queue")
}

func TestHomeView__LoadedQueueShowsItsAttributes(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = sqsfake.New(
		sqsfake.WithQueues([]string{queueUrl("orders-queue")}),
		sqsfake.WithAttributes(newQueueAttributes()),
	)
	var screen = uitest.ShowPage(t, appCtx.App, NewSqsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "arn:aws:sqs:us-east-1:123456789012:orders")
}

func TestDetailsPageView__FirstLoadDoesNotLoadDetailsWithoutAQueueUrl(t *testing.T) {
	for name, queues := range map[string][]string{
		"queues":    {queueUrl("orders-queue")},
		"no queues": nil,
	} {
		t.Run(name, func(t *testing.T) {
			uitest.RunInBubble(t, func(t *testing.T) {
				var api = sqsfake.New(sqsfake.WithQueues(queues), sqsfake.WithAttributes(newQueueAttributes()))
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

				select {
				case message := <-received:
					t.Fatalf("Unexpected error on first load: %q", message)
				case <-time.After(200 * time.Millisecond):
				}
				if slices.Contains(api.Calls(), "attributes ") || slices.Contains(api.Calls(), "tags ") {
					t.Fatalf("Details were loaded without a queue URL: %v", api.Calls())
				}
			})
		})
	}
}
