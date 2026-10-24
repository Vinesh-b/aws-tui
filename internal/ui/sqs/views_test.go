package sqs

import (
	"testing"

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
