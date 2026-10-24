package vpc

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type VpcTabName = string

const (
	VpcTabNameSubnets        VpcTabName = "Subnets"
	VpcTabNameSecurityGroups VpcTabName = "Security Groups"
	VpcTabNameEndpoints      VpcTabName = "Endpoints"
	VpcTabNameTags           VpcTabName = "Tags"
)

type VpcDetailsPageView struct {
	*core.ServicePageView
	VpcListTable           *VpcListTable
	VpcEndpointsTable      *VpcEndpointsTable
	VpcSubnetsTable        *VpcSubnetsTable
	VpcSecurityGroupsTable *VpcSecurityGroupsTable
	TagsTable              *core.TagsTable[types.Tag, awsapi.Ec2Api]
	TabsView               *core.TabViewHorizontal
	serviceCtx             *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcDetailsPageView(
	vpcListTable *VpcListTable,
	vpcEndpointsTable *VpcEndpointsTable,
	vpcSubnetsTable *VpcSubnetsTable,
	vpcSecurityGroupsTable *VpcSecurityGroupsTable,
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcDetailsPageView {
	var tagsTable = core.NewTagsTable[types.Tag](serviceCtx).
		SetExtractKeyValFunc(func(t types.Tag) (k string, v string) {
			return aws.ToString(t.Key), aws.ToString(t.Value)
		}).
		SetGetTagsFunc(func() func() ([]types.Tag, error) {
			var tags = vpcListTable.GetSelectedVpc().Tags
			return func() ([]types.Tag, error) { return tags, nil }
		})

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(VpcTabNameSubnets, vpcSubnetsTable, 0, 1, true).
		AddTab(VpcTabNameSecurityGroups, vpcSecurityGroupsTable, 0, 1, true).
		AddTab(VpcTabNameEndpoints, vpcEndpointsTable, 0, 1, true).
		AddTab(VpcTabNameTags, tagsTable, 0, 1, true)

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
		case VpcTabNameTags:
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

func NewVpcHomeView(appCtx *core.AppContext, api awsapi.Ec2Api) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		eventbridgeDetailsView = NewVpcDetailsPageView(
			NewVpcListTable(serviceCtx),
			NewVpcEndpointsTable(serviceCtx),
			NewVpcSubnetsTable(serviceCtx),
			NewVpcSecurityGroupsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.VPC), appCtx)

	serviceRootView.
		AddAndSwitchToPage("VPCs", eventbridgeDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
