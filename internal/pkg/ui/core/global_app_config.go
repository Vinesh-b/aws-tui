package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
	tview.Styles.PrimitiveBackgroundColor = inst.BackgroundColour
	tview.Styles.ContrastBackgroundColor = inst.ContrastBackgroundColor
	tview.Styles.MoreContrastBackgroundColor = inst.MoreContrastBackgroundColor
}

func (inst *AppTheme) GetFocusFormItemStyle() tcell.Style {
	return tcell.Style{}.
		Foreground(inst.InverseTextColour).
		Background(inst.TertiaryTextColour)
}

func (inst *AppTheme) GetBlurFormItemStyle() tcell.Style {
	return tcell.Style{}.Foreground(inst.PrimaryTextColour)
}

func (inst *AppTheme) ChangeColourScheme(colour tcell.Color) {
	inst.ResetGlobalStyle()

	tview.Styles.BorderColor = colour
	tview.Styles.MoreContrastBackgroundColor = colour
}

type KeyBindings struct {
	Help               rune
	Escape             tcell.Key
	ToggleServicesMenu tcell.Key
	ToggleServicePages tcell.Key
	Reset              rune
	LoadMoreData       rune
	ClearTable         tcell.Key
	SaveTable          rune
	Done               tcell.Key
	Find               rune
	PageChangeModKey   tcell.ModMask
	PageForward        rune
	PageBack           rune
	ViewFocusUp        tcell.Key
	ViewFocusDown      tcell.Key
	ViewFocusLeft      tcell.Key
	ViewFocusRight     tcell.Key
	ViewResizeModKey   tcell.ModMask
	ViewResizeReset    rune
	MoveUpRune         rune
	MoveDownRune       rune
	MoveLeftRune       rune
	MoveRightRune      rune
	MoveLineStartRune  rune
	MoveLineEndRune    rune
	MovePageTopRune    rune
	MovePageBottomRune rune
	NextSearch         rune
	PrevSearch         rune
	FormFocusNext      tcell.Key
	FormFocusPrev      tcell.Key
	TableScan          rune
	TableQuery         rune
	TextCopy           rune
	TextViewUp         rune
	TextViewDown       rune
	TextViewPageUp     tcell.Key
	TextViewPageDown   tcell.Key
	TextViewWordRight  rune
	TextViewWordLeft   rune
	TextViewUndo       rune
	TextViewRedo       tcell.Key
}

var APP_KEY_BINDINGS = KeyBindings{
	Help:               '?',
	Escape:             tcell.KeyESC,
	ToggleServicesMenu: tcell.KeyCtrlM,
	ToggleServicePages: tcell.KeyCtrlP,
	Reset:              'r',
	LoadMoreData:       'n',
	ClearTable:         tcell.KeyCtrlX,
	SaveTable:          'd',
	Done:               tcell.KeyEnter,
	Find:               '/',
	PageChangeModKey:   tcell.ModAlt,
	PageForward:        ']',
	PageBack:           '[',
	ViewFocusUp:        tcell.KeyCtrlK,
	ViewFocusDown:      tcell.KeyCtrlJ,
	ViewFocusLeft:      tcell.KeyCtrlH,
	ViewFocusRight:     tcell.KeyCtrlL,
	ViewResizeModKey:   tcell.ModAlt,
	ViewResizeReset:    '0',
	MoveUpRune:         'k',
	MoveDownRune:       'j',
	MoveLeftRune:       'h',
	MoveRightRune:      'l',
	MoveLineStartRune:  '^',
	MoveLineEndRune:    '$',
	MovePageTopRune:    'g',
	MovePageBottomRune: 'G',
	NextSearch:         'n',
	PrevSearch:         'N',
	FormFocusNext:      tcell.KeyTab,
	FormFocusPrev:      tcell.KeyBacktab,
	TableScan:          's',
	TableQuery:         'q',
	TextCopy:           'y',
	TextViewPageUp:     tcell.KeyCtrlU,
	TextViewPageDown:   tcell.KeyCtrlD,
	TextViewWordLeft:   'b',
	TextViewWordRight:  'w',
	TextViewUndo:       'u',
	TextViewRedo:       tcell.KeyCtrlR,
}
