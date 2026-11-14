package sfn

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.STATE_MACHINES,
	Icon:        "󱁊",
	Description: "Step Functions and executions",
	Shortcut:    '4',
	Aliases:     []string{"sfn"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewStepFunctionsHomeView(
			appCtx,
			awsapi.NewStateMachineApi(appCtx.GetApiClients(), appCtx.Logger),
			awsapi.NewCloudWatchLogsApi(appCtx.GetApiClients(), appCtx.Logger),
		)
	},
}
