package route53

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.ROUTE53,
	Icon:        "󰇧",
	Description: "Hosted zones, records and health checks",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewRoute53HomeView(appCtx, awsapi.NewRoute53Api(appCtx.GetApiClients(), appCtx.Logger))
	},
}
