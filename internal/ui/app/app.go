package app

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/settings"
	"aws-tui/internal/ui/cloudformation"
	"aws-tui/internal/ui/cloudwatch"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/dynamodb"
	"aws-tui/internal/ui/eventbridge"
	"aws-tui/internal/ui/iam"
	"aws-tui/internal/ui/lambda"
	"aws-tui/internal/ui/s3"
	"aws-tui/internal/ui/services"
	"aws-tui/internal/ui/sfn"
	"aws-tui/internal/ui/sns"
	"aws-tui/internal/ui/sqs"
	"aws-tui/internal/ui/ssm"
	"aws-tui/internal/ui/vpc"
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
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

type DebugLogView struct {
	*tview.TextView
}

func (inst *DebugLogView) GetLastFocusedView() tview.Primitive {
	return inst.TextView
}

const maxDebugLogLines = 5000

// NewDebugLogView follows the end of the log as lines arrive until the user
// scrolls up, and G follows again. The logger writes from background
// goroutines, so redraws are queued to the app.
func NewDebugLogView(app *tview.Application) *DebugLogView {
	var textView = tview.NewTextView().
		SetWordWrap(false).
		SetMaxLines(maxDebugLogLines).
		SetChangedFunc(func() { app.QueueUpdateDraw(func() {}) })
	textView.ScrollToEnd()

	return &DebugLogView{TextView: textView}
}

type ServiceItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	ServicePage   core.ServicePage
	// Build creates the page again, with new api objects and the current theme.
	// It is nil for pages that can be neither.
	Build func() core.ServicePage
	// HoldsAwsData is true for pages that must be built again when the api
	// clients change. Pages that are only built again for a new theme leave it
	// false, so switching profile does not replace the page being used.
	HoldsAwsData bool
}

func newServiceItem(
	mainText string, secondaryText string, shortcut rune, build func() core.ServicePage,
) *ServiceItem {
	return &ServiceItem{mainText, secondaryText, shortcut, build(), build, true}
}

// themedServiceItem is a page that holds no data from AWS but takes its colours
// from the theme, so it is built again when the theme changes.
func themedServiceItem(
	mainText string, secondaryText string, shortcut rune, build func() core.ServicePage,
) *ServiceItem {
	return &ServiceItem{mainText, secondaryText, shortcut, build(), build, false}
}

func staticServiceItem(
	mainText string, secondaryText string, shortcut rune, page core.ServicePage,
) *ServiceItem {
	return &ServiceItem{mainText, secondaryText, shortcut, page, nil, false}
}

// rebuildServicePages replaces every page that holds AWS data with a new one,
// which drops what was loaded and the caches of its api objects.
func rebuildServicePages(pages *tview.Pages, items []*ServiceItem) {
	rebuildPages(pages, items, func(item *ServiceItem) bool { return item.HoldsAwsData })
}

// rebuildThemedPages replaces every page that can be built again, so they take
// the colours of the current theme.
func rebuildThemedPages(pages *tview.Pages, items []*ServiceItem) {
	rebuildPages(pages, items, func(item *ServiceItem) bool { return true })
}

func rebuildPages(pages *tview.Pages, items []*ServiceItem, include func(*ServiceItem) bool) {
	var visible = map[string]bool{}
	for _, name := range pages.GetPageNames(true) {
		visible[name] = true
	}

	for _, item := range items {
		if item.Build == nil || !include(item) {
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

	// Removing the only visible page makes the last page visible, which is the
	// home page, so put the visibility of every page back as it was
	for _, name := range pages.GetPageNames(false) {
		if visible[name] {
			pages.ShowPage(name)
		} else {
			pages.HidePage(name)
		}
	}
}

// openResource switches to the page of the service of the resource and opens
// it there
func openResource(app *tview.Application, pages *tview.Pages, items []*ServiceItem, resource arn.ARN) error {
	for _, item := range items {
		var opener, ok = item.ServicePage.(core.ResourceOpener)
		if !ok || !opener.OpensArnService(resource.Service) {
			continue
		}
		pages.SwitchToPage(item.MainText)
		opener.OpenResource(resource)
		app.SetFocus(item.ServicePage.GetLastFocusedView())
		return nil
	}
	return fmt.Errorf("can't open %s resources", resource.Service)
}

type appOptions struct {
	settingsPath       string
	pollSessionAtStart bool
}

// Option changes how the application is created.
type Option func(*appOptions)

// WithSettingsFile keeps the settings the user chooses, such as the theme, in the
// file. They are read when the application is created and written when they
// change. Without it nothing is read or written.
func WithSettingsFile(path string) Option {
	return func(options *appOptions) { options.settingsPath = path }
}

// WithSessionPollAtStart shows the session details as soon as a service is
// opened, instead of after the first poll interval. It is left out of tests so
// they do not look up the credentials.
func WithSessionPollAtStart() Option {
	return func(options *appOptions) { options.pollSessionAtStart = true }
}

// loadSettings reads the saved settings. Problems with them are returned to be
// logged, since the app still starts, with the defaults for what is missing.
func loadSettings(path string) (settings.Settings, []error) {
	if path == "" {
		return settings.Settings{}, nil
	}

	var saved, err = settings.Load(path)
	if err != nil {
		return settings.Settings{}, []error{err}
	}

	return saved, nil
}

func CreateApplication(
	ctx context.Context, config aws.Config, version string, opts ...Option,
) *tview.Application {
	var options appOptions
	for _, opt := range opts {
		opt(&options)
	}

	var saved, problems = loadSettings(options.settingsPath)

	var appTheme = core.DefaultTheme()
	if saved.Theme != "" {
		if preset, ok := core.FindThemePreset(saved.Theme); ok {
			appTheme = preset
		} else {
			problems = append(problems, fmt.Errorf("unknown theme %q in the settings file", saved.Theme))
		}
	}

	appTheme.ResetGlobalStyle()

	var (
		app           = tview.NewApplication()
		errorTextArea = NewDebugLogView(app)
		inAppLogger   = log.New(
			errorTextArea,
			log.Default().Prefix(),
			log.Default().Flags(),
		)
		appContext = core.NewAppContext(ctx, app, &config, inAppLogger, &appTheme)
	)
	appContext.PollSessionAtStart = options.pollSessionAtStart
	appContext.DownloadDir = saved.DownloadDir

	for _, problem := range problems {
		inAppLogger.Print(problem)
	}

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
				awsapi.NewCloudWatchMetricsApi(appContext.Logger),
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
		newServiceItem("󰌆 "+string(core.IAM), "Roles, users and policies", rune('󰯉'), func() core.ServicePage {
			return iam.NewIamHomeView(appContext, awsapi.NewIamApi(appContext.Logger))
		}),
		newServiceItem("󱃔 "+string(core.SQS), "Queues and their attributes", rune('󰯉'), func() core.ServicePage {
			return sqs.NewSqsHomeView(appContext, awsapi.NewSqsApi(appContext.Logger))
		}),
		newServiceItem("󰂚 "+string(core.SNS), "Topics and their attributes", rune('󰯉'), func() core.ServicePage {
			return sns.NewSnsHomeView(appContext, awsapi.NewSnsApi(appContext.Logger))
		}),
		themedServiceItem("󰘥 "+string(core.HELP), "Help docs on how to use this app", rune('?'), func() core.ServicePage {
			return services.NewHelpHomeView(appContext)
		}),
		themedServiceItem(" "+string(core.SETTINGS), "Configure and tweak the app", rune('s'), func() core.ServicePage {
			return services.NewSettingsView(appContext)
		}),
		staticServiceItem(" "+string(core.DEBUG_LOGS), "View debug logs", rune('0'),
			errorTextArea,
		),
		themedServiceItem(" "+string(core.PROFILE_SELECTION), "AWS Profile selection", rune('p'), func() core.ServicePage {
			return services.NewProfileSelectionView(appContext)
		}),
	}

	errorTextArea.
		SetBorder(true).
		SetTitle("Logs").
		SetTitleAlign(tview.AlignLeft)

	var servicesList = services.NewServicesHomeView(appContext)
	var versionView = tview.NewTextView().
		SetText(version).
		SetTextColor(tcell.ColorGrey)
	var flexLanding = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(servicesList, 0, 1, true).
		AddItem(versionView, 5, 0, false)
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

	// Links in one service open the resource in the page of its service
	appContext.SetResourceOpener(func(resource arn.ARN) error {
		if err := openResource(app, pages, serviceViews, resource); err != nil {
			return err
		}
		serviceListHidden = true
		return nil
	})

	var floatingServices = core.FloatingView("Quick select", servicesList, 70, 27)

	pages.
		AddPage(FLOATING_SERVICE_LIST, floatingServices, true, true).
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

	// Views take their colours when they are built, so the pages are built
	// again for a new theme. The views that stay, as they hold the log or
	// are used to switch pages, are given the colours of the theme.
	appContext.OnThemeChanged(func() {
		if options.settingsPath != "" {
			saved.Theme = appContext.Theme.Name
			if err := settings.Save(options.settingsPath, saved); err != nil {
				appContext.Logger.Print(err)
			}
		}

		rebuildThemedPages(pages, serviceViews)

		var theme = appContext.Theme
		pages.SetBackgroundColor(theme.BackgroundColour)
		core.RestyleChrome(floatingServices, theme)
		core.RestyleChrome(flexLanding, theme)
		core.RestyleChrome(errorTextArea.TextView, theme)
		versionView.SetTextColor(tcell.ColorGrey)

		// The page the theme was picked on has been replaced
		if name, _ := pages.GetFrontPage(); serviceListHidden {
			for _, item := range serviceViews {
				if item.MainText == name {
					app.SetFocus(item.ServicePage.GetLastFocusedView())
				}
			}
		}
	})

	app.SetRoot(pages, true)
	return app
}
