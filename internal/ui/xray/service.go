package xray

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.XRAY,
	Icon:        "\U000F0437",
	Description: "Traces, trace analytics, service map and sampling rules",
	Shortcut:    core.NoShortcut,
	Aliases:     []string{"xray"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewXRayHomeView(appCtx, awsapi.NewXRayApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
