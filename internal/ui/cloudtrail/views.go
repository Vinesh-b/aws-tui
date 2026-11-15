package cloudtrail

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newDetailsPage is the details over the list, as the pages of the other
// services are laid out
func newDetailsPage(
	details core.View, list core.View, appCtx *core.AppContext,
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
	serviceView.InitViewNavigation(
		[][]core.View{
			{details},
			{list},
		},
	)
	return serviceView
}

type EventsPageView struct {
	*core.ServicePageView
	EventsTable  *EventsTable
	DetailsTable *EventDetailsTable
}

func NewEventsPageView(
	eventsTable *EventsTable,
	detailsTable *EventDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.CloudTrailApi],
) *EventsPageView {
	var serviceView = newDetailsPage(detailsTable, eventsTable, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	eventsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	// The history holds all of each event, so the details are shown at once
	eventsTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetEvent(eventsTable.GetSelectedEvent())
	})

	return &EventsPageView{
		ServicePageView: serviceView,
		EventsTable:     eventsTable,
		DetailsTable:    detailsTable,
	}
}

type TrailsPageView struct {
	*core.ServicePageView
	TrailsTable  *TrailsTable
	DetailsTable *TrailDetailsTable
}

func NewTrailsPageView(
	trailsTable *TrailsTable,
	detailsTable *TrailDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.CloudTrailApi],
) *TrailsPageView {
	var serviceView = newDetailsPage(detailsTable, trailsTable, serviceCtx.AppContext)
	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	trailsTable.ErrorMessageCallback = errorHandler
	detailsTable.ErrorMessageCallback = errorHandler

	// How the trail is set up is in the list, so it is shown at once, but its
	// status is only loaded for the trail the selection stops on
	trailsTable.SetSelectionChangedFunc(func(row, column int) {
		detailsTable.SetTrail(trailsTable.GetSelectedTrail())
	})
	trailsTable.SetSelectionSettledFunc(func(row, column int) {
		detailsTable.RefreshStatus()
	})

	return &TrailsPageView{
		ServicePageView: serviceView,
		TrailsTable:     trailsTable,
		DetailsTable:    detailsTable,
	}
}

// The pages of the service
const (
	cloudTrailPageEvents = iota
	cloudTrailPageTrails
)

// isOpenableTrail is true for the trails the trails page shows, whose
// resource is "trail/<name>"
func isOpenableTrail(resource arn.ARN) bool {
	var name, found = strings.CutPrefix(resource.Resource, "trail/")
	return found && len(name) > 0
}

func NewCloudTrailHomeView(appCtx *core.AppContext, api awsapi.CloudTrailApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xB0084D))
	defer appCtx.Theme.ResetGlobalStyle()

	var (
		serviceCtx = core.NewServiceViewContext(appCtx, api)

		eventsView = NewEventsPageView(
			NewEventsTable(serviceCtx),
			NewEventDetailsTable(appCtx),
			serviceCtx,
		)
		trailsView = NewTrailsPageView(
			NewTrailsTable(serviceCtx),
			NewTrailDetailsTable(serviceCtx),
			serviceCtx,
		)
	)

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDTRAIL), appCtx)

	serviceRootView.
		AddAndSwitchToPage("Events", eventsView, true).
		AddPage("Trails", trailsView, true, true)

	serviceRootView.InitPageNavigation()

	serviceRootView.SetResourceOpener("cloudtrail", isOpenableTrail, func(resource arn.ARN) {
		serviceRootView.ChangePage(cloudTrailPageTrails, nil)
		trailsView.TrailsTable.Open(resource.String())
	})

	return serviceRootView
}
