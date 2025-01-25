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
	TabView                  *core.TabView

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
	var paramValueView = core.JsonTextView[any]{
		TextView: core.NewSearchableTextView("", app),
		ExtractTextFunc: func(data any) string {
			switch data.(type) {
			case types.Parameter:
				return aws.ToString(data.(types.Parameter).Value)
			case types.ParameterHistory:
				return aws.ToString(data.(types.ParameterHistory).Value)
			}
			return ""
		},
	}

	paramValueView.SetTitle("Value")
	ssmParamsListTable.SetSelectionChangedFunc(func(row, column int) {
		paramValueView.SetText(ssmParamsListTable.GetSeletedParameter())
	})

	ssmParamHistoryTable.SetSelectionChangedFunc(func(row, column int) {
		paramValueView.SetText(ssmParamHistoryTable.GetSeletedHistory())
	})

	const expandItemViewSize = 25
	const itemsTableSize = 75

	var tabView = core.NewTabView(app, logger).
		AddAndSwitchToTab("Parameters", ssmParamsListTable, 0, 1, true).
		AddTab("Param History", ssmParamHistoryTable, 0, 1, true)

	var mainPage = core.NewResizableView(
		paramValueView.TextView, expandItemViewSize,
		tabView, itemsTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SystemManagerDetailsPageView{
		ServicePageView:          serviceView,
		SSMParametersListTable:   ssmParamsListTable,
		SSMParameterHistoryTable: ssmParamHistoryTable,
		TabView:                  tabView,
		app:                      app,
		api:                      api,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	ssmParamsListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{paramValueView.TextView},
			{tabView.GetTabsList(), tabView.GetTabDisplayView()},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *SystemManagerDetailsPageView) initInputCapture() {
	inst.SSMParametersListTable.SetSelectedFunc(func(row, column int) {
		inst.SSMParameterHistoryTable.SetSeletedParameter(inst.SSMParametersListTable.GetSeletedParameter())
		inst.SSMParameterHistoryTable.RefreshHistory(true)
		inst.TabView.SwitchToTab("Param History")
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
		AddAndSwitchToPage("Parameter Store", systemManagersDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
