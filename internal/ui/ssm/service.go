package ssm

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.SYSTEMS_MANAGER,
	Icon:        "󰙵",
	Description: "Application parameters",
	Shortcut:    '6',
	Aliases:     []string{"ssm"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewSystemManagerHomeView(appCtx, awsapi.NewSystemsManagerApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
