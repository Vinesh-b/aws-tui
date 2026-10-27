package core

import (
	"context"
	"errors"
	"io"
	"log"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rivo/tview"
)

// fakeSession is AWS and the clock for a session monitor. The monitor polls
// from its own goroutine while the tests change it.
type fakeSession struct {
	mu         sync.Mutex
	creds      aws.Credentials
	retrieveEr error
	identifies int
	now        time.Time
	seen       []*sessionInfo
}

func (f *fakeSession) retrieve(context.Context) (aws.Credentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.creds, f.retrieveEr
}

func (f *fakeSession) identify(context.Context) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.identifies++
	return "123456789012", "AROA:me", nil
}

func (f *fakeSession) clock() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeSession) set(change func(f *fakeSession)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(f)
}

func (f *fakeSession) record(session *sessionInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append(f.seen, session)
}

func (f *fakeSession) counts() (identifies int, seen int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.identifies, len(f.seen)
}

func (f *fakeSession) last() *sessionInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.seen) == 0 {
		return nil
	}
	return f.seen[len(f.seen)-1]
}

// newMonitorTest is a monitor that polls the fake every millisecond and runs
// its updates on its own goroutine, until the test ends
func newMonitorTest(t *testing.T) (*AppContext, *SessionMonitor, *fakeSession) {
	var ctx, cancel = context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var theme = AppTheme{}
	var appCtx = NewAppContext(ctx, tview.NewApplication(), nil, log.New(io.Discard, "", 0), &theme)
	appCtx.PollSessionAtStart = true
	var fake = &fakeSession{
		creds: aws.Credentials{AccessKeyID: "AKIA1"},
		now:   time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC),
	}

	var monitor = appCtx.Session()
	monitor.rate = time.Millisecond
	monitor.retrieve = fake.retrieve
	monitor.identify = fake.identify
	monitor.profile = func() string { return "dev" }
	monitor.now = fake.clock
	monitor.queue = func(update func()) { update() }
	return appCtx, monitor, fake
}

// waitFor polls the condition, as the monitor has no app to wait on
func waitFor(t *testing.T, description string, condition func() bool) {
	t.Helper()
	var deadline = time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(time.Millisecond) // sleep-ok: polling the monitor's goroutine, there is no app to wait on
	}
}

// pollsPass waits for the monitor to poll a few more times
func pollsPass(t *testing.T) {
	t.Helper()
	var before = time.Now()
	waitFor(t, "a few polls", func() bool { return time.Since(before) > 20*time.Millisecond })
}

func TestSessionMonitor__AllTheBarsShareOnePoller(t *testing.T) {
	var appCtx, _, fake = newMonitorTest(t)
	var before = runtime.NumGoroutine()

	for range 5 {
		appCtx.Session().Subscribe(fake.record)
	}

	if got := runtime.NumGoroutine(); got != before+1 {
		t.Errorf("goroutines = %d, want one more than the %d before", got, before)
	}
	waitFor(t, "the session for every bar", func() bool { var _, seen = fake.counts(); return seen >= 10 })
}

func TestSessionMonitor__AsksForTheIdentityOnlyWhenTheCredentialsChange(t *testing.T) {
	var appCtx, _, fake = newMonitorTest(t)
	appCtx.Session().Subscribe(fake.record)
	waitFor(t, "the first session", func() bool { return fake.last() != nil })

	pollsPass(t)
	if identifies, _ := fake.counts(); identifies != 1 {
		t.Errorf("identity asked %d times for the same credentials, want once", identifies)
	}

	fake.set(func(f *fakeSession) { f.creds.AccessKeyID = "AKIA2" })
	waitFor(t, "the identity of the new credentials", func() bool { var identifies, _ = fake.counts(); return identifies == 2 })
}

func TestSessionMonitor__TellsTheBarsOnlyWhenWhatTheyShowChanges(t *testing.T) {
	var appCtx, _, fake = newMonitorTest(t)
	fake.set(func(f *fakeSession) {
		f.creds.CanExpire = true
		f.creds.Expires = f.now.Add(2*time.Hour + 30*time.Second)
	})
	appCtx.Session().Subscribe(fake.record)
	waitFor(t, "the first session", func() bool { return fake.last() != nil })

	// Seconds go by, which the bar does not show while there are hours left
	fake.set(func(f *fakeSession) { f.now = f.now.Add(20 * time.Second) })
	pollsPass(t)
	if _, seen := fake.counts(); seen != 2 {
		t.Errorf("bar told %d times, want the loading and the first session only", seen)
	}

	fake.set(func(f *fakeSession) { f.now = f.now.Add(time.Minute) })
	waitFor(t, "the minute that went by", func() bool { var _, seen = fake.counts(); return seen == 3 })
	if got := fake.last().duration; got != time.Hour+59*time.Minute+10*time.Second {
		t.Errorf("duration = %s", got)
	}
}

func TestSessionMonitor__ShowsCredentialErrorsAndAsksAgainOnceTheyAreFixed(t *testing.T) {
	var appCtx, _, fake = newMonitorTest(t)
	fake.set(func(f *fakeSession) { f.retrieveEr = errors.New("token expired") })
	appCtx.Session().Subscribe(fake.record)

	waitFor(t, "the error", func() bool { var last = fake.last(); return last != nil && last.err == "token expired" })

	fake.set(func(f *fakeSession) { f.retrieveEr = nil })
	waitFor(t, "the session", func() bool { var last = fake.last(); return last != nil && last.accountId == "123456789012" })
}

func TestSessionMonitor__AResetClearsTheBarsAndAsksForTheIdentityAgain(t *testing.T) {
	var appCtx, _, fake = newMonitorTest(t)
	appCtx.Session().Subscribe(fake.record)
	waitFor(t, "the first session", func() bool { return fake.last() != nil })

	notifyApiReset(appCtx)

	waitFor(t, "the identity asked again", func() bool { var identifies, _ = fake.counts(); return identifies == 2 })
	waitFor(t, "the session shown again", func() bool { return fake.last() != nil })
	fake.mu.Lock()
	defer fake.mu.Unlock()
	var cleared = false
	for _, session := range fake.seen[2:] {
		cleared = cleared || session == nil
	}
	if !cleared {
		t.Error("the bars were not cleared by the reset")
	}
}

func TestSessionMonitor__AnEndedSubscriptionIsNotTold(t *testing.T) {
	var appCtx, _, fake = newMonitorTest(t)
	var other = &fakeSession{}
	var unsubscribe = appCtx.Session().Subscribe(other.record)
	appCtx.Session().Subscribe(fake.record)

	unsubscribe()
	fake.set(func(f *fakeSession) { f.creds.AccessKeyID = "AKIA2" })
	waitFor(t, "the session", func() bool { var _, seen = fake.counts(); return seen >= 2 })

	if _, seen := other.counts(); seen != 1 {
		t.Errorf("the ended subscription was told %d times, want only when it subscribed", seen)
	}
}

func TestSessionMonitor__StopsWithTheApp(t *testing.T) {
	var ctx, cancel = context.WithCancel(context.Background())
	var theme = AppTheme{}
	var appCtx = NewAppContext(ctx, tview.NewApplication(), nil, log.New(io.Discard, "", 0), &theme)
	var monitor = appCtx.Session()
	monitor.retrieve = func(context.Context) (aws.Credentials, error) { return aws.Credentials{}, nil }
	monitor.identify = func(context.Context) (string, string, error) { return "", "", nil }
	monitor.queue = func(func()) {}
	var before = runtime.NumGoroutine()
	monitor.Subscribe(func(*sessionInfo) {})

	cancel()

	waitFor(t, "the poller to stop", func() bool { return runtime.NumGoroutine() <= before })
}
