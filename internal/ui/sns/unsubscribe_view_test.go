package sns

import (
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/gdamore/tcell/v2"
)

type unsubscribeFixture struct {
	table  *SubscriptionsTable
	api    *fakeSnsApi
	errors <-chan string
}

func newUnsubscribeFixture(t *testing.T, api *fakeSnsApi) *unsubscribeFixture {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewSubscriptionsTable(serviceCtx)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(table, true)
		serviceCtx.App.SetFocus(table)
	})
	table.RefreshSubscriptions(topicArn("orders"))
	return &unsubscribeFixture{table: table, api: api, errors: received}
}

func (f *unsubscribeFixture) press(t *testing.T, event *tcell.EventKey) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.table.serviceCtx.App.QueueEvent(event)
	uitest.OnUI(t, f.table.serviceCtx.App, func() {})
}

func (f *unsubscribeFixture) open(t *testing.T) {
	t.Helper()
	f.press(t, tcell.NewEventKey(tcell.KeyRune, unsubscribeKey, tcell.ModNone))
	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return f.table.IsAnOverlayVisible() })
}

func (f *unsubscribeFixture) confirm(t *testing.T) {
	t.Helper()
	f.press(t, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	f.press(t, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
}

func (f *unsubscribeFixture) message(t *testing.T) string {
	t.Helper()
	var message string
	uitest.OnUI(t, f.table.serviceCtx.App, func() { message = f.table.UnsubscribeView.Input.GetMessage() })
	return message
}

func twoSubscriptions() []types.Subscription {
	return []types.Subscription{
		subscription("email", "a@example.com", "arn:sub:1"),
		subscription("sqs", "arn:aws:sqs:us-east-1:1:queue", "arn:sub:2"),
	}
}

func TestUnsubscribeView__ConfirmingDeletesTheSelectedSubscriptionAndReloads(t *testing.T) {
	var f = newUnsubscribeFixture(t, &fakeSnsApi{subscriptions: twoSubscriptions()})
	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return f.table.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, f.table.serviceCtx.App, func() { f.table.Select(2, 0) })

	f.open(t)
	if got := f.message(t); !strings.Contains(got, "Unsubscribe sqs arn:aws:sqs:us-east-1:1:queue from orders?") {
		t.Errorf("message = %q", got)
	}
	f.confirm(t)

	uitest.AwaitRows(t, f.table.serviceCtx.App, f.table, []core.TableRow{
		subscriptionHeadings,
		{"email", "a@example.com", "Confirmed", "arn:sub:1"},
	})
	if !slices.Contains(f.api.recorded(), "unsubscribe arn:sub:2") {
		t.Errorf("Unexpected calls %v", f.api.recorded())
	}
	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return !f.table.IsAnOverlayVisible() })
}

func TestUnsubscribeView__CancellingDeletesNothing(t *testing.T) {
	var f = newUnsubscribeFixture(t, &fakeSnsApi{subscriptions: twoSubscriptions()})
	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return f.table.GetTable().GetRowCount() == 3 })

	f.open(t)
	// Focus starts on Cancel, so Enter alone must not delete
	f.press(t, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return !f.table.IsAnOverlayVisible() })
	if slices.ContainsFunc(f.api.recorded(), func(call string) bool { return strings.HasPrefix(call, "unsubscribe") }) {
		t.Errorf("Unexpected calls %v", f.api.recorded())
	}
}

func TestUnsubscribeView__PendingSubscriptionIsWarnedAboutAndReportsTheError(t *testing.T) {
	var f = newUnsubscribeFixture(t, &fakeSnsApi{
		subscriptions: []types.Subscription{subscription("email", "b@example.com", pendingConfirmation)},
	})
	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return f.table.GetTable().GetRowCount() == 2 })

	f.open(t)
	if got := f.message(t); !strings.Contains(got, "pending confirmation") {
		t.Errorf("message = %q, want a pending warning", got)
	}
	f.confirm(t)

	if got := uitest.AwaitError(t, f.errors); !strings.Contains(got, "pending confirmation") {
		t.Errorf("error = %q", got)
	}
}

func TestUnsubscribeView__ReportsApiError(t *testing.T) {
	var f = newUnsubscribeFixture(t, &fakeSnsApi{subscriptions: twoSubscriptions()})
	uitest.Eventually(t, f.table.serviceCtx.App, func() bool { return f.table.GetTable().GetRowCount() == 3 })
	f.api.setErr(errString("access denied"))

	f.open(t)
	f.confirm(t)

	if got := uitest.AwaitError(t, f.errors); got != "access denied" {
		t.Errorf("error = %q", got)
	}
}

func TestUnsubscribeView__NothingSelectedReportsAnError(t *testing.T) {
	var f = newUnsubscribeFixture(t, &fakeSnsApi{})
	uitest.AwaitRows(t, f.table.serviceCtx.App, f.table, []core.TableRow{subscriptionHeadings})

	f.open(t)
	if got := f.message(t); got != "No subscription selected" {
		t.Errorf("message = %q", got)
	}
	f.confirm(t)

	if got := uitest.AwaitError(t, f.errors); got != "No subscription selected" {
		t.Errorf("error = %q", got)
	}
	if slices.ContainsFunc(f.api.recorded(), func(call string) bool { return strings.HasPrefix(call, "unsubscribe") }) {
		t.Errorf("Unexpected calls %v", f.api.recorded())
	}
}

type errString string

func (e errString) Error() string { return string(e) }
