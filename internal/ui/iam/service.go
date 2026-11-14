package iam

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.IAM,
	Icon:        "󰌆",
	Description: "Roles, users and policies",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewIamHomeView(appCtx, awsapi.NewIamApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
