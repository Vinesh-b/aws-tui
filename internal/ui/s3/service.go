package s3

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// Service adds the service to the app
var Service = core.Service{
	Id:          core.S3BUCKETS,
	Icon:        "󱐕",
	Description: "S3 buckets, objects and access points",
	Shortcut:    '5',
	Aliases:     []string{"s3"},
	Build: func(appCtx *core.AppContext) core.ServicePage {
		return NewS3bucketsHomeView(appCtx, awsapi.NewS3BucketsApi(appCtx.GetApiClients(), appCtx.Logger))
	},
}
