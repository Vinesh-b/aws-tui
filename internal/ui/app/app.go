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
	// Build creates the page again, with new api objects. It is nil for pages
	// that hold no data from AWS.
	Build func() core.ServicePage
}

func newServiceItem(
	mainText string, secondaryText string, shortcut rune, build func() core.ServicePage,
) *ServiceItem {
	return &ServiceItem{mainText, secondaryText, shortcut, build(), build}
}

func staticServiceItem(
	mainText string, secondaryText string, shortcut rune, page core.ServicePage,
) *ServiceItem {
	return &ServiceItem{mainText, secondaryText, shortcut, page, nil}
}

// rebuildServicePages replaces every page that holds AWS data with a new one,
// which drops what was loaded and the caches of its api objects.
func rebuildServicePages(pages *tview.Pages, items []*ServiceItem) {
	var visible = map[string]bool{}
	for _, name := range pages.GetPageNames(true) {
		visible[name] = true
	}

	for _, item := range items {
		if item.Build == nil {
			continue
		}
		if closer, ok := item.ServicePage.(interface{ Close() }); ok {
			closer.Close()
		}
		item.ServicePage = item.Build()
		pages.RemovePage(item.MainText)
		pages.AddPage(item.MainText, item.ServicePage, true, visible[item.MainText])
		// Service pages sit below the floating service list and the home page
		pages.SendToBack(item.MainText)
	}
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

	var serviceViews = []*ServiceItem{
		newServiceItem("󰘧 "+string(core.LAMBDA), "Lambdas and logs", rune('1'), func() core.ServicePage {
			return lambda.NewLambdaHomeView(
				appContext,
				awsapi.NewLambdaApi(appContext.Logger),
				awsapi.NewCloudWatchLogsApi(appContext.Logger),
			)
		}),
		newServiceItem("󰺮 "+string(core.CLOUDWATCH_LOGS_INSIGHTS), "Query and filter logs", rune('2'), func() core.ServicePage {
			return cloudwatch.NewLogsInsightsHomeView(appContext, awsapi.NewCloudWatchLogsApi(appContext.Logger))
		}),
		newServiceItem("󰆼 "+string(core.DYNAMODB), "View and search DynamoDB tables", rune('3'), func() core.ServicePage {
			return dynamodb.NewDynamoDBHomeView(appContext, awsapi.NewDynamoDBApi(appContext.Logger))
		}),
		newServiceItem("󱁊 "+string(core.STATE_MACHINES), "Step Functions and executions", rune('4'), func() core.ServicePage {
			return sfn.NewStepFunctionsHomeView(
				appContext,
				awsapi.NewStateMachineApi(appContext.Logger),
				awsapi.NewCloudWatchLogsApi(appContext.Logger),
			)
		}),
		newServiceItem("󱐕 "+string(core.S3BUCKETS), "S3 buckets and objects", rune('5'), func() core.ServicePage {
			return s3.NewS3bucketsHomeView(appContext, awsapi.NewS3BucketsApi(appContext.Logger))
		}),
		newServiceItem("󰙵 "+string(core.SYSTEMS_MANAGER), "Application parameters", rune('6'), func() core.ServicePage {
			return ssm.NewSystemManagerHomeView(appContext, awsapi.NewSystemsManagerApi(appContext.Logger))
		}),
		newServiceItem(" "+string(core.CLOUDWATCH_LOGS_GROUPS), "Logs groups and streams", rune('7'), func() core.ServicePage {
			return cloudwatch.NewLogsHomeView(appContext, awsapi.NewCloudWatchLogsApi(appContext.Logger))
		}),
		newServiceItem("󰘘 "+string(core.EVENTBRIDGE), "Event buses, rules, schedules...", rune('8'), func() core.ServicePage {
			return eventbridge.NewEventBridgeHomeView(appContext, awsapi.NewEventBridgeApi(appContext.Logger))
		}),
		newServiceItem(" "+string(core.CLOUDFORMATION), "Cloud formation stacks", rune('󰯉'), func() core.ServicePage {
			return cloudformation.NewStacksHomeView(appContext, awsapi.NewCloudFormationApi(appContext.Logger))
		}),
		newServiceItem("󰞏 "+string(core.CLOUDWATCH_ALARMS), "Metric alarms", rune('󰯉'), func() core.ServicePage {
			return cloudwatch.NewAlarmsHomeView(appContext, awsapi.NewCloudWatchAlarmsApi(appContext.Logger))
		}),
		newServiceItem(" "+string(core.CLOUDWATCH_METRICS), "View metrics", rune('󰯉'), func() core.ServicePage {
			return cloudwatch.NewMetricsHomeView(appContext, awsapi.NewCloudWatchMetricsApi(appContext.Logger))
		}),
		newServiceItem("󱇱 "+string(core.VPC), "VPCs, endpoints, subnets...", rune('󰯉'), func() core.ServicePage {
			return vpc.NewVpcHomeView(appContext, awsapi.NewEc2Api(appContext.Logger))
		}),
		newServiceItem("󱃔 "+string(core.SQS), "Queues and their attributes", rune('󰯉'), func() core.ServicePage {
			return sqs.NewSqsHomeView(appContext, awsapi.NewSqsApi(appContext.Logger))
		}),
		staticServiceItem("󰘥 "+string(core.HELP), "Help docs on how to use this app", rune('?'),
			services.NewHelpHomeView(appContext),
		),
		staticServiceItem(" "+string(core.SETTINGS), "Configure and tweak the app", rune('s'),
			&EmptyPlaceholderView{Box: tview.NewBox()},
		),
		staticServiceItem(" "+string(core.DEBUG_LOGS), "View debug logs", rune('0'),
			errorTextArea,
		),
		staticServiceItem(" "+string(core.PROFILE_SELECTION), "AWS Profile selection", rune('p'),
			services.NewProfileSelectionView(appContext),
		),
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

	appContext.OnApiClientsReset(func() {
		rebuildServicePages(pages, serviceViews)
	})

	app.SetRoot(pages, true)
	return app
}
