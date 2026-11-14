package ec2

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.EC2,
	Icon:        "",
	Description: "Instances and their details",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewEc2HomeView(appCtx, awsapi.NewEc2Api(appCtx.GetApiClients(), appCtx.Logger))
	},
}
