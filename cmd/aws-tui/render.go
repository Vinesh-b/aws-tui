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

type PageName = string

const (
	HOME_PAGE             PageName = "Services"
	SELECTED_SERVICE      PageName = "ServiceHome"
	FLOATING_SERVICE_LIST PageName = "FloatingServices"
)

type EmptyPlaceholderView struct {
	*tview.Box
}

func (inst *EmptyPlaceholderView) GetLastFocusedView() tview.Primitive {
	return inst.Box
}

type DebugLogView struct {
	*tview.TextView
}

func (inst *DebugLogView) GetLastFocusedView() tview.Primitive {
	return inst.TextView
}

type ServiceItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	ServicePage   core.ServicePage
}

func RenderUI(config aws.Config, version string) {

	var appTheme = core.AppTheme{
		PrimaryTextColour:           tcell.NewHexColor(0xBFBFBF),
		SecondaryTextColour:         tcell.NewHexColor(0xFFFFFF),
		TertiaryTextColour:          tcell.NewHexColor(0xCC8B00),
		TitleColour:                 tcell.NewHexColor(0x43B143),
		BorderColour:                tcell.NewHexColor(0x404040),
		InverseTextColour:           tcell.NewHexColor(0x404040),
		BackgroundColour:            tcell.ColorDefault,
		ContrastBackgroundColor:     tcell.NewHexColor(0x303030),
		MoreContrastBackgroundColor: tcell.NewHexColor(0x404040),
		PlaceholderTextColour:       tcell.NewHexColor(0x717171),
	}

	appTheme.ResetGlobalStyle()

	var (
		app           = tview.NewApplication()
		errorTextArea = &DebugLogView{TextView: tview.NewTextView().SetWordWrap(false)}
		inAppLogger   = log.New(
			errorTextArea,
			log.Default().Prefix(),
			log.Default().Flags(),
		)
		appContext = core.NewAppContext(app, &config, inAppLogger, &appTheme)
	)

	config.Logger = logging.StandardLogger{Logger: inAppLogger}

	var serviceViews = []ServiceItem{
		{"󰘧 " + string(services.LAMBDA), "Lambdas and logs", rune('1'),
			services.NewLambdaHomeView(appContext),
		},
		{"󰺮 " + string(services.CLOUDWATCH_LOGS_INSIGHTS), "Query and filter logs", rune('2'),
			services.NewLogsInsightsHomeView(appContext),
		},
		{"󰆼 " + string(services.DYNAMODB), "View and search DynamoDB tables", rune('3'),
			services.NewDynamoDBHomeView(appContext),
		},
		{"󱁊 " + string(services.STATE_MACHINES), "Step Functions and executions", rune('4'),
			services.NewStepFunctionsHomeView(appContext),
		},
		{"󱐕 " + string(services.S3BUCKETS), "S3 buckets and objects", rune('5'),
			services.NewS3bucketsHomeView(appContext),
		},
		{"󰙵 " + string(services.SYSTEMS_MANAGER), "Application parameters", rune('6'),
			services.NewSystemManagerHomeView(appContext),
		},
		{" " + string(services.CLOUDFORMATION), "Cloud formation stacks", rune('󰯉'),
			services.NewStacksHomeView(appContext),
		},
		{"󰞏 " + string(services.CLOUDWATCH_ALARMS), "Metric alarms", rune('󰯉'),
			services.NewAlarmsHomeView(appContext),
		},
		{" " + string(services.CLOUDWATCH_METRICS), "View metrics", rune('󰯉'),
			services.NewMetricsHomeView(appContext),
		},
		{" " + string(services.CLOUDWATCH_LOGS_GROUPS), "Logs groups and streams", rune('󰯉'),
			services.NewLogsHomeView(appContext),
		},
		{"󰘘 " + string(services.EVENTBRIDGE), "Event buses, rules, schedules...", rune('󰯉'),
			services.NewEventBridgeHomeView(appContext),
		},
		{"󰘥 " + string(services.HELP), "Help docs on how to use this app", rune('?'),
			services.NewHelpHomeView(appContext),
		},
		{" " + string(services.SETTINGS), "Configure and tweak the app", rune('s'),
			&EmptyPlaceholderView{Box: tview.NewBox()},
		},
		{" " + string(services.DEBUG_LOGS), "View debug logs", rune('0'),
			errorTextArea,
		},
	}

	errorTextArea.
		SetBorder(true).
		SetTitle("Logs").
		SetTitleAlign(tview.AlignLeft)

	var servicesList = services.NewServicesHomeView(appContext)
	var flexLanding = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(servicesList, 0, 1, true).
		AddItem(tview.NewTextView().
			SetText(version).
			SetTextColor(tcell.ColorGrey),
			5, 0, false,
		)
	flexLanding.SetBorder(true)

	var pages = tview.NewPages()
	var serviceListHidden = false

	for _, item := range serviceViews {
		var name = item.MainText
		pages.AddPage(name, item.ServicePage, true, true)
		servicesList.AddItem(name, item.SecondaryText, item.Shortcut, func() {
			pages.SwitchToPage(name)
			app.SetFocus(item.ServicePage.GetLastFocusedView())

			serviceListHidden = true
		})
	}

	pages.
		AddPage(FLOATING_SERVICE_LIST,
			core.FloatingView("Quick select", servicesList, 70, 27),
			true, true,
		).
		AddAndSwitchToPage(HOME_PAGE, flexLanding, true)

	var lastFocus = app.GetFocus()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Escape:
			if !serviceListHidden && servicesList.IsEscapable() {
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
