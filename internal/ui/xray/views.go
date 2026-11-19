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
	FilterView   *TraceFilterView
	GroupView    *TraceGroupView
	TraceIdView  *TraceIdView
	DetailsTable *TraceDetailsTable
}

func NewTracesPageView(
	tracesTable *TracesTable,
	detailsTable *TraceDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.XRayApi],
) *TracesPageView {
	// The range is picked, and the filter entered, over the list of traces,
	// beside the ID of a trace to open that may not be listed
	var rangeView = NewTraceRangeView(tracesTable, serviceCtx.AppContext)
	var filterView = NewTraceFilterView(tracesTable, serviceCtx.AppContext)
	var groupView = NewTraceGroupView(filterView, serviceCtx)
	var traceIdView = NewTraceIdView(detailsTable, serviceCtx.AppContext)
	var list = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rangeView, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(groupView, traceGroupViewWidth, 0, false).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(filterView, 0, 1, false).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(traceIdView, traceIdViewWidth, 0, false),
			1, 0, false).
		AddItem(tracesTable, 0, 1, true)
	var serviceView = newDetailsPage(detailsTable, list, [][]core.View{
		{detailsTable},
		rangeView.NavigationRow(),
		{groupView, filterView, traceIdView},
		{tracesTable},
	}, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	tracesTable.ErrorMessageCallback = errorHandler
	rangeView.ErrorMessageCallback = errorHandler
	traceIdView.ErrorMessageCallback = errorHandler
	groupView.ErrorMessageCallback = errorHandler
	// The groups are loaded with the traces, when they are loaded again
	tracesTable.SetResetFunc(groupView.RefreshGroups)
	detailsTable.ErrorMessageCallback = errorHandler

	// The summary of the trace is in the list, so it is shown at once, but its
	// segments are only loaded for the trace the selection stops on
	tracesTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetTrace(tracesTable.GetSelectedTrace())
	})
	tracesTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshSegments()
	})

	// The logs are of the trace shown, which is the one selected or the one
	// opened by its ID, and are found by its segments too when they are loaded.
	// They are queried from the details as well, as an opened trace may not be
	// listed, and an empty list takes no keys but reset and the range.
	var queryLogs = func(errorCallback func(text string, a ...any)) func() {
		return func() {
			var query, err = traceLogsQuery(detailsTable.Summary(), detailsTable.LoadedTrace())
			if err == nil {
				err = serviceCtx.AppContext.OpenLogsQuery(query)
			}
			if err != nil {
				errorCallback("%s", err.Error())
			}
		}
	}
	tracesTable.SetQueryLogsFunc(queryLogs(func(text string, a ...any) { tracesTable.ErrorMessageCallback(text, a...) }))
	detailsTable.SetQueryLogsFunc(queryLogs(func(text string, a ...any) { detailsTable.ErrorMessageCallback(text, a...) }))

	return &TracesPageView{
		ServicePageView: serviceView,
		TracesTable:     tracesTable,
		RangeView:       rangeView,
		FilterView:      filterView,
		GroupView:       groupView,
		TraceIdView:     traceIdView,
		DetailsTable:    detailsTable,
	}
}

type TraceAnalyticsPageView struct {
	*core.ServicePageView
	AnalyticsTable *TraceAnalyticsTable
	DetailsTable   *TraceGroupDetailsTable
}

// NewTraceAnalyticsPageView groups the traces of the traces table, again each
// time they are loaded
func NewTraceAnalyticsPageView(
	analyticsTable *TraceAnalyticsTable,
	detailsTable *TraceGroupDetailsTable,
	tracesTable *TracesTable,
	serviceCtx *core.ServiceContext[awsapi.XRayApi],
) *TraceAnalyticsPageView {
	var serviceView = newDetailsPage(detailsTable, analyticsTable, [][]core.View{
		{detailsTable},
		{analyticsTable},
	}, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	analyticsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	analyticsTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetGroup(analyticsTable.GetSelectedGroup())
	})
	tracesTable.SetLoadedChangedFunc(analyticsTable.Refresh)

	return &TraceAnalyticsPageView{
		ServicePageView: serviceView,
		AnalyticsTable:  analyticsTable,
		DetailsTable:    detailsTable,
	}
}

type ServiceMapPageView struct {
	*core.ServicePageView
	ServicesTable *ServiceMapTable
	DetailsTable  *ServiceDetailsTable
}

func NewServiceMapPageView(
	servicesTable *ServiceMapTable,
	detailsTable *ServiceDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.XRayApi],
) *ServiceMapPageView {
	var serviceView = newDetailsPage(detailsTable, servicesTable, [][]core.View{
		{detailsTable},
		{servicesTable},
	}, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	servicesTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	// The map holds all of each service and their calls, so the details are
	// shown at once
	servicesTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetService(servicesTable.GetSelectedService(), servicesTable.Services())
	})

	return &ServiceMapPageView{
		ServicePageView: serviceView,
		ServicesTable:   servicesTable,
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
	xRayPageAnalytics
	xRayPageServiceMap
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
		analyticsView = NewTraceAnalyticsPageView(
			NewTraceAnalyticsTable(tracesView.TracesTable, appCtx),
			NewTraceGroupDetailsTable(appCtx),
			tracesView.TracesTable,
			serviceCtx,
		)
		serviceMapView = NewServiceMapPageView(
			NewServiceMapTable(serviceCtx),
			NewServiceDetailsTable(appCtx),
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
		AddPage("Analytics", analyticsView, true, true).
		AddPage("Service Map", serviceMapView, true, true).
		AddPage("Sampling Rules", rulesView, true, true)

	serviceRootView.InitPageNavigation()

	// A group shows its traces, found with the filter for it
	analyticsView.AnalyticsTable.SetSelectedFunc(func(row, column int) {
		var filter, err = analyticsView.AnalyticsTable.GetSelectedGroup().filterExpression()
		if err != nil {
			analyticsView.AnalyticsTable.ErrorMessageCallback("%s", err.Error())
			return
		}
		tracesView.FilterView.SetText(filter)
		tracesView.FilterView.ApplyFilter()
		serviceRootView.ChangePage(xRayPageTraces, nil)
	})

	serviceRootView.SetResourceOpener("xray", isOpenableSamplingRule, func(resource arn.ARN) {
		serviceRootView.ChangePage(xRayPageSamplingRules, nil)
		rulesView.RulesTable.Open(resource.String())
	})

	return serviceRootView
}
