package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CloudFormationDetailsPage struct {
	StackListTable    *StackListTable
	StackDetailsTable *StackDetailsTable
	RootView          *tview.Flex
	app               *tview.Application
	api               *awsapi.CloudFormationApi
}

func NewStacksDetailsPage(
	stackListTable *StackListTable,
	stackDetailsTable *StackDetailsTable,
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationDetailsPage {
	const stackDetailsSize = 5000
	const stackTablesSize = 3000

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(stackDetailsTable.Table, 0, stackDetailsSize, false).
		AddItem(stackListTable.RootView, 0, stackTablesSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		stackDetailsTable.Table, stackListTable.RootView,
		stackDetailsSize, stackTablesSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			stackListTable.Table,
			stackDetailsTable.Table,
		},
	)
	return &CloudFormationDetailsPage{
		StackListTable:    stackListTable,
		StackDetailsTable: stackDetailsTable,
		RootView:          serviceView.RootView,
		app:               app,
		api:               api,
	}
}

func (inst *CloudFormationDetailsPage) InitInputCapture() {
	inst.StackListTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.StackListTable.RefreshStacks(false)
		}
	})

	var refreshDetails = func(row int, force bool) {
		if row < 1 {
			return
		}
		inst.StackDetailsTable.SetStackName(inst.StackListTable.GetSelectedStackName())
		inst.StackDetailsTable.RefreshDetails(force)
	}

	inst.StackListTable.SetSelectionChangedFunc(func(row, column int) {
		refreshDetails(row, false)
	})

	inst.StackListTable.Table.SetSelectedFunc(func(row, column int) {
		refreshDetails(row, false)
		inst.app.SetFocus(inst.StackDetailsTable.Table)
	})
}

type CloudFormationStackEventsPage struct {
	StackEventsTable *StackEventsTable
	RootView         *tview.Flex
	selectedStack    string
	searchPositions  []int
	app              *tview.Application
	api              *awsapi.CloudFormationApi
}

func NewStackEventsPage(
	stackEventsTable *StackEventsTable,
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationStackEventsPage {
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

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(expandedMsgView, 0, expandedMsgSize, false).
		AddItem(stackEventsTable.RootView, 0, stackEventsSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)
	serviceView.SetResizableViews(
		expandedMsgView, stackEventsTable.RootView,
		expandedMsgSize, stackEventsSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			stackEventsTable.Table,
			expandedMsgView,
		},
	)
	return &CloudFormationStackEventsPage{
		StackEventsTable: stackEventsTable,
		RootView:         serviceView.RootView,
		selectedStack:    "",
		app:              app,
		api:              api,
	}
}

func (inst *CloudFormationStackEventsPage) InitInputCapture() {
	inst.StackEventsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.StackEventsTable.RefreshEvents(true)
		case tcell.KeyCtrlN:
			inst.StackEventsTable.RefreshEvents(false)
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
		stacksDetailsView = NewStacksDetailsPage(
			NewStackListTable(app, api, logger),
			NewStackDetailsTable(app, api, logger),
			app, api, logger,
		)
		stackEventsView = NewStackEventsPage(
			NewStackEventsTable(app, api, logger),
			app, api, logger,
		)
	)

	var pages = tview.NewPages().
		AddPage("Events", stackEventsView.RootView, true, true).
		AddAndSwitchToPage("Stacks", stacksDetailsView.RootView, true)

	var orderedPages = []string{
		"Stacks",
		"Events",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDFORMATION), pages, orderedPages).Init()

	stacksDetailsView.StackDetailsTable.Table.SetSelectedFunc(func(row, column int) {
		var selectedStackName = stacksDetailsView.StackListTable.GetSelectedStackName()
		stackEventsView.StackEventsTable.SetSelectedStackName(selectedStackName)
		stackEventsView.StackEventsTable.RefreshEvents(true)
		serviceRootView.ChangePage(1, stackEventsView.StackEventsTable.Table)
	})

	stackEventsView.InitInputCapture()
	stacksDetailsView.InitInputCapture()

	return serviceRootView.RootView
}
