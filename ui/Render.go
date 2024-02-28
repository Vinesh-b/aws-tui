package ui

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
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

	// Purple
	contrastBackgroundColor     tcell.Color = tcell.NewHexColor(0x4B0082)
	moreContrastBackgroundColor tcell.Color = tcell.NewHexColor(0x5A00A3)
)

func initGlobalStyle() {
	tview.Borders.TopLeft = tview.BoxDrawingsLightArcDownAndRight
	tview.Borders.TopRight = tview.BoxDrawingsLightArcDownAndLeft
	tview.Borders.BottomLeft = tview.BoxDrawingsLightArcUpAndRight
	tview.Borders.BottomRight = tview.BoxDrawingsLightArcUpAndLeft

	tview.Styles.TitleColor = titleColour
	tview.Styles.BorderColor = moreContrastBackgroundColor
	tview.Styles.PrimaryTextColor = textColour
	tview.Styles.SecondaryTextColor = secondaryTextColor
	tview.Styles.TertiaryTextColor = tertiaryTextColor
	tview.Styles.PrimitiveBackgroundColor = backgroundColor
	tview.Styles.ContrastBackgroundColor = contrastBackgroundColor
	tview.Styles.MoreContrastBackgroundColor = moreContrastBackgroundColor
}

func RenderUI(config aws.Config) {
	initGlobalStyle()

	var (
		app           = tview.NewApplication()
		errorTextArea = tview.NewTextView().SetWordWrap(false)
		inAppLogger   = log.New(
			errorTextArea,
			log.Default().Prefix(),
			log.Default().Flags(),
		)

		params       = tableCreationParams{app, inAppLogger}
		serviceViews = map[viewId]tview.Primitive{
			LAMBDA:            createLambdaHomeView(app, config, inAppLogger),
			CLOUDWATCH_LOGS:   createLogsHomeView(app, config, inAppLogger),
			CLOUDWATCH_ALARMS: createAlarmsHomeView(app, config, inAppLogger),
			DYNAMODB:          createDynamoDBHomeView(app, config, inAppLogger),
		}
	)

	errorTextArea.
		SetBorder(true).
		SetTitle("Logs").
		SetTitleAlign(tview.AlignLeft)

	var currentServiceView = tview.NewFlex()
	currentServiceView.AddItem(serviceViews[LAMBDA], 0, 1, false)

	var flexHome = tview.NewFlex().SetDirection(tview.FlexColumn)
	flexHome.AddItem(tview.NewFlex().
		AddItem(currentServiceView, 0, 1, false),
		0, 1, false,
	)

	var servicesSearch = servicesHomeView()
	var flexSearch = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(servicesSearch, 3, 0, true)

	var pages = tview.NewPages().
		AddPage("ErrorLogs", errorTextArea, true, true).
		AddPage("ServiceHome", flexHome, true, true).
		AddAndSwitchToPage("Search", flexSearch, true)

	servicesSearch.SetDoneFunc(func(key tcell.Key) {
		var resultChannel = make(chan struct{})
		var table = tview.NewTable()

		switch key {
		case tcell.KeyEnter:
			var view, ok = serviceViews[viewId(servicesSearch.GetText())]
			if ok {
				go func() {
					resultChannel <- struct{}{}
				}()
				go loadData(params.App, table.Box, resultChannel, func() {
					currentServiceView.Clear()
					currentServiceView.AddItem(view, 0, 1, false)
					pages.SwitchToPage("ServiceHome")
					app.SetFocus(view)
				})
			}
		case tcell.KeyEsc:
			servicesSearch.SetText("")
		default:
			return
		}
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC:
			pages.SwitchToPage("Search")
			app.SetRoot(pages, true)
			app.SetFocus(servicesSearch)
		}
		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
