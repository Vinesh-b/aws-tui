package transfer

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.TRANSFER_FAMILY,
	Icon:        "󰕒",
	Description: "SFTP, FTPS, FTP and AS2 servers, users and connectors",
	Shortcut:    core.NoShortcut,
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewTransferHomeView(appCtx, awsapi.NewTransferApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
