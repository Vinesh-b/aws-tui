package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type VpcDetailsPageView struct {
	*core.ServicePageView
	VpcListTable           *tables.VpcListTable
	VpcEndpointsTable      *tables.VpcEndpointsTable
	VpcSubnetsTable        *tables.VpcSubnetsTable
	VpcSecurityGroupsTable *tables.VpcSecurityGroupsTable
	TagsTable              *tables.TagsTable[types.Tag, awsapi.Ec2Api]
	TabsView               *core.TabViewHorizontal
	serviceCtx             *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcDetailsPageView(
	vpcListTable *tables.VpcListTable,
	vpcEndpointsTable *tables.VpcEndpointsTable,
	vpcSubnetsTable *tables.VpcSubnetsTable,
	vpcSecurityGroupsTable *tables.VpcSecurityGroupsTable,
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcDetailsPageView {
	var tagsTable = tables.NewTagsTable(
		serviceCtx,
		func(t types.Tag) (string, string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		},
		func() ([]types.Tag, error) {
			return vpcListTable.GetSeletedVpc().Tags, nil
		},
	)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab("Subnets", vpcSubnetsTable, 0, 1, true).
		AddTab("Security Groups", vpcSecurityGroupsTable, 0, 1, true).
		AddTab("Endpoints", vpcEndpointsTable, 0, 1, true).
		AddTab("Tags", tagsTable, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		vpcListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &VpcDetailsPageView{
		ServicePageView:        serviceView,
		VpcListTable:           vpcListTable,
		VpcEndpointsTable:      vpcEndpointsTable,
		VpcSubnetsTable:        vpcSubnetsTable,
		VpcSecurityGroupsTable: vpcSecurityGroupsTable,
		TagsTable:              tagsTable,
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
			{tabView.GetTabDisplayView()},
			{vpcListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *VpcDetailsPageView) initInputCapture() {
	var loadedTabs = map[int]bool{}
	var tabChangeFunc = func(tabName string, index int) {
		// Only automaticaly load new data on first change
		if loadedTabs[index] {
			return
		}

		var selectedVpc = inst.VpcListTable.GetSeletedVpc()

		switch tabName {
		case "Subnets":
			inst.VpcSubnetsTable.RefreshVpcSubnets(true, selectedVpc)
		case "Security Groups":
			inst.VpcSecurityGroupsTable.RefreshVpcSecurityGroups(true, selectedVpc)
		case "Endpoints":
			inst.VpcEndpointsTable.RefreshVpcEndpoints(true, selectedVpc)
		case "Tags":
			inst.TagsTable.RefreshDetails()
		}

		loadedTabs[index] = true
	}

	inst.VpcListTable.SetSelectedFunc(func(row, column int) {
		var tabName, index = inst.TabsView.GetDefaultTab()
		tabChangeFunc(tabName, index)
	})

	inst.TabsView.SetOnTabChangeFunc(tabChangeFunc)
}

func NewVpcHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewEc2Api(appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		eventbridgeDetailsView = NewVpcDetailsPageView(
			tables.NewVpcListTable(serviceCtx),
			tables.NewVpcEndpointsTable(serviceCtx),
			tables.NewVpcSubnetsTable(serviceCtx),
			tables.NewVpcSecurityGroupsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(VPC), appCtx)

	serviceRootView.
		AddAndSwitchToPage("VPCs", eventbridgeDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
