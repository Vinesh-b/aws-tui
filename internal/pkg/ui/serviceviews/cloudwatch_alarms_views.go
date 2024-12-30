package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AlarmsDetailsPageView struct {
	*core.ServicePageView
	AlarmsTable  *AlarmListTable
	HistoryTable *AlarmHistoryTable
	DetailsTable *AlarmDetailsTable
	app          *tview.Application
	api          *awsapi.CloudWatchAlarmsApi
}

func NewAlarmsDetailsPageView(
	alarmListTable *AlarmListTable,
	alarmHistoryTable *AlarmHistoryTable,
	alarmDetailsTable *AlarmDetailsTable,
	app *tview.Application,
	api *awsapi.CloudWatchAlarmsApi,
	logger *log.Logger,
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

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, false)

	serviceView.InitViewNavigation(
		[]core.View{
			alarmListTable,
			alarmHistoryTable,
			alarmDetailsTable,
		},
	)

	var errorHandler = func(text string) {
		serviceView.SetAndDisplayError(text)
	}

	alarmListTable.ErrorMessageHandler = errorHandler
	alarmHistoryTable.ErrorMessageHandler = errorHandler
	alarmDetailsTable.ErrorMessageHandler = errorHandler

	return &AlarmsDetailsPageView{
		ServicePageView: serviceView,
		AlarmsTable:     alarmListTable,
		DetailsTable:    alarmDetailsTable,
		HistoryTable:    alarmHistoryTable,
		app:             app,
		api:             api,
	}

}

func (inst *AlarmsDetailsPageView) InitInputCapture() {
	var refreshDetails = func() {
		var name = inst.AlarmsTable.GetSelectedAlarm()
		inst.DetailsTable.SetSelectedAlarm(name)
		inst.DetailsTable.RefreshDetails()
		inst.HistoryTable.SetSelectedAlarm(name)
		inst.HistoryTable.RefreshHistory(true)
	}

	inst.AlarmsTable.SetSelectedFunc(func(row, column int) {
		refreshDetails()
	})

	inst.AlarmsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.AlarmsTable.RefreshAlarms(true)
			refreshDetails()
		}
		return event
	})

	inst.HistoryTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			refreshDetails()
		case tcell.KeyCtrlN:
			inst.HistoryTable.RefreshHistory(false)
		}
		return event
	})
}

func NewAlarmsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchAlarmsApi(config, logger)
	var alarmsDetailsView = NewAlarmsDetailsPageView(
		NewAlarmListTable(app, api, logger),
		NewAlarmHistoryTable(app, api, logger),
		NewAlarmDetailsTable(app, api, logger),
		app, api, logger,
	)
	alarmsDetailsView.InitInputCapture()

	var pages = tview.NewPages().
		AddAndSwitchToPage("Alarms", alarmsDetailsView, true)

	var orderedPages = []string{
		"Alarms",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDWATCH_ALARMS), pages, orderedPages).Init()

	return serviceRootView
}
