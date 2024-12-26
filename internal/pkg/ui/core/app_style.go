package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Global theme colours
var (
	TextColour           tcell.Color = tcell.NewHexColor(0xBFBFBF)
	SecondaryTextColor   tcell.Color = tcell.NewHexColor(0xFFFFFF)
	TertiaryTextColor    tcell.Color = tcell.NewHexColor(0xCC8B00)
	TitleColour          tcell.Color = tcell.NewHexColor(0x43B143)
	BackgroundColor      tcell.Color = tcell.NewHexColor(0x212129)
	PlaceHolderTextColor tcell.Color = tcell.NewHexColor(0x717171)

	// Grey (Default)
	ContrastBackgroundColor     tcell.Color = tcell.NewHexColor(0x303030)
	MoreContrastBackgroundColor tcell.Color = tcell.NewHexColor(0x404040)
)

func ResetGlobalStyle() {
	tview.Borders.TopLeft = tview.BoxDrawingsLightArcDownAndRight
	tview.Borders.TopRight = tview.BoxDrawingsLightArcDownAndLeft
	tview.Borders.BottomLeft = tview.BoxDrawingsLightArcUpAndRight
	tview.Borders.BottomRight = tview.BoxDrawingsLightArcUpAndLeft

	tview.Styles.TitleColor = TitleColour
	tview.Styles.BorderColor = MoreContrastBackgroundColor
	tview.Styles.PrimaryTextColor = TextColour
	tview.Styles.SecondaryTextColor = SecondaryTextColor
	tview.Styles.TertiaryTextColor = TertiaryTextColor
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = ContrastBackgroundColor
	tview.Styles.MoreContrastBackgroundColor = MoreContrastBackgroundColor
}

func ChangeColourScheme(colour tcell.Color) {
	ResetGlobalStyle()

	tview.Styles.BorderColor = colour
	tview.Styles.MoreContrastBackgroundColor = colour
}
