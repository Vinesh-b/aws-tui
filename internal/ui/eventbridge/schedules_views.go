package eventbridge

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/rivo/tview"
)

const (
	SchedulesTabNameDetails EventBridgeTabName = "Details"
	SchedulesTabNameTarget  EventBridgeTabName = "Target"
)

type SchedulesPageView struct {
	*core.ServicePageView
	ScheduleListTable    *ScheduleListTable
	ScheduleDetailsTable *ScheduleDetailsTable
	TargetView           *core.JsonTextView[scheduler.GetScheduleOutput]
	TabView              *core.TabViewHorizontal
	serviceCtx           *core.ServiceContext[awsapi.SchedulerApi]
}

func NewSchedulesPageView(
	scheduleListTable *ScheduleListTable,
	scheduleDetailsTable *ScheduleDetailsTable,
	serviceCtx *core.ServiceContext[awsapi.SchedulerApi],
) *SchedulesPageView {
	var targetView = &core.JsonTextView[scheduler.GetScheduleOutput]{
		TextView:        core.NewSearchableTextView("", serviceCtx.AppContext),
		ExtractTextFunc: extractScheduleTargetJson,
	}
	targetView.SetTitle("Target")

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(SchedulesTabNameDetails, scheduleDetailsTable, 0, 1, true).
		AddTab(SchedulesTabNameTarget, targetView.TextView, 0, 1, true)

	const detailsViewSize = 5000
	const tableViewSize = 5000

	var mainPage = core.NewResizableView(
		tabView, detailsViewSize,
		scheduleListTable, tableViewSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SchedulesPageView{
		ServicePageView:      serviceView,
		ScheduleListTable:    scheduleListTable,
		ScheduleDetailsTable: scheduleDetailsTable,
		TargetView:           targetView,
		TabView:              tabView,
		serviceCtx:           serviceCtx,
	}

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	scheduleListTable.ErrorMessageCallback = errorHandler
	scheduleDetailsTable.ErrorMessageCallback = errorHandler
	targetView.TextView.ErrorMessageCallback = errorHandler

	scheduleDetailsTable.SetOnLoadedFunc(func(schedule scheduler.GetScheduleOutput) {
		targetView.SetText(schedule)
	})

	// The views of a deleted schedule are emptied, the reloaded list then
	// shows whatever is selected next
	scheduleListTable.SetOnScheduleDeleted(func(id scheduleId) {
		if id != scheduleDetailsTable.requested {
			return
		}
		scheduleDetailsTable.ClearDetails()
	})

	view.InitViewNavigation(
		[][]core.View{
			{tabView},
			{scheduleListTable},
		},
	)
	view.initInputCapture()

	return view
}

func (inst *SchedulesPageView) initInputCapture() {
	inst.ScheduleListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		var id = scheduleIdOf(inst.ScheduleListTable.GetSelectedSchedule())
		if id.isEmpty() {
			return
		}
		inst.ScheduleDetailsTable.RefreshDetails(id)
	})
}
