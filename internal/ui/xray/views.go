package xray

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newDetailsPage is the details over the list, as the pages of the other
// services are laid out, moved between by the rows of views of navigation
func newDetailsPage(
	details core.View, list core.View, navigation [][]core.View, appCtx *core.AppContext,
) *core.ServicePageView {
	const detailsSize = 3000
	const listSize = 5000

	var mainPage = core.NewResizableView(
		details, detailsSize,
		list, listSize,
		tview.FlexRow,
		appCtx,
	)
	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(navigation)
	return serviceView
}

type TracesPageView struct {
	*core.ServicePageView
	TracesTable  *TracesTable
	RangeView    *TraceRangeView
	DetailsTable *TraceDetailsTable
}

func NewTracesPageView(
	tracesTable *TracesTable,
	detailsTable *TraceDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.XRayApi],
) *TracesPageView {
	// The range is picked over the list of traces
	var rangeView = NewTraceRangeView(tracesTable, serviceCtx.AppContext)
	var list = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rangeView, 1, 0, false).
		AddItem(tracesTable, 0, 1, true)
	var serviceView = newDetailsPage(detailsTable, list, [][]core.View{
		{detailsTable},
		rangeView.NavigationRow(),
		{tracesTable},
	}, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	tracesTable.ErrorMessageCallback = errorHandler
	rangeView.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	// The summary of the trace is in the list, so it is shown at once, but its
	// segments are only loaded for the trace the selection stops on
	tracesTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetTrace(tracesTable.GetSelectedTrace())
	})
	tracesTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshSegments()
	})

	return &TracesPageView{
		ServicePageView: serviceView,
		TracesTable:     tracesTable,
		RangeView:       rangeView,
		DetailsTable:    detailsTable,
	}
}

type SamplingRulesPageView struct {
	*core.ServicePageView
	RulesTable   *SamplingRulesTable
	DetailsTable *SamplingRuleDetailsTable
}

func NewSamplingRulesPageView(
	rulesTable *SamplingRulesTable,
	detailsTable *SamplingRuleDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.XRayApi],
) *SamplingRulesPageView {
	var serviceView = newDetailsPage(detailsTable, rulesTable, [][]core.View{
		{detailsTable},
		{rulesTable},
	}, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	rulesTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	// The list holds all of each rule, so the details are shown at once
	rulesTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetRule(rulesTable.GetSelectedRule())
	})

	return &SamplingRulesPageView{
		ServicePageView: serviceView,
		RulesTable:      rulesTable,
		DetailsTable:    detailsTable,
	}
}

// The pages of the service
const (
	xRayPageTraces = iota
	xRayPageSamplingRules
)

// isOpenableSamplingRule is true for the rules the sampling rules page shows,
// whose resource is "sampling-rule/<name>"
func isOpenableSamplingRule(resource arn.ARN) bool {
	var name, found = strings.CutPrefix(resource.Resource, "sampling-rule/")
	return found && len(name) > 0
}

func NewXRayHomeView(appCtx *core.AppContext, api awsapi.XRayApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x8C4FFF))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		tracesView = NewTracesPageView(
			NewTracesTable(serviceCtx),
			NewTraceDetailsTable(serviceCtx),
			serviceCtx,
		)
		rulesView = NewSamplingRulesPageView(
			NewSamplingRulesTable(serviceCtx),
			NewSamplingRuleDetailsTable(appCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.XRAY), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Traces", tracesView, true).
		AddPage("Sampling Rules", rulesView, true, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("xray", isOpenableSamplingRule, func(resource arn.ARN) {
		serviceRootView.ChangePage(xRayPageSamplingRules, nil)
		rulesView.RulesTable.Open(resource.String())
	})

	return serviceRootView
}
