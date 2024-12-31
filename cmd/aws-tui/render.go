package main

import (
	uicore "aws-tui/internal/pkg/ui/core"
	uiviews "aws-tui/internal/pkg/ui/serviceviews"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/logging"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	HOME_PAGE             string = "Services"
	SELECTED_SERVICE             = "ServiceHome"
	FLOATING_SERVICE_LIST        = "FloatingServices"
)

type DebugLogView struct {
	*tview.TextView
}

func (inst *DebugLogView) GetLastFocusedView() tview.Primitive {
	return inst.TextView
}

func RenderUI(config aws.Config, version string) {
	uicore.ResetGlobalStyle()

	var (
		app           = tview.NewApplication()
		errorTextArea = &DebugLogView{TextView: tview.NewTextView().SetWordWrap(false)}
		inAppLogger   = log.New(
			errorTextArea,
			log.Default().Prefix(),
			log.Default().Flags(),
		)
	)

	config.Logger = logging.StandardLogger{Logger: inAppLogger}

	var serviceViews = map[uiviews.ViewId]uicore.ServicePage{
		uiviews.LAMBDA:                   uiviews.NewLambdaHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_LOGS_GROUPS:   uiviews.NewLogsHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_LOGS_INSIGHTS: uiviews.NewLogsInsightsHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_ALARMS:        uiviews.NewAlarmsHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_METRICS:       uiviews.NewMetricsHomeView(app, config, inAppLogger),
		uiviews.CLOUDFORMATION:           uiviews.NewStacksHomeView(app, config, inAppLogger),
		uiviews.DYNAMODB:                 uiviews.NewDynamoDBHomeView(app, config, inAppLogger),
		uiviews.S3BUCKETS:                uiviews.NewS3bucketsHomeView(app, config, inAppLogger),
		uiviews.STATE_MACHINES:           uiviews.NewStepFunctionsHomeView(app, config, inAppLogger),

		uiviews.HELP:       uiviews.NewHelpHomeView(app, inAppLogger),
		uiviews.DEBUG_LOGS: errorTextArea,
	}

	errorTextArea.
		SetBorder(true).
		SetTitle("Logs").
		SetTitleAlign(tview.AlignLeft)

	var currentServiceView = tview.NewFlex()
	currentServiceView.AddItem(serviceViews[uiviews.LAMBDA], 0, 1, false)

	var servicesList = uiviews.ServicesHomeView()
	var flexLanding = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(servicesList, 0, 1, true).
		AddItem(tview.NewTextView().
			SetText(version).
			SetTextColor(tcell.ColorGrey),
			5, 0, false,
		)
	flexLanding.SetBorder(true)

	var pages = tview.NewPages().
		AddPage(SELECTED_SERVICE, currentServiceView, true, true).
		AddPage(FLOATING_SERVICE_LIST,
			uicore.FloatingView("Quick select", servicesList, 70, 25),
			true, true,
		).
		AddAndSwitchToPage(HOME_PAGE, flexLanding, true)

	var showServicesListToggle = false
	servicesList.SetSelectedFunc(func(i int, serviceName string, _ string, r rune) {
		var view, ok = serviceViews[uiviews.ViewId(serviceName)]
		if ok {
			currentServiceView.Clear()
			currentServiceView.AddItem(view, 0, 1, false)
			pages.SwitchToPage(SELECTED_SERVICE)
			app.SetFocus(view.GetLastFocusedView())

			showServicesListToggle = true
		}
	})

	var lastFocus = app.GetFocus()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC:
			pages.SwitchToPage(HOME_PAGE)
			app.SetFocus(servicesList)
		case tcell.KeyCtrlSpace:
			if showServicesListToggle {
				lastFocus = app.GetFocus()
				pages.ShowPage(FLOATING_SERVICE_LIST)
				app.SetFocus(servicesList)
			} else {
				pages.HidePage(FLOATING_SERVICE_LIST)
				app.SetFocus(lastFocus)
			}
			showServicesListToggle = !showServicesListToggle
		}
		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
