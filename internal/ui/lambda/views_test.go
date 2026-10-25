package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestHomeView__ResetKeyLoadsFunctionsFromApi(t *testing.T) {
	var api = &fakeLambdaApi{functions: []types.FunctionConfiguration{newFunction("orders-handler")}}
	var appCtx = uitest.NewAppContext()
	var page = NewLambdaHomeView(appCtx, api, awsapi.CloudWatchLogsApi(nil), awsapi.CloudWatchMetricsApi(nil))
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-handler")
}
