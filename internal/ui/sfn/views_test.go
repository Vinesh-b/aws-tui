package sfn

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func TestHomeView__ResetKeyLoadsStateMachinesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewStepFunctionsHomeView(appCtx, &fakeStateMachineApi{machines: []types.StateMachineListItem{newStateMachine("orders-flow", types.StateMachineTypeStandard)}}, nil)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-flow")
}
