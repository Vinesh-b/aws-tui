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
	SsmTabNameUpdate           SsmTabName = "Update Value"
	SsmTabNameCreate           SsmTabName = "Create Parameter"
)

type ParametersPageView struct {
	*core.ServicePageView
	SSMParametersListTable   *SSMParametersListTable
	SSMParameterHistoryTable *SSMParameterHistoryTable
	ParameterDetailsTable    *ParameterDetailsTable
	ParameterValueView       *core.JsonTextView[any]
	UpdateValueView          *ParameterUpdateValuePageView
	CreateParameterView      *ParameterCreatePageView
	DetailsTabView           *core.TabViewHorizontal
	TabView                  *core.TabViewHorizontal
	serviceCtx               *core.ServiceContext[awsapi.SystemsManagerApi]
	decrypted                decryptedValues
	// shownValue is the parameter or the version whose value is shown, to show
	// it again once it is decrypted. It is only used on the UI goroutine.
	shownValue any
}

func NewParametersPageView(
	parameterDetailsTable *ParameterDetailsTable,
	updateValueView *ParameterUpdateValuePageView,
	createParameterView *ParameterCreatePageView,
	ssmParamsListTable *SSMParametersListTable,
	ssmParamHistoryTable *SSMParameterHistoryTable,
	serviceCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *ParametersPageView {
	var decrypted = decryptedValues{}
	var paramValueView = &core.JsonTextView[any]{
		TextView: core.NewSearchableTextView("", serviceCtx.AppContext),
		ExtractTextFunc: func(data any) string {
			switch d := data.(type) {
			case types.Parameter:
				return decrypted.text(aws.ToString(d.Name), d.Version, d.Type, aws.ToString(d.Value))
			case types.ParameterHistory:
				return decrypted.text(aws.ToString(d.Name), d.Version, d.Type, aws.ToString(d.Value))
			}
			return ""
		},
	}
	paramValueView.SetTitle("Value")

	var detailsTabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SsmTabNameValue, paramValueView.TextView, 0, 1, true).
		AddTab(SsmTabNameDetails, parameterDetailsTable, 0, 1, true).
		AddTab(SsmTabNameUpdate, updateValueView, 0, 1, true).
		AddTab(SsmTabNameCreate, createParameterView, 0, 1, true)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SsmTabNameParameters, ssmParamsListTable, 0, 1, true).
		AddTab(SsmTabNameParameterHistory, ssmParamHistoryTable, 0, 1, true)

	const detailsSize = 40
	const itemsTableSize = 60

	var mainPage = core.NewResizableView(
		detailsTabView, detailsSize,
		tabView, itemsTableSize,
		tview.FlexRow,
		serviceCtx.AppContext,
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
		UpdateValueView:          updateValueView,
		CreateParameterView:      createParameterView,
		DetailsTabView:           detailsTabView,
		TabView:                  tabView,
		serviceCtx:               serviceCtx,
		decrypted:                decrypted,
	}
}

// showValue shows the value of the parameter or of the version of one, or how
// to see it if it is secure and was not decrypted yet
func (inst *ParametersPageView) showValue(data any) {
	inst.shownValue = data
	inst.ParameterValueView.SetText(data)
}

// decryptValue fetches the value of the version of a secure parameter, which
// is only done when it is asked for, and shows it if that version is still the
// one shown. It is false if there was nothing to fetch, as the parameter is
// not secure or its value was already decrypted.
func (inst *ParametersPageView) decryptValue(name string, version int64, paramType types.ParameterType) bool {
	var key = valueVersion{name, version}
	if _, shown := inst.decrypted[key]; !isSecure(paramType) || shown || name == "" {
		return false
	}

	var value string
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		value, apiErr = inst.serviceCtx.Api.GetParameterValue(inst.serviceCtx.Ctx, name, version)
	})

	dataLoader.AsyncUpdateView(inst.ParameterValueView.TextView.Box, func() {
		if apiErr != nil {
			inst.DisplayMessage(core.ErrorPrompt, "%s", apiErr.Error())
			return
		}
		inst.decrypted[key] = value
		// Another row may have been selected while this one was loading,
		// which is then shown as it is
		inst.ParameterValueView.SetText(inst.shownValue)
	})
	return true
}

func (inst *ParametersPageView) InitInputCapture() {
	inst.SSMParametersListTable.SetSelectionChangedFunc(func(row, column int) {
		var parameter = inst.SSMParametersListTable.GetSelectedParameter()
		inst.showValue(parameter)
		inst.ParameterDetailsTable.RefreshDetails(parameter)
		inst.UpdateValueView.SetSelectedParameter(parameter)
	})

	// The new parameter is in the listing once it is loaded again
	inst.CreateParameterView.SetOnParameterCreated(func(name string) {
		inst.SSMParametersListTable.RefreshParameters("/", true)
	})

	// The listing has the old value until it is loaded again
	inst.UpdateValueView.SetOnValueUpdated(func(parameter types.Parameter) {
		inst.SSMParametersListTable.RefreshParameters("/", true)
	})

	inst.SSMParameterHistoryTable.SetSelectionChangedFunc(func(row, column int) {
		inst.showValue(inst.SSMParameterHistoryTable.GetSelectedHistory())
	})

	// A secure value of a version is only decrypted when it is asked for
	inst.SSMParameterHistoryTable.SetSelectedFunc(func(row, column int) {
		var version = inst.SSMParameterHistoryTable.GetSelectedHistory()
		inst.decryptValue(aws.ToString(version.Name), version.Version, version.Type)
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

	// A secure value is only decrypted when it is asked for, which the first
	// Enter on its parameter does. The history is opened by the next one, and
	// straight away for the other parameters.
	inst.SSMParametersListTable.SetSelectedFunc(func(row, column int) {
		var parameter = inst.SSMParametersListTable.GetSelectedParameter()
		if inst.decryptValue(aws.ToString(parameter.Name), parameter.Version, parameter.Type) {
			return
		}

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
		NewParameterUpdateValuePageView(serviceCtx),
		NewParameterCreatePageView(serviceCtx),
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
