package services

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AlarmsDetailsPageView struct {
	*core.ServicePageView
	AlarmsTable  *tables.AlarmListTable
	HistoryTable *tables.AlarmHistoryTable
	DetailsTable *tables.AlarmDetailsTable
	serviceCtx   *core.ServiceContext[awsapi.CloudWatchAlarmsApi]
}

func NewAlarmsDetailsPageView(
	alarmListTable *tables.AlarmListTable,
	alarmHistoryTable *tables.AlarmHistoryTable,
	alarmDetailsTable *tables.AlarmDetailsTable,
	serviceContext *core.ServiceContext[awsapi.CloudWatchAlarmsApi],
) *AlarmsDetailsPageView {
	const alarmsTableSize = 3500
	const alarmHistorySize = 3000

	var resizableView = core.NewResizableView(
		alarmHistoryTable, alarmHistorySize,
		alarmListTable, alarmsTableSize,
		tview.FlexRow,
	)

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(alarmDetailsTable, 14, 0, false).
		AddItem(resizableView, 0, 1, true)

	var serviceView = core.NewServicePageView(serviceContext.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, false)

	serviceView.InitViewNavigation(
		[][]core.View{
			{alarmDetailsTable},
			{alarmHistoryTable},
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

	inst.HistoryTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			refreshDetails()
		case core.APP_KEY_BINDINGS.LoadMoreData:
			inst.HistoryTable.RefreshHistory(false)
		}
		return event
	})
}

func NewAlarmsHomeView(appCtx *core.AppContext) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchAlarmsApi(*appCtx.Config, appCtx.Logger)
	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var alarmsDetailsView = NewAlarmsDetailsPageView(
		tables.NewAlarmListTable(serviceCtx),
		tables.NewAlarmHistoryTable(serviceCtx),
		tables.NewAlarmDetailsTable(serviceCtx),
		serviceCtx,
	)
	alarmsDetailsView.InitInputCapture()

	var serviceRootView = core.NewServiceRootView(string(CLOUDWATCH_ALARMS), appCtx)

	serviceRootView.AddAndSwitchToPage("Alarms", alarmsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
