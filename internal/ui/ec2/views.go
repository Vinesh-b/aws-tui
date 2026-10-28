package ec2

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Ec2TabName = string

const (
	Ec2TabNameDetails Ec2TabName = "Details"
)

type Ec2InstancesPageView struct {
	*core.ServicePageView
	InstancesTable *Ec2InstancesTable
	DetailsTable   *Ec2DetailsTable
	TabView        *core.TabViewHorizontal
	serviceCtx     *core.ServiceContext[awsapi.Ec2Api]
}

func NewEc2InstancesPageView(
	instancesTable *Ec2InstancesTable,
	detailsTable *Ec2DetailsTable,
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *Ec2InstancesPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(Ec2TabNameDetails, detailsTable, 0, 1, true)

	const tabViewSize = 3000
	const instancesTableSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		instancesTable, instancesTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &Ec2InstancesPageView{
		ServicePageView: serviceView,
		InstancesTable:  instancesTable,
		DetailsTable:    detailsTable,
		TabView:         tabView,
		serviceCtx:      serviceCtx,
	}

	instancesTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{instancesTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *Ec2InstancesPageView) initInputCapture() {
	inst.InstancesTable.SetSelectionChangedFunc(func(row, column int) {
		inst.DetailsTable.SetInstance(inst.InstancesTable.GetSelectedInstance())
	})
}

func NewEc2HomeView(appCtx *core.AppContext, api awsapi.Ec2Api) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xCC6600))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		instancesView = NewEc2InstancesPageView(
			NewEc2InstancesTable(serviceCtx),
			NewEc2DetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.EC2), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Instances", instancesView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
