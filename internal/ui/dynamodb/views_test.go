package dynamodb

import (
	"aws-tui/internal/ui/dynamodb/dynamodbfake"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func TestHomeView__ResetKeyLoadsTablesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewDynamoDBHomeView(appCtx, dynamodbfake.New(dynamodbfake.WithTables([]string{"orders-table"})))
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-table")
}

func TestItemsPageView__MessageViewIsInATab(t *testing.T) {
	var serviceCtx = uitest.NewServiceContext[awsapi.DynamoDBApi]()
	var view = NewDynamoDBItemsPageView(NewDynamoDBGenericTable(serviceCtx), serviceCtx)

	if name, _ := view.TabView.GetCurrentTab(); name != DdbTabNameMessage {
		t.Fatalf("Expected the %q tab but got %q", DdbTabNameMessage, name)
	}
	if view.TabView.GetTab(DdbTabNameMessage) == nil {
		t.Fatalf("Expected a %q tab", DdbTabNameMessage)
	}
}
