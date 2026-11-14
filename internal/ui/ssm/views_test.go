package ssm

import (
	"errors"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/ssm/ssmfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestHomeView__ResetKeyLoadsParametersFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewSystemManagerHomeView(appCtx, ssmfake.New(ssmfake.WithParameters([]types.Parameter{newParameter("/orders/db", 1)})))
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "/orders/db")
}

// encryptedValue is what is listed for a secure value, which is never shown
const encryptedValue = "AQICAHencrypted"

// showSecureParameterPage shows the page with the secure parameter
// /app/db/pass listed and selected
func showSecureParameterPage(t *testing.T, api *ssmfake.Api) (*tview.Application, tcell.SimulationScreen) {
	t.Helper()
	api.SetParameters([]types.Parameter{newSecretParameter("/app/db/pass", encryptedValue)})
	var appCtx = uitest.NewAppContext()
	var page = NewSystemManagerHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "/app/db/pass")
	uitest.AwaitScreenText(t, appCtx.App, screen, "Secure value, not decrypted")
	return appCtx.App, screen
}

func TestHomeView__SecureValueStaysHiddenUntilEnterIsPressed(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithValue("/app/db/pass", 1, "hunter2"))
	var app, screen = showSecureParameterPage(t, api)

	uitest.Consistently(t, app, 200*time.Millisecond, func() bool {
		var text = uitest.ScreenText(screen)
		return !strings.Contains(text, "hunter2") && !strings.Contains(text, encryptedValue) &&
			len(api.ValueRequests()) == 0
	}, "nothing is decrypted or shown before Enter")

	uitest.PressKey(t, app, tcell.KeyEnter)

	uitest.AwaitScreenText(t, app, screen, "hunter2")
	if requests := api.ValueRequests(); len(requests) != 1 ||
		requests[0] != (ssmfake.ValueRequest{Name: "/app/db/pass", Version: 1}) {
		t.Errorf("value requests = %v, want version 1 of /app/db/pass once", requests)
	}
}

func TestHomeView__PlainValueIsShownWithoutDecrypting(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithParameters([]types.Parameter{newPlainParameter("/app/db/host", "db.internal")}))
	var appCtx = uitest.NewAppContext()
	var page = NewSystemManagerHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "db.internal")
	if requests := api.ValueRequests(); len(requests) != 0 {
		t.Errorf("value requests = %v, want none", requests)
	}
}

func TestHomeView__DecryptErrorIsReported(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithValueErr(errors.New("access denied to the key")))
	var app, screen = showSecureParameterPage(t, api)

	uitest.PressKey(t, app, tcell.KeyEnter)

	uitest.AwaitScreenText(t, app, screen, "access denied to the key")
}

// The first Enter decrypts the value, and the next opens the history as it
// does for the other parameters, where the versions are not decrypted either
func TestHomeView__SecureVersionStaysHiddenUntilEnterIsPressedInTheHistory(t *testing.T) {
	var api = ssmfake.New(
		ssmfake.WithValue("/app/db/pass", 1, "hunter2"),
		ssmfake.WithValue("/app/db/pass", 7, "hunter7"),
		ssmfake.WithHistory([]types.ParameterHistory{newSecureHistory("/app/db/pass", 7, encryptedValue)}),
	)
	var app, screen = showSecureParameterPage(t, api)
	uitest.PressKey(t, app, tcell.KeyEnter)
	uitest.AwaitScreenText(t, app, screen, "hunter2")

	uitest.PressKey(t, app, tcell.KeyEnter)
	uitest.AwaitScreenText(t, app, screen, secureValueCell)
	uitest.AwaitScreenText(t, app, screen, "Secure value, not decrypted")
	uitest.Consistently(t, app, 200*time.Millisecond, func() bool {
		var text = uitest.ScreenText(screen)
		return !strings.Contains(text, "hunter7") && !strings.Contains(text, encryptedValue) &&
			len(api.ValueRequests()) == 1
	}, "the version is not decrypted or shown before Enter")

	uitest.PressKey(t, app, tcell.KeyEnter)

	uitest.AwaitScreenText(t, app, screen, "hunter7")
	var want = []ssmfake.ValueRequest{{Name: "/app/db/pass", Version: 1}, {Name: "/app/db/pass", Version: 7}}
	if requests := api.ValueRequests(); len(requests) != 2 || requests[0] != want[0] || requests[1] != want[1] {
		t.Errorf("value requests = %v, want %v", requests, want)
	}
}
