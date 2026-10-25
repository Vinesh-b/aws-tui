package core

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

const CredsPollRate = time.Second * 5

// SessionExpiryWarning is how long before the session expires its duration is
// shown as a warning.
const SessionExpiryWarning = time.Minute * 15

// sessionInfo is the state of the credentials shown on the paginator.
type sessionInfo struct {
	profile   string
	accountId string
	userId    string
	duration  time.Duration
	err       string
}

// textSegment is text drawn with a style.
type textSegment struct {
	text  string
	style tcell.Style
}

// PaginatorView is the status bar of a service. It shows the service and its
// pages as pills, which can be clicked to switch page, and the details of the
// session.
type PaginatorView struct {
	*tview.Box
	serviceName    string
	pageBar        *PillBar
	session        *sessionInfo
	onPageSelected func(name string)
	theme          AppTheme
	serviceColour  tcell.Color
	stop           chan struct{}
	stopOnce       *sync.Once
}

func CreatePaginatorView(service string, appContext *AppContext) *PaginatorView {
	var view = &PaginatorView{
		Box:         tview.NewBox(),
		serviceName: service,
		pageBar:     NewPillBar(appContext.Theme).SetGap(1),
		theme:       *appContext.Theme,
		// The colour scheme of a service is set while its views are built
		serviceColour: tview.Styles.BorderColor,
		stop:          make(chan struct{}),
		stopOnce:      &sync.Once{},
	}
	view.SetBorderPadding(0, 0, 1, 1)

	go view.pollSession(appContext)

	return view
}

// firstPollDelay is how long the session details are loading for.
func firstPollDelay(appContext *AppContext) time.Duration {
	if appContext.PollSessionAtStart {
		return 0
	}
	return CredsPollRate
}

func (inst *PaginatorView) pollSession(appContext *AppContext) {
	// wait returns false when the view has been stopped
	var wait = func(delay time.Duration) bool {
		select {
		case <-inst.stop:
			return false
		case <-time.After(delay):
			return true
		}
	}
	var sleep = func() bool { return wait(CredsPollRate) }

	var app = appContext.App
	var setSession = func(session sessionInfo) {
		app.QueueUpdateDraw(func() { inst.session = &session })
	}

	if !wait(firstPollDelay(appContext)) {
		return
	}

	for first := true; ; first = false {
		if !first && !sleep() {
			return
		}

		var logger = appContext.Logger
		var apiClients = appContext.GetApiClients()

		var creds, err = apiClients.Config.Credentials.Retrieve(appContext.Ctx)
		if err != nil {
			logger.Print(err.Error())
			setSession(sessionInfo{err: err.Error()})
			continue
		}

		identity, err := apiClients.Sts.GetCallerIdentity(
			appContext.Ctx,
			&sts.GetCallerIdentityInput{},
		)
		if err != nil {
			logger.Print(err.Error())
			setSession(sessionInfo{err: err.Error()})
			continue
		}
		var profileName = os.Getenv("AWS_PROFILE")
		if len(profileName) == 0 {
			profileName = apiClients.Profile
			if len(profileName) == 0 {
				profileName = "unset"
			}
		}

		var session = sessionInfo{
			profile:   profileName,
			accountId: aws.ToString(identity.Account),
			userId:    aws.ToString(identity.UserId),
		}

		if creds.CanExpire == false {
			session.duration = math.MinInt64
			setSession(session)
			continue
		}

		for time.Now().Before(creds.Expires) {
			session.duration = creds.Expires.Sub(time.Now()).Truncate(time.Second)
			setSession(session)
			if appContext.GetApiClients().Profile != profileName {
				//Profile updated by user
				break
			}
			if !sleep() {
				return
			}
		}
		session.duration = 0
		setSession(session)
	}
}

// Stop ends the polling of the session details.
func (inst *PaginatorView) Stop() {
	inst.stopOnce.Do(func() { close(inst.stop) })
}

func (inst *PaginatorView) SetServiceName(text string) {
	inst.serviceName = text
}

// SetPages sets the pages shown, in order.
func (inst *PaginatorView) SetPages(names []string) {
	inst.pageBar.SetPills(names)
}

// SetCurrentPage sets the index of the page that is shown.
func (inst *PaginatorView) SetCurrentPage(index int) {
	inst.pageBar.SetCurrent(index)
}

// SetPageSelectedFunc sets the handler called with the name of a page when its
// pill is clicked.
func (inst *PaginatorView) SetPageSelectedFunc(handler func(name string)) {
	inst.onPageSelected = handler
}

// pageBarOffset is where the page pills start, after the pill of the service,
// from the left of the view.
func (inst *PaginatorView) pageBarOffset() int {
	return pillWidth(inst.serviceName) + 1
}

// pageKeysHint is the keys that change page, shown after the page pills.
func pageKeysHint() string {
	return fmt.Sprintf("Alt+%c %c", APP_KEY_BINDINGS.PageBack, APP_KEY_BINDINGS.PageForward)
}

// formatSessionDuration shows hours and minutes, or minutes, and seconds only
// in the last minute, so the duration does not tick every poll.
func formatSessionDuration(duration time.Duration) string {
	switch {
	case duration >= time.Hour:
		var minutes = duration.Truncate(time.Minute)
		return fmt.Sprintf("%dh%02dm", int(minutes.Hours()), int(minutes.Minutes())%60)
	case duration >= time.Minute:
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	default:
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
}

// sessionSegments is the text of the session details, or that they are loading
// until the first poll.
func (inst *PaginatorView) sessionSegments() []textSegment {
	var theme = inst.theme
	var background = tcell.StyleDefault.Background(theme.BackgroundColour)
	var label = background.Foreground(theme.PlaceholderTextColour)
	var value = background.Foreground(theme.PrimaryTextColour)

	var session = inst.session
	if session == nil {
		return []textSegment{{"Loading session…", label}}
	}

	if len(session.err) > 0 {
		return []textSegment{
			{"⚠ Credentials Error: " + session.err, background.Foreground(theme.ErrorTextColour)},
		}
	}

	var durationText string
	var durationStyle = value
	switch {
	case session.duration == math.MinInt64:
		durationText = "no expiry"
	case session.duration <= 0:
		durationText = "Expired"
		durationStyle = background.Foreground(theme.ErrorTextColour)
	default:
		durationText = formatSessionDuration(session.duration)
		if session.duration <= SessionExpiryWarning {
			durationStyle = background.Foreground(theme.TertiaryTextColour)
		}
	}

	var separator = textSegment{"  │  ", label}
	return []textSegment{
		{"Profile ", label}, {session.profile, value}, separator,
		{"Account ", label}, {session.accountId, value}, separator,
		{"Session ", label}, {durationText, durationStyle}, separator,
		{"User ", label}, {session.userId, value},
	}
}

func (inst *PaginatorView) Draw(screen tcell.Screen) {
	inst.Box.DrawForSubclass(screen, inst)

	var x, y, width, height = inst.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}
	var maxX = x + width
	var theme = inst.theme

	var serviceStyle = tcell.StyleDefault.
		Background(inst.serviceColour).
		Foreground(theme.SecondaryTextColour)
	drawPill(screen, x, y, maxX, inst.serviceName, serviceStyle, theme.BackgroundColour)
	var pos = inst.pageBar.Draw(screen, min(x+inst.pageBarOffset(), maxX), y, maxX)

	if len(inst.pageBar.GetPills()) > 1 {
		var hintStyle = tcell.StyleDefault.
			Background(theme.BackgroundColour).
			Foreground(theme.PlaceholderTextColour)
		pos = drawText(screen, min(pos+2, maxX), y, maxX, pageKeysHint(), hintStyle)
	}

	var segments = inst.sessionSegments()
	var sessionWidth = 0
	for _, segment := range segments {
		sessionWidth += uniseg.StringWidth(segment.text)
	}

	// The session details are right aligned, cut off when they meet the pills
	var sessionX = max(maxX-sessionWidth, pos+2)
	for _, segment := range segments {
		sessionX = drawText(screen, sessionX, y, maxX, segment.text, segment.style)
	}
}

func (inst *PaginatorView) MouseHandler() func(
	action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive),
) (consumed bool, capture tview.Primitive) {
	return inst.WrapMouseHandler(func(
		action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive),
	) (consumed bool, capture tview.Primitive) {
		var x, y = event.Position()
		if !inst.InRect(x, y) {
			return false, nil
		}

		// Clicks are consumed so the bar never takes the focus
		if action != tview.MouseLeftClick || inst.onPageSelected == nil {
			return true, nil
		}

		var innerX, _, _, _ = inst.GetInnerRect()
		if idx, ok := inst.pageBar.PillAt(x - innerX - inst.pageBarOffset()); ok {
			inst.onPageSelected(inst.pageBar.GetPills()[idx])
		}
		return true, nil
	})
}
