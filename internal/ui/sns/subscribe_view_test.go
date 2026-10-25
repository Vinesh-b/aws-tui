package sns

import (
	"errors"
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newSubscribeTable(t *testing.T, api *fakeSnsApi) *SubscriptionsTable {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewSubscriptionsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return table
}

func statusMessage(table *SubscriptionsTable) string {
	return table.SubscribeView.Input.message.GetText(true)
}

func TestSubscribeView__SubscribesTheEndpointAndReloadsTheTable(t *testing.T) {
	var api = &fakeSnsApi{subscribeArn: "arn:sub:new"}
	var table = newSubscribeTable(t, api)
	table.RefreshSubscriptions(topicArn("orders"))
	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{subscriptionHeadings})
	uitest.OnUI(t, table.serviceCtx.App, func() {
		table.SubscribeView.Input.endpointInput.SetText("  a@example.com ")
		table.subscribe(table.SubscribeView.Input.GetProtocol(), table.SubscribeView.Input.GetEndpoint())
	})

	uitest.AwaitRows(t, table.serviceCtx.App, table, []core.TableRow{
		subscriptionHeadings,
		{"email", "a@example.com", "Confirmed", "arn:sub:new"},
	})
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return statusMessage(table) == "Subscription created: arn:sub:new"
	})
	if !slices.Contains(api.recorded(), "subscribe "+topicArn("orders")+" email a@example.com") {
		t.Errorf("Unexpected calls %v", api.recorded())
	}
}

func TestSubscribeView__SubscribeButtonUsesTheSelectedProtocolAndEndpoint(t *testing.T) {
	var api = &fakeSnsApi{subscribeArn: "arn:sub:new"}
	var table = newSubscribeTable(t, api)
	table.topicArn = topicArn("orders")
	var view = table.SubscribeView.Input
	uitest.OnUI(t, table.serviceCtx.App, func() {
		view.protocolDropDown.SetCurrentOption(slices.Index(subscriptionProtocols, "sqs"))
		view.endpointInput.SetText("arn:aws:sqs:us-east-1:1:queue")
		view.subscribeButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	})

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return slices.Contains(
			api.recorded(), "subscribe "+topicArn("orders")+" sqs arn:aws:sqs:us-east-1:1:queue",
		)
	})
}

func TestSubscribeView__ReportsApiError(t *testing.T) {
	var api = &fakeSnsApi{err: errors.New("invalid endpoint")}
	var table = newSubscribeTable(t, api)
	table.topicArn = topicArn("orders")

	uitest.OnUI(t, table.serviceCtx.App, func() { table.subscribe("email", "nope") })

	uitest.Eventually(t, table.serviceCtx.App, func() bool { return statusMessage(table) == "invalid endpoint" })
}

func TestSubscribeView__RejectsMissingTopicOrEndpointWithoutCallingTheApi(t *testing.T) {
	for name, tc := range map[string]struct{ topic, endpoint, want string }{
		"no topic":    {"", "a@example.com", "No topic selected"},
		"no endpoint": {topicArn("orders"), "  ", "Endpoint is empty"},
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakeSnsApi{}
			var table = newSubscribeTable(t, api)
			table.topicArn = tc.topic

			uitest.OnUI(t, table.serviceCtx.App, func() { table.subscribe("email", tc.endpoint) })

			if got := statusMessage(table); got != tc.want {
				t.Errorf("status = %q, want %q", got, tc.want)
			}
			if calls := api.recorded(); len(calls) != 0 {
				t.Errorf("Expected no api calls but got %v", calls)
			}
		})
	}
}

func TestSubscribeView__OpensWithTheSubscribeKeyAndStartsClean(t *testing.T) {
	var api = &fakeSnsApi{}
	var table = newSubscribeTable(t, api)
	var app = table.serviceCtx.App
	uitest.OnUI(t, app, func() {
		app.SetRoot(table, true)
		app.SetFocus(table)
		table.SubscribeView.Input.endpointInput.SetText("left over")
		table.SubscribeView.Input.SetStatusMessage("old")
	})

	// Sent through the whole focus chain, as the overlay key is handled by a parent view
	app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, subscribeKey, tcell.ModNone))

	uitest.Eventually(t, app, func() bool { return table.IsAnOverlayVisible() })
	uitest.OnUI(t, app, func() {
		if got := table.SubscribeView.Input.GetEndpoint(); got != "" {
			t.Errorf("endpoint = %q, want it cleared", got)
		}
		if got := statusMessage(table); got != "" {
			t.Errorf("status = %q, want it cleared", got)
		}
	})
}
