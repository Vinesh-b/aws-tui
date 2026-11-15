package kms

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/kms/kmsfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/gdamore/tcell/v2"
)

func TestHomeView__ResetKeyLoadsKeysFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewKmsHomeView(appCtx, newApi())
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "alias/orders")
}

func TestHomeView__EnterShowsThePolicyOfTheSelectedKey(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewKmsHomeView(appCtx, newApi())
	var screen = uitest.ShowPage(t, appCtx.App, page)
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "alias/orders")

	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.AwaitScreenText(t, appCtx.App, screen, "2012-10-17")
}

func TestKeysPageView__MovingThroughKeysDescribesOnlyTheKeyItStopsOn(t *testing.T) {
	var api = kmsfake.New(kmsfake.WithKeys([]types.KeyListEntry{
		newKeyEntry("1111"), newKeyEntry("2222"), newKeyEntry("3333"), newKeyEntry("4444"),
	}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.KmsApi](api)
	var view = NewKmsKeysPageView(NewKmsKeysTable(serviceCtx), NewKmsKeyDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	view.KeysTable.RefreshKeys(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.KeysTable.GetTable().GetRowCount() == 5 })
	var before = len(api.Described())

	uitest.OnUI(t, serviceCtx.App, func() {
		for row := 2; row <= 4; row++ {
			view.KeysTable.Select(row, 0)
		}
	})

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return slices.Contains(api.Described()[before:], "4444")
	}, "key 4444 to be described")
	uitest.Consistently(t, serviceCtx.App, 2*core.SelectionLoadDelay, func() bool {
		var described = api.Described()[before:]
		return !slices.Contains(described, "3333") && described[len(described)-1] == "4444"
	}, "only the first and last keys of the move to be described")
}
