package eventbridge

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.EVENTBRIDGE,
	Icon:        "󰘘",
	Description: "Event buses, rules, schedules...",
	Shortcut:    '8',
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewEventBridgeHomeView(
			appCtx, awsapi.NewEventBridgeApi(appCtx.GetApiClients(), appCtx.Logger), awsapi.NewSchedulerApi(appCtx.GetApiClients(), appCtx.Logger),
		)
	},
}
