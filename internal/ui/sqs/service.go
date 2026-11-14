package sqs

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.SQS,
	Icon:        "󱃔",
	Description: "Queues and their attributes",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewSqsHomeView(appCtx, awsapi.NewSqsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
