package eventbridge

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

func TestHomeView__ResetKeyLoadsBusesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewEventBridgeHomeView(appCtx, &fakeEventBridgeApi{buses: []types.EventBus{newEventBus("orders-bus")}})
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-bus")
}
