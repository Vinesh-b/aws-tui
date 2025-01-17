package main

import (
	"aws-tui/internal/pkg/ui/core"
	"aws-tui/internal/pkg/ui/services"
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
	core.ResetGlobalStyle()

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

	var serviceViews = map[services.ViewId]core.ServicePage{
		services.LAMBDA:                   services.NewLambdaHomeView(app, config, inAppLogger),
		services.CLOUDWATCH_LOGS_GROUPS:   services.NewLogsHomeView(app, config, inAppLogger),
		services.CLOUDWATCH_LOGS_INSIGHTS: services.NewLogsInsightsHomeView(app, config, inAppLogger),
		services.CLOUDWATCH_ALARMS:        services.NewAlarmsHomeView(app, config, inAppLogger),
		services.CLOUDWATCH_METRICS:       services.NewMetricsHomeView(app, config, inAppLogger),
		services.CLOUDFORMATION:           services.NewStacksHomeView(app, config, inAppLogger),
		services.DYNAMODB:                 services.NewDynamoDBHomeView(app, config, inAppLogger),
		services.S3BUCKETS:                services.NewS3bucketsHomeView(app, config, inAppLogger),
		services.STATE_MACHINES:           services.NewStepFunctionsHomeView(app, config, inAppLogger),

		services.HELP:       services.NewHelpHomeView(app, config, inAppLogger),
		services.DEBUG_LOGS: errorTextArea,
	}

	errorTextArea.
		SetBorder(true).
		SetTitle("Logs").
		SetTitleAlign(tview.AlignLeft)

	var servicesList = services.ServicesHomeView()
	var flexLanding = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(servicesList, 0, 1, true).
		AddItem(tview.NewTextView().
			SetText(version).
			SetTextColor(tcell.ColorGrey),
			5, 0, false,
		)
	flexLanding.SetBorder(true)

	var pages = tview.NewPages()

	for id, view := range serviceViews {
		pages.AddPage(string(id), view, true, true)
	}

	pages.
		AddPage(FLOATING_SERVICE_LIST,
			core.FloatingView("Quick select", servicesList, 70, 27),
			true, true,
		).
		AddAndSwitchToPage(HOME_PAGE, flexLanding, true)

	var serviceListHidden = false
	servicesList.SetSelectedFunc(func(i int, serviceName string, _ string, r rune) {
		var view = serviceViews[services.ViewId(serviceName)]
		pages.SwitchToPage(serviceName)
		app.SetFocus(view.GetLastFocusedView())

		serviceListHidden = true
	})

	var lastFocus = app.GetFocus()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC:
			if !serviceListHidden {
				pages.HidePage(FLOATING_SERVICE_LIST)
				app.SetFocus(lastFocus)
				serviceListHidden = true
				return nil
			}
		case core.APP_KEY_BINDINGS.ToggleServicesMenu:
			if serviceListHidden {
				lastFocus = app.GetFocus()
				pages.ShowPage(FLOATING_SERVICE_LIST)
				app.SetFocus(servicesList)
			} else {
				pages.HidePage(FLOATING_SERVICE_LIST)
				app.SetFocus(lastFocus)
			}
			serviceListHidden = !serviceListHidden
		}
		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
