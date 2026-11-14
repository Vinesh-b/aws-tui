package sns

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.SNS,
	Icon:        "󰂚",
	Description: "Topics and their attributes",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewSnsHomeView(appCtx, awsapi.NewSnsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
