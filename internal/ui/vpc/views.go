package vpc

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type VpcTabName = string

const (
	VpcTabNameDetails        VpcTabName = "Details"
	VpcTabNameSubnets        VpcTabName = "Subnets"
	VpcTabNameSecurityGroups VpcTabName = "Security Groups"
	VpcTabNameEndpoints      VpcTabName = "Endpoints"
)

type VpcDetailsPageView struct {
	*core.ServicePageView
	VpcListTable           *VpcListTable
	VpcDetailsTable        *VpcDetailsTable
	VpcEndpointsTable      *VpcEndpointsTable
	VpcSubnetsTable        *VpcSubnetsTable
	VpcSecurityGroupsTable *VpcSecurityGroupsTable
	TabsView               *core.TabViewHorizontal
	serviceCtx             *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcDetailsPageView(
	vpcListTable *VpcListTable,
	vpcDetailsTable *VpcDetailsTable,
	vpcEndpointsTable *VpcEndpointsTable,
	vpcSubnetsTable *VpcSubnetsTable,
	vpcSecurityGroupsTable *VpcSecurityGroupsTable,
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcDetailsPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(VpcTabNameDetails, vpcDetailsTable, 0, 1, true).
		AddTab(VpcTabNameSubnets, vpcSubnetsTable, 0, 1, true).
		AddTab(VpcTabNameSecurityGroups, vpcSecurityGroupsTable, 0, 1, true).
		AddTab(VpcTabNameEndpoints, vpcEndpointsTable, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		vpcListTable, tableViewSize,
		tview.FlexRow,
		serviceCtx.AppContext,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &VpcDetailsPageView{
		ServicePageView:        serviceView,
		VpcListTable:           vpcListTable,
		VpcDetailsTable:        vpcDetailsTable,
		VpcEndpointsTable:      vpcEndpointsTable,
		VpcSubnetsTable:        vpcSubnetsTable,
		VpcSecurityGroupsTable: vpcSecurityGroupsTable,
		TabsView:               tabView,
		serviceCtx:             serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	vpcEndpointsTable.ErrorMessageCallback = errorHandler
	vpcListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{vpcListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *VpcDetailsPageView) initInputCapture() {
	var loadedTabs = map[int]bool{}
	var lastSelection = ""
	var tabChangeFunc = func(tabName string, index int) {
		var selectedVpc = inst.VpcListTable.GetSelectedVpc()

		if vpcId := aws.ToString(selectedVpc.VpcId); vpcId != lastSelection {
			lastSelection = vpcId
			utils.ClearMap(loadedTabs)
		}

		// Only automaticaly load new data on first change
		if loadedTabs[index] {
			return
		}

		switch tabName {
		case VpcTabNameSubnets:
			inst.VpcSubnetsTable.RefreshVpcSubnets(true, selectedVpc)
		case VpcTabNameSecurityGroups:
			inst.VpcSecurityGroupsTable.RefreshVpcSecurityGroups(true, selectedVpc)
		case VpcTabNameEndpoints:
			inst.VpcEndpointsTable.RefreshVpcEndpoints(true, selectedVpc)
		}

		loadedTabs[index] = true
	}

	inst.VpcListTable.SetSelectedFunc(func(row, column int) {
		var tabName, index = inst.TabsView.GetDefaultTab()
		tabChangeFunc(tabName, index)
	})

	inst.VpcListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.VpcDetailsTable.SetVpc(inst.VpcListTable.GetSelectedVpc())
	})

	inst.TabsView.SetOnTabChangeFunc(tabChangeFunc)
}

const (
	NetworkAclTabNameDetails       = "Details"
	NetworkAclTabNameInboundRules  = "Inbound Rules"
	NetworkAclTabNameOutboundRules = "Outbound Rules"
	NetworkAclTabNameAssociations  = "Subnet Associations"
)

// NetworkAclsPageView is the network ACLs of the region, with the details,
// rules and subnet associations of the one selected in tabs over them
type NetworkAclsPageView struct {
	*core.ServicePageView
	NetworkAclsTable   *NetworkAclsTable
	DetailsTable       *NetworkAclDetailsTable
	InboundRulesTable  *NetworkAclRulesTable
	OutboundRulesTable *NetworkAclRulesTable
	AssociationsTable  *NetworkAclAssociationsTable
	TabView            *core.TabViewHorizontal
}

func NewNetworkAclsPageView(
	networkAclsTable *NetworkAclsTable,
	detailsTable *NetworkAclDetailsTable,
	inboundRulesTable *NetworkAclRulesTable,
	outboundRulesTable *NetworkAclRulesTable,
	associationsTable *NetworkAclAssociationsTable,
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *NetworkAclsPageView {
	const tabViewSize = 5000
	const listSize = 5000

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(NetworkAclTabNameDetails, detailsTable, 0, 1, true).
		AddTab(NetworkAclTabNameInboundRules, inboundRulesTable, 0, 1, true).
		AddTab(NetworkAclTabNameOutboundRules, outboundRulesTable, 0, 1, true).
		AddTab(NetworkAclTabNameAssociations, associationsTable, 0, 1, true)

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		networkAclsTable, listSize,
		tview.FlexRow,
		serviceCtx.AppContext,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{networkAclsTable},
		},
	)

	networkAclsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	// The list has everything the tabs show, so they all change with it
	networkAclsTable.SetSelectionChangedFunc(func(row, column int) {
		var acl = networkAclsTable.GetSelectedNetworkAcl()
		detailsTable.SetNetworkAcl(acl)
		inboundRulesTable.SetNetworkAcl(acl)
		outboundRulesTable.SetNetworkAcl(acl)
		associationsTable.SetNetworkAcl(acl)
	})

	return &NetworkAclsPageView{
		ServicePageView:    serviceView,
		NetworkAclsTable:   networkAclsTable,
		DetailsTable:       detailsTable,
		InboundRulesTable:  inboundRulesTable,
		OutboundRulesTable: outboundRulesTable,
		AssociationsTable:  associationsTable,
		TabView:            tabView,
	}
}

func NewVpcHomeView(appCtx *core.AppContext, api awsapi.Ec2Api) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		eventbridgeDetailsView = NewVpcDetailsPageView(
			NewVpcListTable(serviceCtx),
			NewVpcDetailsTable(serviceCtx),
			NewVpcEndpointsTable(serviceCtx),
			NewVpcSubnetsTable(serviceCtx),
			NewVpcSecurityGroupsTable(serviceCtx),
			serviceCtx,
		)
		networkAclsView = NewNetworkAclsPageView(
			NewNetworkAclsTable(serviceCtx),
			NewNetworkAclDetailsTable(serviceCtx),
			NewNetworkAclInboundRulesTable(serviceCtx),
			NewNetworkAclOutboundRulesTable(serviceCtx),
			NewNetworkAclAssociationsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.VPC), appCtx)

	serviceRootView.
		AddAndSwitchToPage("VPCs", eventbridgeDetailsView, true).
		AddPage("Network ACLs", networkAclsView, true, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
