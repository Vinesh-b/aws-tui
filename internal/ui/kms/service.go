package kms

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.KMS,
	Icon:        "󰯄",
	Description: "Encryption keys, aliases and policies",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewKmsHomeView(appCtx, awsapi.NewKmsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
