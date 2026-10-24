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
)

type SystemManagerDetailsPageView struct {
	*core.ServicePageView
	SSMParametersListTable   *SSMParametersListTable
	SSMParameterHistoryTable *SSMParameterHistoryTable
	TabView                  *core.TabViewHorizontal
	serviceCtx               *core.ServiceContext[awsapi.SystemsManagerApi]
}

func NewSystemManagerDetailsPageView(
	ssmParamsListTable *SSMParametersListTable,
	ssmParamHistoryTable *SSMParameterHistoryTable,
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
		paramValueView.SetText(ssmParamsListTable.GetSelectedParameter())
	})

	ssmParamHistoryTable.SetSelectionChangedFunc(func(row, column int) {
		paramValueView.SetText(ssmParamHistoryTable.GetSelectedHistory())
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

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		systemManagersDetailsView = NewSystemManagerDetailsPageView(
			NewSSMParametersListTable(serviceCtx),
			NewSSMParameterHistoryTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.SYSTEMS_MANAGER), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Parameter Store", systemManagersDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
