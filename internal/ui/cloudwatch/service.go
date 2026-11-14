package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// LogsService adds the service to the app
var LogsService = core.Service{
	Id:          core.CLOUDWATCH_LOGS,
	Icon:        "",
	Description: "Log groups, streams and Insights queries",
	Shortcut:    '7',
	Aliases:     []string{"logs"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewLogsHomeView(appCtx, awsapi.NewCloudWatchLogsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}

// AlarmsService adds the service to the app
var AlarmsService = core.Service{
	Id:          core.CLOUDWATCH_ALARMS,
	Icon:        "󰞏",
	Description: "Metric alarms",
	Shortcut:    core.NoShortcut,
	Aliases:     []string{"alarms"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewAlarmsHomeView(appCtx, awsapi.NewCloudWatchAlarmsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}

// MetricsService adds the service to the app
var MetricsService = core.Service{
	Id:          core.CLOUDWATCH_METRICS,
	Icon:        "",
	Description: "View metrics",
	Shortcut:    core.NoShortcut,
	Aliases:     []string{"metrics"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewMetricsHomeView(appCtx, awsapi.NewCloudWatchMetricsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
