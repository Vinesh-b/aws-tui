package vpc

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.VPC,
	Icon:        "󱇱",
	Description: "VPCs, endpoints, subnets, network ACLs...",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewVpcHomeView(appCtx, awsapi.NewEc2Api(appCtx.GetApiClients(), appCtx.Logger))
	},
}
