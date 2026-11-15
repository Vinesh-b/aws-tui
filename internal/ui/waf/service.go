package waf

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.WAF,
	Icon:        "󰒃",
	Description: "Web ACLs and IP sets",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewWafHomeView(appCtx, awsapi.NewWafApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
