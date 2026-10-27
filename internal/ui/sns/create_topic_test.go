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

func TestValidTopicName(t *testing.T) {
	for name, tc := range map[string]struct {
		input   string
		fifo    bool
		want    string
		wantErr string
	}{
		"standard":                 {"orders", false, "orders", ""},
		"trimmed":                  {"  orders_1-a ", false, "orders_1-a", ""},
		"fifo suffix is added":     {"orders", true, "orders.fifo", ""},
		"fifo suffix is kept":      {"orders.fifo", true, "orders.fifo", ""},
		"empty":                    {"  ", false, "", "Name is empty"},
		"standard with suffix":     {"orders.fifo", false, "", "can't end with .fifo"},
		"spaces":                   {"my topic", false, "", "letters, digits"},
		"dots":                     {"my.topic", true, "", "letters, digits"},
		"too long":                 {strings.Repeat("a", 257), false, "", "at most 256"},
		"fifo too long with .fifo": {strings.Repeat("a", 252), true, "", "at most 256"},
	} {
		t.Run(name, func(t *testing.T) {
			var got, err = validTopicName(tc.input, tc.fifo)

			if tc.wantErr == "" && (err != nil || got != tc.want) {
				t.Errorf("validTopicName = %q, %v, want %q", got, err, tc.want)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("validTopicName error = %v, want one containing %q", err, tc.wantErr)
			}
		})
	}
}

type createTopicFixture struct {
	table *TopicListTable
	api   *snsfake.Api
}

func newCreateTopicFixture(t *testing.T, api *snsfake.Api) *createTopicFixture {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SnsApi](api)
	var table = NewTopicListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() {
		serviceCtx.App.SetRoot(table, true)
		serviceCtx.App.SetFocus(table)
	})
	return &createTopicFixture{table: table, api: api}
}

func (f *createTopicFixture) app() *tview.Application { return f.table.serviceCtx.App }

func (f *createTopicFixture) view() *CreateTopicView { return f.table.CreateView.Input }

func (f *createTopicFixture) press(t *testing.T, key tcell.Key, r rune) {
	t.Helper()
	// Sent through the whole focus chain, as the keys are handled by parent views
	f.app().QueueEvent(tcell.NewEventKey(key, r, tcell.ModNone))
	uitest.OnUI(t, f.app(), func() {})
}

func (f *createTopicFixture) open(t *testing.T) {
	t.Helper()
	f.press(t, tcell.KeyRune, createTopicKey)
	uitest.Eventually(t, f.app(), func() bool { return f.table.IsAnOverlayVisible() })
}

// create moves focus from the name field to the Create button and presses it
func (f *createTopicFixture) create(t *testing.T) {
	t.Helper()
	for range 3 {
		f.press(t, tcell.KeyDown, 0)
	}
	f.press(t, tcell.KeyEnter, 0)
}

// statusNow reads the status and must be called on the UI goroutine
func (f *createTopicFixture) statusNow() string { return f.view().message.GetText(true) }

func (f *createTopicFixture) status(t *testing.T) string {
	t.Helper()
	var status string
	uitest.OnUI(t, f.app(), func() { status = f.statusNow() })
	return status
}

func (f *createTopicFixture) createCalls() []string {
	return f.api.CallsWithPrefix("create ")
}

func TestCreateTopic__CreatesAStandardTopicAndReloads(t *testing.T) {
	var f = newCreateTopicFixture(t, snsfake.New())
	f.open(t)
	uitest.OnUI(t, f.app(), func() { f.view().nameInput.SetText(" orders ") })

	f.create(t)

	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{topicHeadings, {"orders", "Standard"}})
	uitest.Eventually(t, f.app(), func() bool {
		return f.statusNow() == "Topic created: "+topicArn("orders")
	})
	if want := []string{"create orders fifo=false dedup=false"}; !slices.Equal(f.createCalls(), want) {
		t.Errorf("Unexpected calls %v", f.createCalls())
	}
}

func TestCreateTopic__CreatesAFifoTopicWithContentBasedDeduplication(t *testing.T) {
	var f = newCreateTopicFixture(t, snsfake.New())
	f.open(t)
	uitest.OnUI(t, f.app(), func() {
		f.view().nameInput.SetText("orders")
		f.view().typeDropDown.SetCurrentOption(1)
		f.view().dedupDropDown.SetCurrentOption(1)
	})

	f.create(t)

	uitest.AwaitRows(t, f.app(), f.table, []core.TableRow{topicHeadings, {"orders.fifo", "FIFO"}})
	if want := []string{"create orders.fifo fifo=true dedup=true"}; !slices.Equal(f.createCalls(), want) {
		t.Errorf("Unexpected calls %v", f.createCalls())
	}
}

func TestCreateTopic__InvalidNameIsReportedWithoutCallingTheApi(t *testing.T) {
	var f = newCreateTopicFixture(t, snsfake.New())
	f.open(t)
	uitest.OnUI(t, f.app(), func() { f.view().nameInput.SetText("my topic") })

	f.create(t)

	if got := f.status(t); !strings.Contains(got, "letters, digits") {
		t.Errorf("status = %q", got)
	}
	if calls := f.createCalls(); len(calls) != 0 {
		t.Errorf("Unexpected calls %v", calls)
	}
}

func TestCreateTopic__ReportsApiError(t *testing.T) {
	var f = newCreateTopicFixture(t, snsfake.New(snsfake.WithErr(errString("access denied"))))
	f.open(t)
	uitest.OnUI(t, f.app(), func() { f.view().nameInput.SetText("orders") })

	f.create(t)

	uitest.Eventually(t, f.app(), func() bool { return f.statusNow() == "access denied" })
}

func TestCreateTopic__StartsCleanEachTimeItIsShown(t *testing.T) {
	var f = newCreateTopicFixture(t, snsfake.New())
	uitest.OnUI(t, f.app(), func() {
		f.view().nameInput.SetText("left over")
		f.view().typeDropDown.SetCurrentOption(1)
		f.view().dedupDropDown.SetCurrentOption(1)
		f.view().SetStatusMessage("old")
	})

	f.open(t)

	uitest.OnUI(t, f.app(), func() {
		if f.view().GetName() != "" || f.view().IsFifo() || f.view().ContentBasedDeduplication() {
			t.Error("expected the form to be reset")
		}
	})
	if got := f.status(t); got != "" {
		t.Errorf("status = %q, want it cleared", got)
	}
}
