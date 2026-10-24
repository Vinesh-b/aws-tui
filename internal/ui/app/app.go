package app

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudformation"
	"aws-tui/internal/ui/cloudwatch"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/dynamodb"
	"aws-tui/internal/ui/eventbridge"
	"aws-tui/internal/ui/lambda"
	"aws-tui/internal/ui/s3"
	"aws-tui/internal/ui/services"
	"aws-tui/internal/ui/sfn"
	"aws-tui/internal/ui/sqs"
	"aws-tui/internal/ui/ssm"
	"aws-tui/internal/ui/vpc"
	"context"
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

func CreateApplication(ctx context.Context, config aws.Config, version string) *tview.Application {

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
		appContext = core.NewAppContext(ctx, app, &config, inAppLogger, &appTheme)
	)

	config.Logger = logging.StandardLogger{Logger: inAppLogger}
	// The api clients must log to the in-app view; SDK warnings written to the
	// terminal would shift the whole screen.
	appContext.ResetApiClients(config, "")

	var serviceViews = []ServiceItem{
		{"󰘧 " + string(core.LAMBDA), "Lambdas and logs", rune('1'),
			lambda.NewLambdaHomeView(
				appContext,
				awsapi.NewLambdaApi(appContext.Logger),
				awsapi.NewCloudWatchLogsApi(appContext.Logger),
			),
		},
		{"󰺮 " + string(core.CLOUDWATCH_LOGS_INSIGHTS), "Query and filter logs", rune('2'),
			cloudwatch.NewLogsInsightsHomeView(appContext, awsapi.NewCloudWatchLogsApi(appContext.Logger)),
		},
		{"󰆼 " + string(core.DYNAMODB), "View and search DynamoDB tables", rune('3'),
			dynamodb.NewDynamoDBHomeView(appContext, awsapi.NewDynamoDBApi(appContext.Logger)),
		},
		{"󱁊 " + string(core.STATE_MACHINES), "Step Functions and executions", rune('4'),
			sfn.NewStepFunctionsHomeView(
				appContext,
				awsapi.NewStateMachineApi(appContext.Logger),
				awsapi.NewCloudWatchLogsApi(appContext.Logger),
			),
		},
		{"󱐕 " + string(core.S3BUCKETS), "S3 buckets and objects", rune('5'),
			s3.NewS3bucketsHomeView(appContext, awsapi.NewS3BucketsApi(appContext.Logger)),
		},
		{"󰙵 " + string(core.SYSTEMS_MANAGER), "Application parameters", rune('6'),
			ssm.NewSystemManagerHomeView(appContext, awsapi.NewSystemsManagerApi(appContext.Logger)),
		},
		{" " + string(core.CLOUDWATCH_LOGS_GROUPS), "Logs groups and streams", rune('7'),
			cloudwatch.NewLogsHomeView(appContext, awsapi.NewCloudWatchLogsApi(appContext.Logger)),
		},
		{"󰘘 " + string(core.EVENTBRIDGE), "Event buses, rules, schedules...", rune('8'),
			eventbridge.NewEventBridgeHomeView(appContext, awsapi.NewEventBridgeApi(appContext.Logger)),
		},
		{" " + string(core.CLOUDFORMATION), "Cloud formation stacks", rune('󰯉'),
			cloudformation.NewStacksHomeView(appContext, awsapi.NewCloudFormationApi(appContext.Logger)),
		},
		{"󰞏 " + string(core.CLOUDWATCH_ALARMS), "Metric alarms", rune('󰯉'),
			cloudwatch.NewAlarmsHomeView(appContext, awsapi.NewCloudWatchAlarmsApi(appContext.Logger)),
		},
		{" " + string(core.CLOUDWATCH_METRICS), "View metrics", rune('󰯉'),
			cloudwatch.NewMetricsHomeView(appContext, awsapi.NewCloudWatchMetricsApi(appContext.Logger)),
		},
		{"󱇱 " + string(core.VPC), "VPCs, endpoints, subnets...", rune('󰯉'),
			vpc.NewVpcHomeView(appContext, awsapi.NewEc2Api(appContext.Logger)),
		},
		{"󱃔 " + string(core.SQS), "Queues and their attributes", rune('󰯉'),
			sqs.NewSqsHomeView(appContext, awsapi.NewSqsApi(appContext.Logger)),
		},
		{"󰘥 " + string(core.HELP), "Help docs on how to use this app", rune('?'),
			services.NewHelpHomeView(appContext),
		},
		{" " + string(core.SETTINGS), "Configure and tweak the app", rune('s'),
			&EmptyPlaceholderView{Box: tview.NewBox()},
		},
		{" " + string(core.DEBUG_LOGS), "View debug logs", rune('0'),
			errorTextArea,
		},
		{" " + string(core.PROFILE_SELECTION), "AWS Profile selection", rune('p'),
			services.NewProfileSelectionView(appContext),
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
		var hideServiceListFunc = func() {
			pages.HidePage(FLOATING_SERVICE_LIST)
			app.SetFocus(lastFocus)
			serviceListHidden = true

		}

		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Quit:
			if !serviceListHidden && servicesList.IsEscapable() {
				hideServiceListFunc()
				return nil
			}
		}

		switch event.Key() {
		case core.APP_KEY_BINDINGS.Escape, tcell.Key(core.APP_KEY_BINDINGS.Quit):
			if !serviceListHidden && servicesList.IsEscapable() {
				hideServiceListFunc()
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

	app.SetRoot(pages, true)
	return app
}
