package cloudwatch

import (
	"testing"

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

func TestLogsInsightsHomeView__ResetKeyLoadsLogGroupsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeCloudWatchLogsApi{groups: []types.LogGroup{newLogGroup("/aws/lambda/orders")}}
	var screen = uitest.ShowPage(t, appCtx.App, NewLogsInsightsHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "/aws/lambda/orders")
}
