package lambda

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.LAMBDA,
	Icon:        "󰘧",
	Description: "Lambdas and logs",
	Shortcut:    '1',
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewLambdaHomeView(
			appCtx,
			awsapi.NewLambdaApi(appCtx.GetApiClients(), appCtx.Logger),
			awsapi.NewCloudWatchLogsApi(appCtx.GetApiClients(), appCtx.Logger),
			awsapi.NewCloudWatchMetricsApi(appCtx.GetApiClients(), appCtx.Logger),
		)
	},
}
