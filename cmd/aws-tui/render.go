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

func floatingView(p tview.Primitive, width, height int) tview.Primitive {
	var wrapper = tview.NewFlex().
		AddItem(p, 0, 1, true)
	wrapper.SetBorder(true).SetTitle("Quick select")
	var window = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(wrapper, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)

	return window
}

func RenderUI(config aws.Config, version string) {
	uicore.ResetGlobalStyle()

	var (
		app           = tview.NewApplication()
		errorTextArea = tview.NewTextView().SetWordWrap(false)
		inAppLogger   = log.New(
			errorTextArea,
			log.Default().Prefix(),
			log.Default().Flags(),
		)
	)

	config.Logger = logging.StandardLogger{Logger: inAppLogger}

	var serviceViews = map[uiviews.ViewId]tview.Primitive{
		uiviews.LAMBDA:                   uiviews.CreateLambdaHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_LOGS_GROUPS:   uiviews.CreateLogsHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_LOGS_INSIGHTS: uiviews.CreateLogsInsightsHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_ALARMS:        uiviews.CreateAlarmsHomeView(app, config, inAppLogger),
		uiviews.CLOUDWATCH_METRICS:       uiviews.CreateMetricsHomeView(app, config, inAppLogger),
		uiviews.CLOUDFORMATION:           uiviews.CreateStacksHomeView(app, config, inAppLogger),
		uiviews.DYNAMODB:                 uiviews.CreateDynamoDBHomeView(app, config, inAppLogger),
		uiviews.S3BUCKETS:                uiviews.CreateS3bucketsHomeView(app, config, inAppLogger),
		uiviews.STATE_MACHINES:           uiviews.CreateStepFunctionsHomeView(app, config, inAppLogger),

		uiviews.HELP:       uiviews.CreateHelpHomeView(app, inAppLogger),
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
		AddPage(FLOATING_SERVICE_LIST, floatingView(servicesList, 70, 25), true, true).
		AddAndSwitchToPage(HOME_PAGE, flexLanding, true)

	var showServicesListToggle = false
	var lastFocus = app.GetFocus()
	servicesList.SetSelectedFunc(func(i int, serviceName string, _ string, r rune) {
		var view, ok = serviceViews[uiviews.ViewId(serviceName)]
		if ok {
			currentServiceView.Clear()
			currentServiceView.AddItem(view, 0, 1, false)
			pages.SwitchToPage(SELECTED_SERVICE)
			app.SetFocus(view)

			lastFocus = view
			showServicesListToggle = true
		}
	})

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
