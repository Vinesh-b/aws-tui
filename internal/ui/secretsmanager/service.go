package secretsmanager

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.SECRETS_MANAGER,
	Icon:        "󰌋",
	Description: "Secrets and their values",
	Shortcut:    core.NoShortcut,
	Aliases:     []string{"secrets"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewSecretsManagerHomeView(appCtx, awsapi.NewSecretsManagerApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
