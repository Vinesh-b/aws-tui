package cloudwatch

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestAlarmsHomeView__ResetKeyLoadsAlarmsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeCloudWatchAlarmsApi{alarms: []cwTypes.MetricAlarm{newAlarm("orders-alarm", cwTypes.StateValueOk)}}
	var screen = uitest.ShowPage(t, appCtx.App, NewAlarmsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-alarm")
}

func TestMetricsHomeView__ResetKeyLoadsMetricsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeCloudWatchMetricsApi{metrics: []cwTypes.Metric{newMetric("AWS/Lambda", "orders-invocations")}}
	var screen = uitest.ShowPage(t, appCtx.App, NewMetricsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "AWS/Lambda")
}

func TestLogsHomeView__ResetKeyLoadsLogGroupsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeCloudWatchLogsApi{groups: []types.LogGroup{newLogGroup("/aws/lambda/orders")}}
	var screen = uitest.ShowPage(t, appCtx.App, NewLogsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "/aws/lambda/orders")
}

// sendRune sends the key through the app as it would be pressed, which runs
// the key handlers of the pages and redraws the screen
func sendRune(t *testing.T, appCtx *core.AppContext, r rune, mod tcell.ModMask) {
	t.Helper()
	appCtx.App.QueueEvent(tcell.NewEventKey(tcell.KeyRune, r, mod))
	uitest.OnUI(t, appCtx.App, func() {})
}

// showLogGroups shows the logs on the groups page with the groups loaded
func showLogGroups(t *testing.T, appCtx *core.AppContext, names ...string) tcell.SimulationScreen {
	t.Helper()
	var groups []types.LogGroup
	for _, name := range names {
		groups = append(groups, newLogGroup(name))
	}
	var screen = uitest.ShowPage(t, appCtx.App, NewLogsHomeView(appCtx, &fakeCloudWatchLogsApi{groups: groups}))
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, names[0])
	return screen
}

func TestLogsHomeView__MarkKeyMarksTheLogGroup(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var screen = showLogGroups(t, appCtx, "/aws/lambda/orders")

	sendRune(t, appCtx, logGroupMarkRune, tcell.ModNone)

	uitest.AwaitScreenText(t, appCtx.App, screen, "❬1 marked❭")
	uitest.AwaitScreenText(t, appCtx.App, screen, logGroupMarker+" /aws/lambda/orders")
}

func TestLogsHomeView__QueryKeyWithoutMarksQueriesTheGroupItIsOn(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var screen = showLogGroups(t, appCtx, "/aws/lambda/orders")

	sendRune(t, appCtx, core.APP_KEY_BINDINGS.TableQuery, tcell.ModNone)

	uitest.AwaitScreenText(t, appCtx.App, screen, "Saved Queries")
	// The groups page is the next one after Insights
	sendRune(t, appCtx, core.APP_KEY_BINDINGS.PageForward, tcell.ModAlt)
	uitest.AwaitScreenText(t, appCtx.App, screen, "❬1 marked❭")
}
