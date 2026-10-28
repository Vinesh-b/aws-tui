package kms

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type KmsTabName = string

const (
	KmsTabNameDetails KmsTabName = "Details"
)

type KmsKeysPageView struct {
	*core.ServicePageView
	KeysTable    *KmsKeysTable
	DetailsTable *KmsKeyDetailsTable
	TabView      *core.TabViewHorizontal
	serviceCtx   *core.ServiceContext[awsapi.KmsApi]
}

func NewKmsKeysPageView(
	keysTable *KmsKeysTable,
	detailsTable *KmsKeyDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.KmsApi],
) *KmsKeysPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(KmsTabNameDetails, detailsTable, 0, 1, true)

	const tabViewSize = 3000
	const keysTableSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		keysTable, keysTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &KmsKeysPageView{
		ServicePageView: serviceView,
		KeysTable:       keysTable,
		DetailsTable:    detailsTable,
		TabView:         tabView,
		serviceCtx:      serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	keysTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{keysTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *KmsKeysPageView) initInputCapture() {
	inst.KeysTable.SetSelectionChangedFunc(func(row, column int) {
		inst.DetailsTable.SetSelectedKey(inst.KeysTable.GetSelectedKey())
		inst.DetailsTable.RefreshDetails()
	})
}

// The pages of the service
const (
	kmsPageKeys = iota
	kmsPagePolicy
)

func NewKmsHomeView(appCtx *core.AppContext, api awsapi.KmsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC2264))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		keysView = NewKmsKeysPageView(
			NewKmsKeysTable(serviceCtx),
			NewKmsKeyDetailsTable(serviceCtx),
			serviceCtx,
		)
		policyView = NewKmsKeyPolicyPageView(serviceCtx)
	)

	var serviceRootView = core.NewServiceRootView(string(core.KMS), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Keys", keysView, true).
		AddPage("Policy", policyView, true, true)

	serviceRootView.InitPageNavigation()

	keysView.KeysTable.SetSelectedFunc(func(row, column int) {
		if keyId := keysView.KeysTable.GetSelectedKey().keyId; len(keyId) > 0 {
			policyView.LoadPolicy(keyId)
			serviceRootView.ChangePage(kmsPagePolicy, nil)
		}
	})

	serviceRootView.SetResourceOpener("kms", func(resource arn.ARN) {
		serviceRootView.ChangePage(kmsPageKeys, nil)
		keysView.KeysTable.Open(resource.Resource)
	})

	return serviceRootView
}
