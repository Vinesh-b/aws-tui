package sns

import (
	"aws-tui/internal/ui/sns/snsfake"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type deleteTopicFixture struct {
	page   *SnsDetailsPageView
	table  *TopicListTable
	api    *snsfake.Api
	errors <-chan string
}

func newDeleteTopicFixture(t *testing.T, api *snsfake.Api) *deleteTopicFixture {
	t.Helper()
	api.SetAttributes(newTopicAttributes())
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var page = NewSnsDetailsPageView(NewTopicListTable(serviceCtx), NewTopicDetailsTable(serviceCtx), serviceCtx)
	var callback, received = uitest.CaptureErrors()
	page.TopicListTable.ErrorMessageCallback = callback
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(page.TopicListTable, true)
		serviceCtx.App.SetFocus(page.TopicListTable)
	})
	page.TopicListTable.RefreshTopics(true)
	return &deleteTopicFixture{page: page, table: page.TopicListTable, api: api, errors: received}
}

func (f *deleteTopicFixture) app() *tview.Application { return f.table.serviceCtx.App }

func (f *deleteTopicFixture) press(t *testing.T, event *tcell.EventKey) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(event)
	uitest.OnUI(t, f.app(), func() {})
}

func (f *deleteTopicFixture) open(t *testing.T) {
	t.Helper()
	f.press(t, tcell.NewEventKey(tcell.KeyRune, deleteTopicKey, tcell.ModNone))
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

func (f *deleteTopicFixture) confirm(t *testing.T) {
	t.Helper()
	f.focusNext(t)
	f.press(t, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
}

func (f *deleteTopicFixture) message(t *testing.T) string {
	t.Helper()
	var message string
	uitest.OnUI(t, f.app(), func() { message = f.table.DeleteView.Input.GetMessage() })
	return message
}

func (f *deleteTopicFixture) deleteCalls() []string {
	return f.api.CallsWithPrefix("delete ")
}

func TestDeleteTopic__ConfirmingDeletesTheSelectedTopicAndReloads(t *testing.T) {
	var f = newDeleteTopicFixture(t, snsfake.New(snsfake.WithTopics([]string{topicArn("billing"), topicArn("orders")})))
	uitest.Eventually(t, f.app(), func() bool { return f.table.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, f.app(), func() { f.table.Select(2, 0) })

	f.open(t)
	if got := f.message(t); !strings.Contains(got, "Delete topic orders and all of its subscriptions?") {
		t.Errorf("message = %q", got)
	}
	f.confirm(t)

	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{topicHeadings, {"billing", "Standard"}})
	if want := []string{"delete " + topicArn("orders")}; !slices.Equal(f.deleteCalls(), want) {
		t.Errorf("Unexpected calls %v", f.deleteCalls())
	}
	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
}

func TestDeleteTopic__CancellingDeletesNothing(t *testing.T) {
	var f = newDeleteTopicFixture(t, snsfake.New(snsfake.WithTopics([]string{topicArn("orders")})))
	uitest.Eventually(t, f.app(), func() bool { return f.table.GetTable().GetRowCount() == 2 })

	f.open(t)
	// Focus starts on Cancel, so Enter alone must not delete
	f.press(t, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	uitest.Eventually(t, f.app(), func() bool { return !f.table.IsAnOverlayVisible() })
	if calls := f.deleteCalls(); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

func TestDeleteTopic__ReportsApiError(t *testing.T) {
	var f = newDeleteTopicFixture(t, snsfake.New(snsfake.WithTopics([]string{topicArn("orders")})))
	uitest.Eventually(t, f.app(), func() bool { return f.table.GetTable().GetRowCount() == 2 })
	f.api.SetErr(errString("access denied"))

	f.open(t)
	f.confirm(t)

	if got := uitest.AwaitError(t, f.errors); got != "access denied" {
		t.Errorf("error = %q", got)
	}
}

func TestDeleteTopic__NothingSelectedReportsAnError(t *testing.T) {
	var f = newDeleteTopicFixture(t, snsfake.New())
	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{topicHeadings})

	f.open(t)
	if got := f.message(t); got != "No topic selected" {
		t.Errorf("message = %q", got)
	}
	f.confirm(t)

	if got := uitest.AwaitError(t, f.errors); got != "No topic selected" {
		t.Errorf("error = %q", got)
	}
	if calls := f.deleteCalls(); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

func TestDeleteTopic__DeletingTheOnlyTopicEmptiesTheOtherViews(t *testing.T) {
	var f = newDeleteTopicFixture(t, snsfake.New(snsfake.WithTopics([]string{topicArn("orders")}), snsfake.WithSubscriptions(twoSubscriptions())))
	uitest.Eventually(t, f.app(), func() bool {
		return f.page.TopicDetailsTable.GetCellText(0, 1) == "orders" &&
			f.page.SubscriptionsTable.GetTable().GetRowCount() == 3 &&
			f.page.PublishView.selectedTopicArn == topicArn("orders")
	})

	f.open(t)
	f.confirm(t)

	uitest.Eventually(t, f.app(), func() bool {
		return f.page.TopicDetailsTable.GetCellText(0, 1) == "" &&
			f.page.SubscriptionsTable.GetTable().GetRowCount() == 1 &&
			f.page.PublishView.selectedTopicArn == "" &&
			f.table.GetSelectedTopic() == ""
	})
}

func TestDeleteTopic__DeletingOneOfManyShowsTheNextTopic(t *testing.T) {
	var f = newDeleteTopicFixture(t, snsfake.New(snsfake.WithTopics([]string{topicArn("billing"), topicArn("orders")})))
	uitest.Eventually(t, f.app(), func() bool { return f.page.PublishView.selectedTopicArn == topicArn("billing") })

	f.open(t)
	f.confirm(t)

	uitest.Eventually(t, f.app(), func() bool {
		return f.page.PublishView.selectedTopicArn == topicArn("orders") &&
			f.page.TopicDetailsTable.GetCellText(0, 1) == "orders"
	})
}

// focusNext moves focus from Cancel to the confirm button of the open overlay
func (f *deleteTopicFixture) focusNext(t *testing.T) {
	t.Helper()
	uitest.OnUI(t, f.app(), func() { f.table.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusRight) })
}
