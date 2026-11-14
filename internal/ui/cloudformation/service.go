package cloudformation

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.CLOUDFORMATION,
	Icon:        "",
	Description: "Cloud formation stacks",
	Shortcut:    core.NoShortcut,
	Aliases:     []string{"cfn"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewStacksHomeView(appCtx, awsapi.NewCloudFormationApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
