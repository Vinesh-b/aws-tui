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
	InverseTextColor     tcell.Color = tcell.NewHexColor(0x404040)
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
	tview.Styles.InverseTextColor = InverseTextColor
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = ContrastBackgroundColor
	tview.Styles.MoreContrastBackgroundColor = MoreContrastBackgroundColor
}

func ChangeColourScheme(colour tcell.Color) {
	ResetGlobalStyle()

	tview.Styles.BorderColor = colour
	tview.Styles.MoreContrastBackgroundColor = colour
}

type KeyBindings struct {
	Escape                  tcell.Key
	ToggleServicesMenu      tcell.Key
	Reset                   rune
	LoadMoreData            rune
	ClearTable              tcell.Key
	Done                    tcell.Key
	Find                    tcell.Key
	PageForward             tcell.Key
	PageBack                tcell.Key
	ViewFocusUp             tcell.Key
	ViewFocusDown           tcell.Key
	ViewFocusLeft           tcell.Key
	ViewFocusRight          tcell.Key
	ViewResizeModKey        tcell.ModMask
	MoveUpRune              rune
	MoveDownRune            rune
	ViewResizeReset         rune
	NextSearch              rune
	PrevSearch              rune
	FormFocusNext           tcell.Key
	FormFocusPrev           tcell.Key
	TableScan               tcell.Key
	TableQuery              tcell.Key
	TextViewCopy            rune
	TextViewUp              rune
	TextViewDown            rune
	MoveLeftRune            rune
	MoveRightRune           rune
	TextViewPageUp          rune
	TextViewPageDown        rune
	TextViewSelectPageUp    rune
	TextViewSelectPageDown  rune
	TextViewSelectUp        rune
	TextViewSelectDown      rune
	TextViewSelectLeft      rune
	TextViewSelectRight     rune
	TextViewWordRight       rune
	TextViewWordLeft        rune
	TextViewWordSelectRight rune
	TextViewWordSelectLeft  rune
}

var APP_KEY_BINDINGS = KeyBindings{
	Escape:                  tcell.KeyESC,
	ToggleServicesMenu:      tcell.KeyCtrlSpace,
	Reset:                   'r',
	LoadMoreData:            'n',
	ClearTable:              tcell.KeyCtrlX,
	Done:                    tcell.KeyEnter,
	Find:                    tcell.KeyCtrlF,
	PageForward:             tcell.KeyCtrlRightSq,
	PageBack:                tcell.KeyCtrlLeftSq,
	ViewFocusUp:             tcell.KeyCtrlK,
	ViewFocusDown:           tcell.KeyCtrlJ,
	ViewFocusLeft:           tcell.KeyCtrlH,
	ViewFocusRight:          tcell.KeyCtrlL,
	ViewResizeModKey:        tcell.ModAlt,
	MoveUpRune:              'k',
	MoveDownRune:            'j',
	MoveLeftRune:            'h',
	MoveRightRune:           'l',
	NextSearch:              'f',
	PrevSearch:              'F',
	FormFocusNext:           tcell.KeyTab,
	FormFocusPrev:           tcell.KeyBacktab,
	TableScan:               tcell.KeyCtrlS,
	TableQuery:              tcell.KeyCtrlQ,
	TextViewCopy:            'y',
	TextViewPageUp:          'u',
	TextViewPageDown:        'd',
	TextViewSelectPageUp:    'U',
	TextViewSelectPageDown:  'D',
	TextViewSelectUp:        'K',
	TextViewSelectDown:      'J',
	TextViewSelectLeft:      'H',
	TextViewSelectRight:     'L',
	TextViewWordLeft:        'b',
	TextViewWordRight:       'w',
	TextViewWordSelectLeft:  'B',
	TextViewWordSelectRight: 'W',
}
