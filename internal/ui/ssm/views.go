package ssm

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SsmTabName = string

const (
	SsmTabNameParameters       SsmTabName = "Parameters"
	SsmTabNameParameterHistory SsmTabName = "Parameter History"
	SsmTabNameValue            SsmTabName = "Value"
	SsmTabNameDetails          SsmTabName = "Details"
)

type ParametersPageView struct {
	*core.ServicePageView
	SSMParametersListTable   *SSMParametersListTable
	SSMParameterHistoryTable *SSMParameterHistoryTable
	ParameterDetailsTable    *ParameterDetailsTable
	ParameterValueView       *core.JsonTextView[any]
	DetailsTabView           *core.TabViewHorizontal
	TabView                  *core.TabViewHorizontal
	serviceCtx               *core.ServiceContext[awsapi.SystemsManagerApi]
}

func NewParametersPageView(
	parameterDetailsTable *ParameterDetailsTable,
	ssmParamsListTable *SSMParametersListTable,
	ssmParamHistoryTable *SSMParameterHistoryTable,
	serviceCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *ParametersPageView {
	var paramValueView = &core.JsonTextView[any]{
		TextView: core.NewSearchableTextView("", serviceCtx.AppContext),
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

	var detailsTabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SsmTabNameValue, paramValueView.TextView, 0, 1, true).
		AddTab(SsmTabNameDetails, parameterDetailsTable, 0, 1, true)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SsmTabNameParameters, ssmParamsListTable, 0, 1, true).
		AddTab(SsmTabNameParameterHistory, ssmParamHistoryTable, 0, 1, true)

	const detailsSize = 25
	const itemsTableSize = 75

	var mainPage = core.NewResizableView(
		detailsTabView, detailsSize,
		tabView, itemsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{detailsTabView},
			{tabView},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	ssmParamsListTable.ErrorMessageCallback = errorHandler
	ssmParamHistoryTable.ErrorMessageCallback = errorHandler

	return &ParametersPageView{
		ServicePageView:          serviceView,
		SSMParametersListTable:   ssmParamsListTable,
		SSMParameterHistoryTable: ssmParamHistoryTable,
		ParameterDetailsTable:    parameterDetailsTable,
		ParameterValueView:       paramValueView,
		DetailsTabView:           detailsTabView,
		TabView:                  tabView,
		serviceCtx:               serviceCtx,
	}
}

func (inst *ParametersPageView) InitInputCapture() {
	inst.SSMParametersListTable.SetSelectionChangedFunc(func(row, column int) {
		var parameter = inst.SSMParametersListTable.GetSelectedParameter()
		inst.ParameterValueView.SetText(parameter)
		inst.ParameterDetailsTable.RefreshDetails(parameter)
	})

	inst.SSMParameterHistoryTable.SetSelectionChangedFunc(func(row, column int) {
		inst.ParameterValueView.SetText(inst.SSMParameterHistoryTable.GetSelectedHistory())
	})

	var loadedTabs = map[int]bool{}
	var lastSelection = ""
	var tabChangeFunc = func(tabName string, index int) {
		var selectedParam = inst.SSMParametersListTable.GetSelectedParameter()

		if paramArn := aws.ToString(selectedParam.ARN); paramArn != lastSelection {
			lastSelection = paramArn
			utils.ClearMap(loadedTabs)
		}

		// Only automaticaly load new data on first change
		if loadedTabs[index] {
			return
		}

		switch tabName {
		case SsmTabNameParameterHistory:
			inst.SSMParameterHistoryTable.SetSelectedParameter(selectedParam)
			inst.SSMParameterHistoryTable.RefreshHistory(true)
		}

		loadedTabs[index] = true
	}

	inst.SSMParametersListTable.SetSelectedFunc(func(row, column int) {
		var tabName, index = inst.TabView.GetCurrentTab()
		tabChangeFunc(tabName, index)
		inst.TabView.SwitchToTab(SsmTabNameParameterHistory)
	})

	inst.TabView.SetOnTabChangeFunc(tabChangeFunc)
}

func NewSystemManagerHomeView(appCtx *core.AppContext, api awsapi.SystemsManagerApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xFF5AAD))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var parametersView = NewParametersPageView(
		NewParameterDetailsTable(serviceCtx),
		NewSSMParametersListTable(serviceCtx),
		NewSSMParameterHistoryTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(core.SYSTEMS_MANAGER), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Parameter Store", parametersView, true)

	serviceRootView.InitPageNavigation()

	parametersView.InitInputCapture()

	return serviceRootView
}
