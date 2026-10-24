package dynamodb

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func TestHomeView__ResetKeyLoadsTablesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewDynamoDBHomeView(appCtx, &fakeDynamoDBApi{tables: []string{"orders-table"}})
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-table")
}
