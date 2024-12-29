package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CloudFormationDetailsPageView struct {
	*core.ServicePageView
	stackListTable    *StackListTable
	stackDetailsTable *StackDetailsTable
	app               *tview.Application
	api               *awsapi.CloudFormationApi
}

func NewStacksDetailsPageView(
	stackListTable *StackListTable,
	stackDetailsTable *StackDetailsTable,
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationDetailsPageView {
	const stackDetailsSize = 5000
	const stackTablesSize = 3000

	var mainPage = core.NewResizableView(
		stackDetailsTable.Table, stackDetailsSize,
		stackListTable, stackTablesSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			stackListTable.Table,
			stackDetailsTable.Table,
		},
	)

	return &CloudFormationDetailsPageView{
		ServicePageView:   serviceView,
		stackListTable:    stackListTable,
		stackDetailsTable: stackDetailsTable,
		app:               app,
		api:               api,
	}
}

func (inst *CloudFormationDetailsPageView) InitInputCapture() {
	inst.stackListTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.stackListTable.RefreshStacks(false)
		}
	})

	var refreshDetails = func(row int, force bool) {
		if row < 1 {
			return
		}
		inst.stackDetailsTable.SetStackName(inst.stackListTable.GetSelectedStackName())
		inst.stackDetailsTable.RefreshDetails(force)
	}

	inst.stackListTable.SetSelectionChangedFunc(func(row, column int) {
		refreshDetails(row, false)
	})

	inst.stackListTable.Table.SetSelectedFunc(func(row, column int) {
		refreshDetails(row, false)
		inst.app.SetFocus(inst.stackDetailsTable.Table)
	})
}

type CloudFormationStackEventsPageView struct {
	*core.ServicePageView
	stackEventsTable *StackEventsTable
	selectedStack    string
	searchPositions  []int
	app              *tview.Application
	api              *awsapi.CloudFormationApi
}

func NewStackEventsPageView(
	stackEventsTable *StackEventsTable,
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationStackEventsPageView {
	var expandedMsgView = tview.NewTextArea()
	expandedMsgView.
		SetBorder(true).
		SetTitle("Message").
		SetTitleAlign(tview.AlignLeft)

	stackEventsTable.SetSelectionChangedFunc(func(row, column int) {
		var logText = stackEventsTable.GetResourceStatusReason(row)
		expandedMsgView.SetText(logText, false)
	})

	const expandedMsgSize = 5
	const stackEventsSize = 15

	var mainPage = core.NewResizableView(
		expandedMsgView, expandedMsgSize,
		stackEventsTable, stackEventsSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			stackEventsTable.Table,
			expandedMsgView,
		},
	)
	return &CloudFormationStackEventsPageView{
		ServicePageView:  serviceView,
		stackEventsTable: stackEventsTable,
		selectedStack:    "",
		app:              app,
		api:              api,
	}
}

func (inst *CloudFormationStackEventsPageView) InitInputCapture() {
	inst.stackEventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.stackEventsTable.RefreshEvents(true)
		case tcell.KeyCtrlN:
			inst.stackEventsTable.RefreshEvents(false)
		}
		return event
	})
}

func NewStacksHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer core.ResetGlobalStyle()

	var (
		api               = awsapi.NewCloudFormationApi(config, logger)
		stacksDetailsView = NewStacksDetailsPageView(
			NewStackListTable(app, api, logger),
			NewStackDetailsTable(app, api, logger),
			app, api, logger,
		)
		stackEventsView = NewStackEventsPageView(
			NewStackEventsTable(app, api, logger),
			app, api, logger,
		)
	)

	var pages = tview.NewPages().
		AddPage("Events", stackEventsView, true, true).
		AddAndSwitchToPage("Stacks", stacksDetailsView, true)

	var orderedPages = []string{
		"Stacks",
		"Events",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDFORMATION), pages, orderedPages).Init()

	stacksDetailsView.stackDetailsTable.Table.SetSelectedFunc(func(row, column int) {
		var selectedStackName = stacksDetailsView.stackListTable.GetSelectedStackName()
		stackEventsView.stackEventsTable.SetSelectedStackName(selectedStackName)
		stackEventsView.stackEventsTable.RefreshEvents(true)
		serviceRootView.ChangePage(1, stackEventsView.stackEventsTable.Table)
	})

	stackEventsView.InitInputCapture()
	stacksDetailsView.InitInputCapture()

	return serviceRootView.RootView
}
