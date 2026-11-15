package bedrock

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.BEDROCK,
	Icon:        "󰚩",
	Description: "Foundation models and inference profiles",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewBedrockHomeView(appCtx, awsapi.NewBedrockApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
