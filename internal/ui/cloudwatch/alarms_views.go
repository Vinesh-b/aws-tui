package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AlarmsTabName = string

const (
	AlarmTabDetails AlarmsTabName = "Details"
	AlarmTabHistory AlarmsTabName = "History"
)

type AlarmsDetailsPageView struct {
	*core.ServicePageView
	AlarmsTable  *AlarmListTable
	HistoryTable *AlarmHistoryTable
	DetailsTable *AlarmDetailsTable
	serviceCtx   *core.ServiceContext[awsapi.CloudWatchAlarmsApi]
}

func NewAlarmsDetailsPageView(
	alarmListTable *AlarmListTable,
	alarmHistoryTable *AlarmHistoryTable,
	alarmDetailsTable *AlarmDetailsTable,
	serviceContext *core.ServiceContext[awsapi.CloudWatchAlarmsApi],
) *AlarmsDetailsPageView {
	const alarmTabViewSize = 3000
	const alarmsTableSize = 7000

	var tabView = core.NewTabViewHorizontal(serviceContext.AppContext).
		AddAndSwitchToTab(AlarmTabDetails, alarmDetailsTable, 0, 1, true).
		AddTab(AlarmTabHistory, alarmHistoryTable, 0, 1, true)

	var resizableView = core.NewResizableView(
		tabView, alarmTabViewSize,
		alarmListTable, alarmsTableSize,
		tview.FlexRow,
	)

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resizableView, 0, 1, true)

	var serviceView = core.NewServicePageView(serviceContext.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, false)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{alarmListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	alarmListTable.ErrorMessageCallback = errorHandler
	alarmHistoryTable.ErrorMessageCallback = errorHandler
	alarmDetailsTable.ErrorMessageCallback = errorHandler

	return &AlarmsDetailsPageView{
		ServicePageView: serviceView,
		AlarmsTable:     alarmListTable,
		DetailsTable:    alarmDetailsTable,
		HistoryTable:    alarmHistoryTable,
		serviceCtx:      serviceContext,
	}

}

func (inst *AlarmsDetailsPageView) InitInputCapture() {
	var refreshDetails = func() {
		var alarm = inst.AlarmsTable.GetSelectedAlarm()
		inst.DetailsTable.RefreshDetails(alarm)
		var alarmName = inst.AlarmsTable.GetSelectedAlarmName()
		inst.HistoryTable.SetSelectedAlarm(alarmName)
		inst.HistoryTable.RefreshHistory(true)
	}

	inst.AlarmsTable.SetSelectedFunc(func(row, column int) {
		refreshDetails()
	})

	inst.HistoryTable.SetLoadMoreFunc(func() { inst.HistoryTable.RefreshHistory(false) })
	inst.HistoryTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			refreshDetails()
			return nil
		}
		return event
	})
}

func NewAlarmsHomeView(appCtx *core.AppContext, api awsapi.CloudWatchAlarmsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var alarmsDetailsView = NewAlarmsDetailsPageView(
		NewAlarmListTable(serviceCtx),
		NewAlarmHistoryTable(serviceCtx),
		NewAlarmDetailsTable(serviceCtx),
		serviceCtx,
	)
	alarmsDetailsView.InitInputCapture()

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDWATCH_ALARMS), appCtx)

	serviceRootView.AddAndSwitchToPage("Alarms", alarmsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
