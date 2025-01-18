package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CloudFormationDetailsPageView struct {
	*core.ServicePageView
	stackListTable    *tables.StackListTable
	stackDetailsTable *tables.StackDetailsTable
	app               *tview.Application
	api               *awsapi.CloudFormationApi
}

func NewStacksDetailsPageView(
	stackListTable *tables.StackListTable,
	stackDetailsTable *tables.StackDetailsTable,
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *CloudFormationDetailsPageView {
	const stackDetailsSize = 5000
	const stackTablesSize = 3000

	var mainPage = core.NewResizableView(
		stackDetailsTable, stackDetailsSize,
		stackListTable, stackTablesSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{stackDetailsTable},
			{stackListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	stackListTable.ErrorMessageCallback = errorHandler
	stackDetailsTable.ErrorMessageCallback = errorHandler

	return &CloudFormationDetailsPageView{
		ServicePageView:   serviceView,
		stackListTable:    stackListTable,
		stackDetailsTable: stackDetailsTable,
		app:               app,
		api:               api,
	}
}

func (inst *CloudFormationDetailsPageView) InitInputCapture() {
	inst.stackListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.stackDetailsTable.RefreshDetails(inst.stackListTable.GetSelectedStack())
	})
}

type CloudFormationStackEventsPageView struct {
	*core.ServicePageView
	stackEventsTable *tables.StackEventsTable
	selectedStack    string
	searchPositions  []int
	app              *tview.Application
	api              *awsapi.CloudFormationApi
}

func NewStackEventsPageView(
	stackEventsTable *tables.StackEventsTable,
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
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{expandedMsgView},
			{stackEventsTable},
		},
	)

	stackEventsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

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
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.stackEventsTable.RefreshEvents(true)
		case core.APP_KEY_BINDINGS.LoadMoreData:
			inst.stackEventsTable.RefreshEvents(false)
		}
		return event
	})
}

func NewStacksHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0x660033))
	defer core.ResetGlobalStyle()

	var (
		api               = awsapi.NewCloudFormationApi(config, logger)
		stacksDetailsView = NewStacksDetailsPageView(
			tables.NewStackListTable(app, api, logger),
			tables.NewStackDetailsTable(app, api, logger),
			app, api, logger,
		)
		stackEventsView = NewStackEventsPageView(
			tables.NewStackEventsTable(app, api, logger),
			app, api, logger,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(CLOUDFORMATION), app, &config, logger)

	serviceRootView.
		AddAndSwitchToPage("Stacks", stacksDetailsView, true).
		AddPage("Events", stackEventsView, true, true)

	serviceRootView.InitPageNavigation()

	stacksDetailsView.stackListTable.SetSelectedFunc(func(row, column int) {
		var selectedStackName = stacksDetailsView.stackListTable.GetSelectedStackName()
		if len(selectedStackName) > 0 {
			stackEventsView.stackEventsTable.SetSelectedStackName(selectedStackName)
			stackEventsView.stackEventsTable.SetTitleExtra(selectedStackName)
			stackEventsView.stackEventsTable.RefreshEvents(true)
			serviceRootView.ChangePage(1, nil)
		}
	})

	stackEventsView.InitInputCapture()
	stacksDetailsView.InitInputCapture()

	return serviceRootView
}
