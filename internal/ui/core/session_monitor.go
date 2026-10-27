package core

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// SessionMonitor looks up the session details every status bar of the app
// shows, once for all of them. The credentials are checked each poll, which
// is cheap as the SDK keeps them, and the identity is only asked of STS when
// they change. The bars are only told when what they show changes.
type SessionMonitor struct {
	appCtx *AppContext
	rate   time.Duration
	// retrieve, identify, profile and now are AWS and the clock, which tests
	// replace before the first bar subscribes
	retrieve func(ctx context.Context) (aws.Credentials, error)
	identify func(ctx context.Context) (account string, user string, err error)
	profile  func() string
	now      func() time.Time
	// queue runs a change of the session on the UI goroutine
	queue func(update func())

	mu          sync.Mutex
	subscribers map[int]func(session *sessionInfo)
	nextId      int
	current     *sessionInfo

	// generation counts the api client resets, so a poll made with the old
	// clients is not shown once they are replaced
	generation atomic.Int64
	// refresh wakes the poller when the api clients are reset
	refresh   chan struct{}
	startOnce sync.Once
}

func newSessionMonitor(appCtx *AppContext) *SessionMonitor {
	var monitor = &SessionMonitor{
		appCtx: appCtx,
		rate:   CredsPollRate,
		retrieve: func(ctx context.Context) (aws.Credentials, error) {
			return appCtx.GetApiClients().Config.Credentials.Retrieve(ctx)
		},
		identify: func(ctx context.Context) (string, string, error) {
			var identity, err = appCtx.GetApiClients().Sts.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				return "", "", err
			}
			return aws.ToString(identity.Account), aws.ToString(identity.UserId), nil
		},
		profile:     func() string { return sessionProfileName(appCtx.GetApiClients().Profile) },
		now:         time.Now,
		queue:       func(update func()) { appCtx.App.QueueUpdateDraw(update) },
		subscribers: map[int]func(*sessionInfo){},
		refresh:     make(chan struct{}, 1),
	}

	// Called on the UI goroutine, so the bars can be cleared directly
	appCtx.OnApiClientsReset(func() {
		monitor.generation.Add(1)
		monitor.set(nil)
		select {
		case monitor.refresh <- struct{}{}:
		default:
		}
	})
	return monitor
}

// Session is the monitor of the session, which starts polling when the first
// status bar subscribes
func (inst *AppContext) Session() *SessionMonitor {
	inst.sessionOnce.Do(func() { inst.session = newSessionMonitor(inst) })
	return inst.session
}

// Subscribe calls the handler with the session as it is, and then each time
// what is shown of it changes, on the UI goroutine. The session is nil while
// it is loading. It returns a func that ends the subscription.
func (inst *SessionMonitor) Subscribe(handler func(session *sessionInfo)) (unsubscribe func()) {
	inst.mu.Lock()
	var id = inst.nextId
	inst.nextId++
	inst.subscribers[id] = handler
	var current = inst.current
	inst.mu.Unlock()

	handler(current)
	inst.startOnce.Do(func() { go inst.poll() })

	return func() {
		inst.mu.Lock()
		defer inst.mu.Unlock()
		delete(inst.subscribers, id)
	}
}

// set keeps the session and tells the subscribers
func (inst *SessionMonitor) set(session *sessionInfo) {
	inst.mu.Lock()
	inst.current = session
	var handlers = make([]func(*sessionInfo), 0, len(inst.subscribers))
	for _, handler := range inst.subscribers {
		handlers = append(handlers, handler)
	}
	inst.mu.Unlock()

	for _, handler := range handlers {
		handler(session)
	}
}

// firstPollDelay is how long the session details are loading for.
func firstPollDelay(appContext *AppContext) time.Duration {
	if appContext.PollSessionAtStart {
		return 0
	}
	return CredsPollRate
}

// poll looks up the session until the app stops
func (inst *SessionMonitor) poll() {
	var ctx = inst.appCtx.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	var delay = firstPollDelay(inst.appCtx)
	if delay > 0 {
		delay = inst.rate
	}

	// What was last shown, and the credentials the identity was asked with
	var shown string
	var accessKey string
	var account, user string

	for {
		select {
		case <-ctx.Done():
			return
		case <-inst.refresh:
			shown, accessKey = "", ""
		case <-time.After(delay):
		}
		delay = inst.rate

		var generation = inst.generation.Load()
		var session sessionInfo
		var creds, err = inst.retrieve(ctx)
		if err == nil && creds.AccessKeyID != accessKey {
			account, user, err = inst.identify(ctx)
			if err == nil {
				accessKey = creds.AccessKeyID
			}
		}

		switch {
		case err != nil:
			accessKey = ""
			session = sessionInfo{err: err.Error()}
		case !creds.CanExpire:
			session = sessionInfo{profile: inst.profile(), accountId: account, userId: user, duration: math.MinInt64}
		default:
			session = sessionInfo{
				profile:   inst.profile(),
				accountId: account,
				userId:    user,
				duration:  max(creds.Expires.Sub(inst.now()).Truncate(time.Second), 0),
			}
		}

		if text := session.shown(); text != shown && generation == inst.generation.Load() {
			shown = text
			if len(session.err) > 0 {
				inst.appCtx.Logger.Print(session.err)
			}
			inst.queue(func() {
				if generation == inst.generation.Load() {
					inst.set(&session)
				}
			})
		}
	}
}

// shown is what a status bar shows of the session, so the bars are only
// drawn again when it changes
func (session sessionInfo) shown() string {
	var duration string
	switch {
	case session.duration == math.MinInt64:
		duration = "no expiry"
	case session.duration <= 0:
		duration = "expired"
	case session.duration <= SessionExpiryWarning:
		duration = "warn " + formatSessionDuration(session.duration)
	default:
		duration = formatSessionDuration(session.duration)
	}
	return session.profile + "|" + session.accountId + "|" + session.userId + "|" + session.err + "|" + duration
}
