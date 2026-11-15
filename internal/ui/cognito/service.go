package cognito

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.COGNITO,
	Icon:        "󰀄",
	Description: "User pools and identity pools",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewCognitoHomeView(appCtx, awsapi.NewCognitoApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
