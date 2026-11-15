package waf

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newTabsPage is the tabs over the list, as the access points page of S3 has
// them
func newTabsPage(
	tabView *core.TabViewHorizontal, list core.View, appCtx *core.AppContext,
) *core.ServicePageView {
	const tabViewSize = 5000
	const listSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		list, listSize,
		tview.FlexRow,
		appCtx,
	)
	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{list},
		},
	)
	return serviceView
}

const (
	WebACLTabNameDetails   = "Details"
	WebACLTabNameRules     = "Rules"
	WebACLTabNameResources = "Associated Resources"
)

// WebACLsPageView is the regional web ACLs, with the details, rules and
// protected resources of the one selected in tabs over them
type WebACLsPageView struct {
	*core.ServicePageView
	WebACLsTable   *WebACLsTable
	DetailsTable   *WebACLDetailsTable
	RulesTable     *WebACLRulesTable
	ResourcesTable *WebACLResourcesTable
	TabView        *core.TabViewHorizontal
}

func NewWebACLsPageView(
	webAclsTable *WebACLsTable,
	detailsTable *WebACLDetailsTable,
	rulesTable *WebACLRulesTable,
	resourcesTable *WebACLResourcesTable,
	serviceCtx *core.ServiceContext[awsapi.WafApi],
) *WebACLsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(WebACLTabNameDetails, detailsTable, 0, 1, true).
		AddTab(WebACLTabNameRules, rulesTable, 0, 1, true).
		AddTab(WebACLTabNameResources, resourcesTable, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		resourcesTable.SetVisible(tabName == WebACLTabNameResources)
	})

	var serviceView = newTabsPage(tabView, webAclsTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	webAclsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	resourcesTable.ErrorMessageCallback = errorHandler

	// The rules come with the web ACL the details load
	detailsTable.SetWebACLLoadedFunc(rulesTable.SetWebACL)

	// What the list has is shown at once, but the rest is only loaded for the
	// web ACL the selection stops on
	webAclsTable.SetSelectionChangedFunc(func(row, column int) {
		var acl = webAclsTable.GetSelectedWebACL()
		detailsTable.SetWebACL(acl)
		resourcesTable.SetWebACL(aws.ToString(acl.ARN))
	})
	webAclsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
		resourcesTable.Load()
	})

	return &WebACLsPageView{
		ServicePageView: serviceView,
		WebACLsTable:    webAclsTable,
		DetailsTable:    detailsTable,
		RulesTable:      rulesTable,
		ResourcesTable:  resourcesTable,
		TabView:         tabView,
	}
}

const (
	IPSetTabNameDetails   = "Details"
	IPSetTabNameAddresses = "Addresses"
)

// IPSetsPageView is the regional IP sets, with the details and addresses of
// the one selected in tabs over them
type IPSetsPageView struct {
	*core.ServicePageView
	IPSetsTable    *IPSetsTable
	DetailsTable   *IPSetDetailsTable
	AddressesTable *IPSetAddressesTable
	TabView        *core.TabViewHorizontal
}

func NewIPSetsPageView(
	ipSetsTable *IPSetsTable,
	detailsTable *IPSetDetailsTable,
	addressesTable *IPSetAddressesTable,
	serviceCtx *core.ServiceContext[awsapi.WafApi],
) *IPSetsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(IPSetTabNameDetails, detailsTable, 0, 1, true).
		AddTab(IPSetTabNameAddresses, addressesTable, 0, 1, true)

	var serviceView = newTabsPage(tabView, ipSetsTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	ipSetsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	// The addresses come with the IP set the details load
	detailsTable.SetIPSetLoadedFunc(addressesTable.SetIPSet)

	ipSetsTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetIPSet(ipSetsTable.GetSelectedIPSet())
	})
	ipSetsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
	})

	return &IPSetsPageView{
		ServicePageView: serviceView,
		IPSetsTable:     ipSetsTable,
		DetailsTable:    detailsTable,
		AddressesTable:  addressesTable,
		TabView:         tabView,
	}
}

// The pages of the service
const (
	wafPageWebACLs = iota
	wafPageIPSets
)

// wafArnKind is the kind of regional resource the ARN names, which is empty
// for ARNs the pages can't show, such as those of CloudFront
func wafArnKind(resource arn.ARN) string {
	var named, err = awsapi.ParseWafArn(resource.String())
	if err != nil || named.Scope != types.ScopeRegional {
		return ""
	}
	return named.Kind
}

func isOpenableWafResource(resource arn.ARN) bool {
	switch wafArnKind(resource) {
	case "webacl", "ipset":
		return true
	}
	return false
}

func NewWafHomeView(appCtx *core.AppContext, api awsapi.WafApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xBF0816))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		webAclsView = NewWebACLsPageView(
			NewWebACLsTable(serviceCtx),
			NewWebACLDetailsTable(serviceCtx),
			NewWebACLRulesTable(serviceCtx),
			NewWebACLResourcesTable(serviceCtx),
			serviceCtx,
		)
		ipSetsView = NewIPSetsPageView(
			NewIPSetsTable(serviceCtx),
			NewIPSetDetailsTable(serviceCtx),
			NewIPSetAddressesTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.WAF), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Web ACLs", webAclsView, true).
		AddPage("IP Sets", ipSetsView, true, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("wafv2", isOpenableWafResource, func(resource arn.ARN) {
		switch wafArnKind(resource) {
		case "webacl":
			serviceRootView.ChangePage(wafPageWebACLs, nil)
			webAclsView.WebACLsTable.Open(resource.String())
		case "ipset":
			serviceRootView.ChangePage(wafPageIPSets, nil)
			ipSetsView.IPSetsTable.Open(resource.String())
		}
	})

	return serviceRootView
}
