package dynamodb

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.DYNAMODB,
	Icon:        "󰆼",
	Description: "View and search DynamoDB tables",
	Shortcut:    '3',
	Aliases:     []string{"ddb"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewDynamoDBHomeView(appCtx, awsapi.NewDynamoDBApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
