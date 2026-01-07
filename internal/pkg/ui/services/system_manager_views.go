package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SsmTabName = string

const (
	SsmTabNameParameters       SsmTabName = "Parameters"
	SsmTabNameParameterHistory SsmTabName = "Parameter History"
)

type SystemManagerDetailsPageView struct {
	*core.ServicePageView
	SSMParametersListTable   *tables.SSMParametersListTable
	SSMParameterHistoryTable *tables.SSMParameterHistoryTable
	TabView                  *core.TabViewHorizontal
	serviceCtx               *core.ServiceContext[awsapi.SystemsManagerApi]
}

func NewSystemManagerDetailsPageView(
	ssmParamsListTable *tables.SSMParametersListTable,
	ssmParamHistoryTable *tables.SSMParameterHistoryTable,
	serviceViewCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *SystemManagerDetailsPageView {
	var paramValueView = core.JsonTextView[any]{
		TextView: core.NewSearchableTextView("", serviceViewCtx.AppContext),
		ExtractTextFunc: func(data any) string {
			switch d := data.(type) {
			case types.Parameter:
				return aws.ToString(d.Value)
			case types.ParameterHistory:
				return aws.ToString(d.Value)
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

	var tabView = core.NewTabViewHorizontal(serviceViewCtx.AppContext).
		AddAndSwitchToTab(SsmTabNameParameters, ssmParamsListTable, 0, 1, true).
		AddTab(SsmTabNameParameterHistory, ssmParamHistoryTable, 0, 1, true)

	var mainPage = core.NewResizableView(
		paramValueView.TextView, expandItemViewSize,
		tabView, itemsTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SystemManagerDetailsPageView{
		ServicePageView:          serviceView,
		SSMParametersListTable:   ssmParamsListTable,
		SSMParameterHistoryTable: ssmParamHistoryTable,
		TabView:                  tabView,
		serviceCtx:               serviceViewCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	ssmParamsListTable.ErrorMessageCallback = errorHandler

	view.InitViewNavigation(
		[][]core.View{
			{paramValueView.TextView},
			{tabView.GetTabDisplayView()},
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

func NewSystemManagerHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xFF5AAD))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		api        = awsapi.NewSystemsManagerApi(appCtx.Logger)
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		systemManagersDetailsView = NewSystemManagerDetailsPageView(
			tables.NewSSMParametersListTable(serviceCtx),
			tables.NewSSMParameterHistoryTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(SYSTEMS_MANAGER), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Parameter Store", systemManagersDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
