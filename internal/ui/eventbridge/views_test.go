package eventbridge

import (
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
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

func TestDetailsPageView__SelectingBusLoadsItsRulesAndTargets(t *testing.T) {
	var api = &fakeEventBridgeApi{
		buses:   []types.EventBus{newEventBus("orders-bus")},
		rules:   []types.Rule{newRule("order-created")},
		targets: []types.Target{newTarget("handler")},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var view = NewEventBridgeDetailsPageView(
		NewEventBusListTable(serviceCtx),
		NewEventBusDetailsTable(serviceCtx),
		NewRulesTabView(NewRulesTable(serviceCtx), serviceCtx),
		serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)

	view.EventBusListTable.RefreshEventBuss(true)

	uitest.AwaitRows(t, serviceCtx.App, view.RulesTabView.RulesTable, []core.TableRow{
		rulesHeadings,
		{"order-created", "ENABLED", ""},
	})
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return strings.Contains(view.RulesTabView.JsonView.TextView.GetText(), "function:handler")
	})
}
