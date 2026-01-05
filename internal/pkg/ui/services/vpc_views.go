package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type VpcDetailsPageView struct {
	*core.ServicePageView
	VpcListTable      *tables.VpcListTable
	VpcEndpointsTable *tables.VpcEndpointsTable
	serviceCtx        *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcDetailsPageView(
	vpcListTable *tables.VpcListTable,
	vpcEndpointsTable *tables.VpcEndpointsTable,
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcDetailsPageView {

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab("Details", tview.NewBox().SetBorder(true), 0, 1, true).
		AddAndSwitchToTab("Endpoints", vpcEndpointsTable, 0, 1, true)

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
		ServicePageView:   serviceView,
		VpcListTable:      vpcListTable,
		VpcEndpointsTable: vpcEndpointsTable,
		serviceCtx:        serviceCtx,
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
	inst.VpcListTable.SetSelectedFunc(func(row, column int) {
		var selectedVpc = inst.VpcListTable.GetSeletedVpc()
		inst.VpcEndpointsTable.RefreshVpcEndpoints(true, selectedVpc)
	})
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
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(VPC), appCtx)

	serviceRootView.
		AddAndSwitchToPage("VPCs", eventbridgeDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
