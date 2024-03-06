package ui

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/logging"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Global theme colours
var (
	textColour         tcell.Color = tcell.NewHexColor(0xBFBFBF)
	secondaryTextColor tcell.Color = tcell.NewHexColor(0xFFFFFF)
	tertiaryTextColor  tcell.Color = tcell.NewHexColor(0xCC8B00)
	titleColour        tcell.Color = tcell.NewHexColor(0x43B143)
	backgroundColor    tcell.Color = tcell.NewHexColor(0x212129)

	// Grey (Default)
	contrastBackgroundColor     tcell.Color = tcell.NewHexColor(0x303030)
	moreContrastBackgroundColor tcell.Color = tcell.NewHexColor(0x404040)
)

func resetGlobalStyle() {
	tview.Borders.TopLeft = tview.BoxDrawingsLightArcDownAndRight
	tview.Borders.TopRight = tview.BoxDrawingsLightArcDownAndLeft
	tview.Borders.BottomLeft = tview.BoxDrawingsLightArcUpAndRight
	tview.Borders.BottomRight = tview.BoxDrawingsLightArcUpAndLeft

	tview.Styles.TitleColor = titleColour
	tview.Styles.BorderColor = moreContrastBackgroundColor
	tview.Styles.PrimaryTextColor = textColour
	tview.Styles.SecondaryTextColor = secondaryTextColor
	tview.Styles.TertiaryTextColor = tertiaryTextColor
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = contrastBackgroundColor
	tview.Styles.MoreContrastBackgroundColor = moreContrastBackgroundColor
}

func changeColourScheme(colour tcell.Color) {
	resetGlobalStyle()

	tview.Styles.BorderColor = colour
	tview.Styles.MoreContrastBackgroundColor = colour
}

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
	resetGlobalStyle()

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

	var serviceViews = map[viewId]tview.Primitive{
		LAMBDA:                   createLambdaHomeView(app, config, inAppLogger),
		CLOUDWATCH_LOGS_GROUPS:   createLogsHomeView(app, config, inAppLogger),
		CLOUDWATCH_LOGS_INSIGHTS: createLogsInsightsHomeView(app, config, inAppLogger),
		CLOUDWATCH_ALARMS:        createAlarmsHomeView(app, config, inAppLogger),
		CLOUDWATCH_METRICS:       createMetricsHomeView(app, config, inAppLogger),
		CLOUDFORMATION:           createStacksHomeView(app, config, inAppLogger),
		DYNAMODB:                 createDynamoDBHomeView(app, config, inAppLogger),
		S3BUCKETS:                createS3bucketsHomeView(app, config, inAppLogger),

		HELP:       createHelpHomeView(app, config, inAppLogger),
		DEBUG_LOGS: errorTextArea,
	}

	errorTextArea.
		SetBorder(true).
		SetTitle("Logs").
		SetTitleAlign(tview.AlignLeft)

	var currentServiceView = tview.NewFlex()
	currentServiceView.AddItem(serviceViews[LAMBDA], 0, 1, false)

	var servicesList = servicesHomeView()
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
		var view, ok = serviceViews[viewId(serviceName)]
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
