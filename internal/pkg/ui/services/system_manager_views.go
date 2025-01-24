package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SystemManagerDetailsPageView struct {
	*core.ServicePageView
	SSMParametersListTable   *tables.SSMParametersListTable
	SSMParameterHistoryTable *tables.SSMParameterHistoryTable

	app *tview.Application
	api *awsapi.SystemsManagerApi
}

func NewSystemManagerDetailsPageView(
	ssmParamsListTable *tables.SSMParametersListTable,
	ssmParamHistoryTable *tables.SSMParameterHistoryTable,
	app *tview.Application,
	api *awsapi.SystemsManagerApi,
	logger *log.Logger,
) *SystemManagerDetailsPageView {
	var paramValueView = core.JsonTextView[types.Parameter]{
		TextView: core.NewSearchableTextView("", app),
		ExtractTextFunc: func(data types.Parameter) string {
			return aws.ToString(data.Value)
		},
	}

	var selectionFunc = func(row int, col int) {
		paramValueView.SetTitle("Value")
		paramValueView.SetText(ssmParamsListTable.GetPrivateData(row, col))
	}

	ssmParamsListTable.SetSelectionChangedFunc(selectionFunc)
	const expandItemViewSize = 25
	const itemsTableSize = 75

	var tabView = core.NewTabView(app, logger).
		AddAndSwitchToTab("Param Value", paramValueView.TextView, 0, 1, true).
		AddTab("Param History", ssmParamHistoryTable, 0, 1, true)

	var mainPage = core.NewResizableView(
		tabView, expandItemViewSize,
		ssmParamsListTable, itemsTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SystemManagerDetailsPageView{
		ServicePageView:          serviceView,
		SSMParametersListTable:   ssmParamsListTable,
		SSMParameterHistoryTable: ssmParamHistoryTable,
		app:                      app,
		api:                      api,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	ssmParamsListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
			{ssmParamsListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *SystemManagerDetailsPageView) initInputCapture() {
	inst.SSMParametersListTable.SetSelectedFunc(func(row, column int) {
		inst.SSMParameterHistoryTable.SetSeletedParameter(inst.SSMParametersListTable.GetSeletedParameter())
		inst.SSMParameterHistoryTable.RefreshHistory(true)
	})
}

func NewSystemManagerHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0xFF5AAD))
	defer core.ResetGlobalStyle()

	var (
		api                       = awsapi.NewSystemsManagerApi(config, logger)
		systemManagersDetailsView = NewSystemManagerDetailsPageView(
			tables.NewSSMParametersListTable(app, api, logger),
			tables.NewSSMParameterHistoryTable(app, api, logger),
			app, api, logger,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(SYSTEMS_MANAGER), app, &config, logger)

	serviceRootView.
		AddAndSwitchToPage("SystemManagers", systemManagersDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
