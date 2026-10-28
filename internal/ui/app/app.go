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
	"aws-tui/internal/ui/kms"
	"aws-tui/internal/ui/lambda"
	"aws-tui/internal/ui/s3"
	"aws-tui/internal/ui/secretsmanager"
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

// navigator shows the service pages and keeps the history of the pages shown
type navigator struct {
	app     *tview.Application
	pages   *tview.Pages
	items   []*ServiceItem
	history *core.NavigationHistory
}

const navigationHistoryLimit = 100

func newNavigator(app *tview.Application, pages *tview.Pages, items []*ServiceItem) *navigator {
	return &navigator{app, pages, items, core.NewNavigationHistory(navigationHistoryLimit)}
}

func (inst *navigator) find(service string) (*ServiceItem, bool) {
	for _, item := range inst.items {
		if item.MainText == service {
			return item, true
		}
	}
	return nil, false
}

func locationOf(item *ServiceItem) core.Location {
	var location = core.Location{Service: item.MainText}
	if paged, ok := item.ServicePage.(core.PagedServicePage); ok {
		location.Page = paged.CurrentServicePage()
	}
	return location
}

// current is the location of the service in front, which is false when no
// service is, such as on the home page
func (inst *navigator) current() (core.Location, bool) {
	var name, _ = inst.pages.GetFrontPage()
	if item, ok := inst.find(name); ok {
		return locationOf(item), true
	}
	return core.Location{}, false
}

// recordPages keeps the pages shown in the page of the service, while the
// service is the one in front
func (inst *navigator) recordPages(item *ServiceItem, page core.ServicePage) {
	var paged, ok = page.(core.PagedServicePage)
	if !ok {
		return
	}
	paged.SetServicePageChangedFunc(func(name string) {
		if front, _ := inst.pages.GetFrontPage(); front == item.MainText {
			inst.history.Visit(core.Location{Service: item.MainText, Page: name})
		}
	})
}

// watch records the pages shown in the page of the service, and in the pages
// that replace it when it is built again
func (inst *navigator) watch(item *ServiceItem) {
	inst.recordPages(item, item.ServicePage)
	if build := item.Build; build != nil {
		item.Build = func() core.ServicePage {
			var page = build()
			inst.recordPages(item, page)
			return page
		}
	}
}

// movePage shows the page offset from the one shown, in the service in front
func (inst *navigator) movePage(offset int) {
	var name, _ = inst.pages.GetFrontPage()
	if item, ok := inst.find(name); ok {
		if paged, ok := item.ServicePage.(core.PagedServicePage); ok {
			paged.MoveServicePage(offset)
		}
	}
}

// showService switches to the service, on the page it was left on
func (inst *navigator) showService(item *ServiceItem) {
	inst.pages.SwitchToPage(item.MainText)
	inst.app.SetFocus(item.ServicePage.GetLastFocusedView())
	inst.history.Visit(locationOf(item))
}

// openResource switches to the page of the service of the resource and opens
// it there. Going back returns to where the link was opened from.
func (inst *navigator) openResource(resource arn.ARN) error {
	for _, item := range inst.items {
		var opener, ok = item.ServicePage.(core.ResourceOpener)
		if !ok || !opener.OpensArnService(resource.Service) {
			continue
		}
		// The service is shown on the page it was left on before the
		// resource is opened, which is not a page that was looked at
		inst.history.WithoutRecording(func() {
			inst.pages.SwitchToPage(item.MainText)
			opener.OpenResource(resource)
		})
		inst.app.SetFocus(item.ServicePage.GetLastFocusedView())
		inst.history.Visit(locationOf(item))
		return nil
	}
	return fmt.Errorf("can't open %s resources", resource.Service)
}

// show shows the location without it being a new visit. It is false if the
// service is no longer in the app.
func (inst *navigator) show(location core.Location) bool {
	var item, ok = inst.find(location.Service)
	if !ok {
		return false
	}
	inst.history.WithoutRecording(func() {
		inst.pages.SwitchToPage(item.MainText)
		if paged, ok := item.ServicePage.(core.PagedServicePage); ok && location.Page != "" {
			paged.ShowServicePage(location.Page)
		}
		inst.app.SetFocus(item.ServicePage.GetLastFocusedView())
	})
	return true
}

func (inst *navigator) Back() bool {
	var location, ok = inst.history.Back()
	return ok && inst.show(location)
}

func (inst *navigator) Forward() bool {
	var location, ok = inst.history.Forward()
	return ok && inst.show(location)
}

// restart starts the history again from the service in front, for when the
// pages it held were built again
func (inst *navigator) restart() {
	if location, ok := inst.current(); ok {
		inst.history.Restart(location)
	} else {
		inst.history.Clear()
	}
}

type appOptions struct {
	settingsPath       string
	pollSessionAtStart bool
	copyToSystem       func(text string) error
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

// WithSystemClipboard copies to the function in place of the clipboard of the
// system, such as in tests, which should leave the clipboard of the user alone
func WithSystemClipboard(copy func(text string) error) Option {
	return func(options *appOptions) { options.copyToSystem = copy }
}

// settingsStore keeps the settings the user chooses, and saves them in the
// settings file when the app has one. It is only used on the UI goroutine.
type settingsStore struct {
	path  string
	saved settings.Settings
}

func (inst *settingsStore) Settings() settings.Settings {
	return inst.saved
}

func (inst *settingsStore) Update(change func(saved *settings.Settings)) error {
	change(&inst.saved)
	if inst.path == "" {
		return nil
	}
	return settings.Save(inst.path, inst.saved)
}

// rebuildPagesForKeys replaces every page that can be built again but the one
// the keys were changed on, so the pages take the new keys
func rebuildPagesForKeys(pages *tview.Pages, items []*ServiceItem, changedOn *ServiceItem) {
	rebuildPages(pages, items, func(item *ServiceItem) bool { return item != changedOn })
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

	// Before any view is built, as views list their keys in their help.
	problems = append(problems, core.ApplyKeyBindingOverrides(saved.Keys)...)

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
	if options.copyToSystem != nil {
		appContext.SetSystemClipboard(options.copyToSystem)
	}
	var store = &settingsStore{path: options.settingsPath, saved: saved}
	appContext.DownloadDir = saved.Files.DownloadDir
	appContext.QueriesPath = settings.QueriesPath(options.settingsPath)

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
		newServiceItem(" "+string(core.CLOUDWATCH_LOGS), "Log groups, streams and Insights queries", rune('7'), func() core.ServicePage {
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
		newServiceItem("󰯄 "+string(core.KMS), "Encryption keys, aliases and policies", rune('󰯉'), func() core.ServicePage {
			return kms.NewKmsHomeView(appContext, awsapi.NewKmsApi(appContext.Logger))
		}),
		newServiceItem("󰌋 "+string(core.SECRETS_MANAGER), "Secrets and their values", rune('󰯉'), func() core.ServicePage {
			return secretsmanager.NewSecretsManagerHomeView(appContext, awsapi.NewSecretsManagerApi(appContext.Logger))
		}),
		themedServiceItem("󰘥 "+string(core.HELP), "Help docs on how to use this app", rune('?'), func() core.ServicePage {
			return services.NewHelpHomeView(appContext)
		}),
		themedServiceItem(" "+string(core.SETTINGS), "Configure and tweak the app", rune('s'), func() core.ServicePage {
			return services.NewSettingsView(appContext, store)
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
	var nav = newNavigator(app, pages, serviceViews)
	var sequencer = core.NewKeySequencer(appContext)

	for _, item := range serviceViews {
		var name = item.MainText
		pages.AddPage(name, item.ServicePage, true, true)
		servicesList.AddItem(name, item.SecondaryText, item.Shortcut, func() {
			nav.showService(item)
			serviceListHidden = true
		})

		nav.watch(item)
	}

	// Links in one service open the resource in the page of its service
	appContext.SetResourceOpener(func(resource arn.ARN) error {
		if err := nav.openResource(resource); err != nil {
			return err
		}
		serviceListHidden = true
		return nil
	})

	var floatingServices = core.FloatingView("Quick select", servicesList, 70, 27)

	var commandLine = core.NewCommandLine(appContext, pages)
	var appCommands = &commands{
		items:  serviceViews,
		appCtx: appContext,
		line:   commandLine,
		show: func(item *ServiceItem) {
			nav.showService(item)
			serviceListHidden = true
		},
		quit: app.Stop,
	}
	commandLine.SetRunFunc(appCommands.Run)

	// gt and gT change the page of the service in front, not of one behind
	// the services menu
	sequencer.SetMovePageFunc(func(offset int) {
		if serviceListHidden {
			nav.movePage(offset)
		}
	})
	commandLine.SetCompleteFunc(appCommands.Complete)

	pages.
		AddPage(FLOATING_SERVICE_LIST, floatingServices, true, true).
		AddAndSwitchToPage(HOME_PAGE, flexLanding, true)

	var lastFocus = app.GetFocus()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		event = core.NormalizeKeyEvent(event)
		if event = sequencer.HandleEvent(event, app.GetFocus()); event == nil {
			return nil
		}
		if event = commandLine.HandleEvent(event, app.GetFocus()); event == nil {
			return nil
		}
		var hideServiceListFunc = func() {
			pages.HidePage(FLOATING_SERVICE_LIST)
			app.SetFocus(lastFocus)
			serviceListHidden = true

		}

		if event.Modifiers() == core.APP_KEY_BINDINGS.HistoryModKey {
			var moved bool
			switch event.Key() {
			case core.APP_KEY_BINDINGS.HistoryBack:
				moved = nav.Back()
			case core.APP_KEY_BINDINGS.HistoryForward:
				moved = nav.Forward()
			default:
				return event
			}
			if moved {
				serviceListHidden = true
			}
			return nil
		}

		switch event.Key() {
		case core.APP_KEY_BINDINGS.Escape:
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
		nav.restart()
	})

	// Views take their colours when they are built, so the pages are built
	// again for a new theme. The views that stay, as they hold the log or
	// are used to switch pages, are given the colours of the theme.
	appContext.OnThemeChanged(func() {
		if err := store.Update(func(saved *settings.Settings) { saved.Theme = appContext.Theme.Name }); err != nil {
			appContext.Logger.Print(err)
		}

		rebuildThemedPages(pages, serviceViews)

		var theme = appContext.Theme
		pages.SetBackgroundColor(theme.BackgroundColour)
		core.RestyleChrome(floatingServices, theme)
		core.RestyleChrome(flexLanding, theme)
		core.RestyleChrome(errorTextArea.TextView, theme)
		commandLine.Restyle(theme)
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

	// Views take some keys when they are built, so the pages are built again
	// for a new key, but the settings, where the key was changed, which
	// shows the new keys itself
	appContext.OnKeyBindingsChanged(func() {
		for _, item := range serviceViews {
			if serviceName(item) == string(core.SETTINGS) {
				rebuildPagesForKeys(pages, serviceViews, item)
			}
		}
	})

	// Copied text is sent to the terminal too, which is reached through the
	// screen the app draws on
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		appContext.SetTerminal(screen)
		return false
	})

	app.SetRoot(pages, true)
	return app
}
