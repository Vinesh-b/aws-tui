package cloudformation

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func TestHomeView__ResetKeyLoadsStacksFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewStacksHomeView(appCtx, &fakeCloudFormationApi{stacks: []types.StackSummary{newStack("orders-stack", types.StackStatusCreateComplete)}})
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-stack")
}
