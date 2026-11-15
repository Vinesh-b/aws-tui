package route53

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newTabsPage is the tabs over the list, as the pages of WAF have them
func newTabsPage(
	tabView *core.TabViewHorizontal, list core.View, appCtx *core.AppContext,
) *core.ServicePageView {
	const tabViewSize = 5000
	const listSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		list, listSize,
		tview.FlexRow,
		appCtx,
	)
	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{list},
		},
	)
	return serviceView
}

const (
	HostedZoneTabNameDetails = "Details"
	HostedZoneTabNameRecords = "Records"
)

// HostedZonesPageView is the hosted zones, with the details and records of the
// one selected in tabs over them
type HostedZonesPageView struct {
	*core.ServicePageView
	HostedZonesTable *HostedZonesTable
	DetailsTable     *HostedZoneDetailsTable
	RecordsTable     *RecordsTable
	TabView          *core.TabViewHorizontal
}

func NewHostedZonesPageView(
	hostedZonesTable *HostedZonesTable,
	detailsTable *HostedZoneDetailsTable,
	recordsTable *RecordsTable,
	serviceCtx *core.ServiceContext[awsapi.Route53Api],
) *HostedZonesPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(HostedZoneTabNameDetails, detailsTable, 0, 1, true).
		AddTab(HostedZoneTabNameRecords, recordsTable, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		recordsTable.SetVisible(tabName == HostedZoneTabNameRecords)
	})

	var serviceView = newTabsPage(tabView, hostedZonesTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	hostedZonesTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler
	recordsTable.ErrorMessageCallback = errorHandler

	// What the list has is shown at once, but the rest is only loaded for the
	// hosted zone the selection stops on
	hostedZonesTable.SetSelectionChangedFunc(func(row, column int) {
		var zone = hostedZonesTable.GetSelectedZone()
		detailsTable.SetZone(zone)
		recordsTable.SetZone(zone)
	})
	hostedZonesTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshDetails()
		recordsTable.Load()
	})

	return &HostedZonesPageView{
		ServicePageView:  serviceView,
		HostedZonesTable: hostedZonesTable,
		DetailsTable:     detailsTable,
		RecordsTable:     recordsTable,
		TabView:          tabView,
	}
}

const (
	HealthCheckTabNameDetails = "Details"
	HealthCheckTabNameStatus  = "Status"
)

// HealthChecksPageView is the health checks, with the details and status of
// the one selected in tabs over them
type HealthChecksPageView struct {
	*core.ServicePageView
	HealthChecksTable *HealthChecksTable
	DetailsTable      *HealthCheckDetailsTable
	StatusTable       *HealthCheckStatusTable
	TabView           *core.TabViewHorizontal
}

func NewHealthChecksPageView(
	healthChecksTable *HealthChecksTable,
	detailsTable *HealthCheckDetailsTable,
	statusTable *HealthCheckStatusTable,
	serviceCtx *core.ServiceContext[awsapi.Route53Api],
) *HealthChecksPageView {
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(HealthCheckTabNameDetails, detailsTable, 0, 1, true).
		AddTab(HealthCheckTabNameStatus, statusTable, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		statusTable.SetVisible(tabName == HealthCheckTabNameStatus)
	})

	var serviceView = newTabsPage(tabView, healthChecksTable, serviceCtx.AppContext)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	healthChecksTable.ErrorMessageCallback = errorHandler
	statusTable.ErrorMessageCallback = errorHandler

	healthChecksTable.SetSelectionChangedFunc(func(row, column int) {
		var check = healthChecksTable.GetSelectedHealthCheck()
		detailsTable.SetHealthCheck(check)
		statusTable.SetHealthCheck(check)
	})
	healthChecksTable.SetSelectionSettledFunc(func(row, column int) {
		statusTable.Load()
	})

	return &HealthChecksPageView{
		ServicePageView:   serviceView,
		HealthChecksTable: healthChecksTable,
		DetailsTable:      detailsTable,
		StatusTable:       statusTable,
		TabView:           tabView,
	}
}

// The pages of the service
const (
	route53PageHostedZones = iota
	route53PageHealthChecks
)

// route53ArnKind is the kind of resource the ARN names, which is empty for
// ARNs that aren't of Route 53
func route53ArnKind(resource arn.ARN) string {
	var named, err = awsapi.ParseRoute53Arn(resource.String())
	if err != nil {
		return ""
	}
	return named.Kind
}

func isOpenableRoute53Resource(resource arn.ARN) bool {
	switch route53ArnKind(resource) {
	case "hostedzone", "healthcheck":
		return true
	}
	return false
}

func NewRoute53HomeView(appCtx *core.AppContext, api awsapi.Route53Api) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x5A30B5))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		hostedZonesView = NewHostedZonesPageView(
			NewHostedZonesTable(serviceCtx),
			NewHostedZoneDetailsTable(serviceCtx),
			NewRecordsTable(serviceCtx),
			serviceCtx,
		)
		healthChecksView = NewHealthChecksPageView(
			NewHealthChecksTable(serviceCtx),
			NewHealthCheckDetailsTable(serviceCtx),
			NewHealthCheckStatusTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.ROUTE53), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Hosted Zones", hostedZonesView, true).
		AddPage("Health Checks", healthChecksView, true, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("route53", isOpenableRoute53Resource, func(resource arn.ARN) {
		var named, _ = awsapi.ParseRoute53Arn(resource.String())
		switch named.Kind {
		case "hostedzone":
			serviceRootView.ChangePage(route53PageHostedZones, nil)
			hostedZonesView.HostedZonesTable.Open(named.Id)
		case "healthcheck":
			serviceRootView.ChangePage(route53PageHealthChecks, nil)
			healthChecksView.HealthChecksTable.Open(named.Id)
		}
	})

	return serviceRootView
}
