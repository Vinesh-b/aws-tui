package cloudtrail

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.CLOUDTRAIL,
	Icon:        "",
	Description: "Event history and trails",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewCloudTrailHomeView(appCtx, awsapi.NewCloudTrailApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
