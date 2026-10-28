package secretsmanager

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SecretsTabName = string

const (
	SecretsTabNameDetails SecretsTabName = "Details"
	SecretsTabNameValue   SecretsTabName = "Value"
)

type SecretsPageView struct {
	*core.ServicePageView
	SecretsTable       *SecretsListTable
	SecretDetailsTable *SecretDetailsTable
	SecretValueView    *SecretValueView
	TabView            *core.TabViewHorizontal
	serviceCtx         *core.ServiceContext[awsapi.SecretsManagerApi]
}

func NewSecretsPageView(
	secretDetailsTable *SecretDetailsTable,
	secretValueView *SecretValueView,
	secretsTable *SecretsListTable,
	serviceCtx *core.ServiceContext[awsapi.SecretsManagerApi],
) *SecretsPageView {

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SecretsTabNameDetails, secretDetailsTable, 0, 1, true).
		AddTab(SecretsTabNameValue, secretValueView, 0, 1, true)

	const detailsSize = 25
	const secretsTableSize = 75

	var mainPage = core.NewResizableView(
		tabView, detailsSize,
		secretsTable, secretsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{secretsTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	secretsTable.ErrorMessageCallback = errorHandler
	secretValueView.ErrorMessageCallback = errorHandler

	return &SecretsPageView{
		ServicePageView:    serviceView,
		SecretsTable:       secretsTable,
		SecretDetailsTable: secretDetailsTable,
		SecretValueView:    secretValueView,
		TabView:            tabView,
		serviceCtx:         serviceCtx,
	}
}

func (inst *SecretsPageView) InitInputCapture() {
	inst.SecretsTable.SetSelectionChangedFunc(func(row, column int) {
		inst.SecretDetailsTable.RefreshDetails(inst.SecretsTable.GetSelectedSecret())
		inst.SecretValueView.Clear()
	})

	// The value is only fetched when it is asked for
	inst.SecretsTable.SetSelectedFunc(func(row, column int) {
		var secret = inst.SecretsTable.GetSelectedSecret()
		inst.SecretValueView.RefreshValue(aws.ToString(secret.ARN), func() {
			inst.TabView.SwitchToTab(SecretsTabNameValue)
		})
	})
}

func NewSecretsManagerHomeView(appCtx *core.AppContext, api awsapi.SecretsManagerApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xDD344C))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var secretsView = NewSecretsPageView(
		NewSecretDetailsTable(serviceCtx),
		NewSecretValueView(serviceCtx),
		NewSecretsListTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(core.SECRETS_MANAGER), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Secrets", secretsView, true)

	serviceRootView.InitPageNavigation()

	secretsView.InitInputCapture()

	return serviceRootView
}
