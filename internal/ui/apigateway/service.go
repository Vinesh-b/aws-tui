package apigateway

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.API_GATEWAY,
	Icon:        "󰘦",
	Description: "REST APIs and their details",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewApiGatewayHomeView(appCtx, awsapi.NewApiGatewayApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
