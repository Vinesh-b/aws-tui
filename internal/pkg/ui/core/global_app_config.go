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

	OnFocusStyle = tcell.Style{}.
			Foreground(BackgroundColor).
			Background(TertiaryTextColor)

	OnBlurStyle = tcell.Style{}.
			Foreground(TextColour)
)

type AppTheme struct {
	PrimaryTextColour           tcell.Color
	SecondaryTextColour         tcell.Color
	TertiaryTextColour          tcell.Color
	TitleColour                 tcell.Color
	BorderColour                tcell.Color
	InverseTextColour           tcell.Color
	BackgroundColour            tcell.Color
	ContrastBackgroundColor     tcell.Color
	MoreContrastBackgroundColor tcell.Color
	PlaceholderTextColour       tcell.Color
}

func (inst *AppTheme) ResetGlobalStyle() {
	tview.Borders.TopLeft = tview.BoxDrawingsLightArcDownAndRight
	tview.Borders.TopRight = tview.BoxDrawingsLightArcDownAndLeft
	tview.Borders.BottomLeft = tview.BoxDrawingsLightArcUpAndRight
	tview.Borders.BottomRight = tview.BoxDrawingsLightArcUpAndLeft

	tview.Styles.TitleColor = inst.TitleColour
	tview.Styles.BorderColor = inst.BorderColour
	tview.Styles.PrimaryTextColor = inst.PrimaryTextColour
	tview.Styles.SecondaryTextColor = inst.SecondaryTextColour
	tview.Styles.TertiaryTextColor = inst.TertiaryTextColour
	tview.Styles.InverseTextColor = inst.InverseTextColour
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = inst.ContrastBackgroundColor
	tview.Styles.MoreContrastBackgroundColor = inst.MoreContrastBackgroundColor
}

func (inst *AppTheme) ChangeColourScheme(colour tcell.Color) {
	inst.ResetGlobalStyle()

	tview.Styles.BorderColor = colour
	tview.Styles.MoreContrastBackgroundColor = colour
}

type KeyBindings struct {
	Help                    rune
	Escape                  tcell.Key
	ToggleServicesMenu      tcell.Key
	ToggleServicePages      tcell.Key
	Reset                   rune
	LoadMoreData            rune
	ClearTable              tcell.Key
	SaveTable               rune
	Done                    tcell.Key
	Find                    tcell.Key
	PageChangeModKey        tcell.ModMask
	PageForward             rune
	PageBack                rune
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
	TableScan               rune
	TableQuery              rune
	TextCopy                rune
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
	Help:                    '?',
	Escape:                  tcell.KeyESC,
	ToggleServicesMenu:      tcell.KeyCtrlSpace,
	ToggleServicePages:      tcell.KeyCtrlP,
	Reset:                   'r',
	LoadMoreData:            'n',
	ClearTable:              tcell.KeyCtrlX,
	SaveTable:               'd',
	Done:                    tcell.KeyEnter,
	Find:                    tcell.KeyCtrlF,
	PageChangeModKey:        tcell.ModAlt,
	PageForward:             ']',
	PageBack:                '[',
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
	TableScan:               's',
	TableQuery:              'q',
	TextCopy:                'y',
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
