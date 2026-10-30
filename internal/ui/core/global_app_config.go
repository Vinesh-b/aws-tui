package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppTheme struct {
	Name                        string
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
	ErrorTextColour             tcell.Color
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
	CommandLine        rune
	Escape             tcell.Key
	ToggleServicesMenu tcell.Key
	ToggleServicePages tcell.Key
	Reset              rune
	ClearTable         tcell.Key
	Done               tcell.Key
	Find               rune
	HistoryModKey      tcell.ModMask
	HistoryBack        tcell.Key
	HistoryForward     tcell.Key
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
	TableScan          rune
	TableQuery         rune
	Mark               rune
	Follow             rune
	TimeRange          rune
	Unmask             rune
	Download           rune
	Filter             rune
	Sort               rune
	RemoveItem         rune
	TextCopy           rune
	ToggleRawView      rune
	TextViewUp         rune
	TextViewDown       rune
	TextViewPageUp     tcell.Key
	TextViewPageDown   tcell.Key
	TextViewWordRight  rune
	TextViewWordLeft   rune
	TextViewUndo       rune
	TextViewRedo       tcell.Key
	TextViewVisualMode rune
	TextViewInsertMode rune

	// Sequences are keys typed one after the other that do what the key they
	// are named after does, like the commands of Vim. Empty turns one off.
	MoveTopSequence     string
	PageForwardSequence string
	PageBackSequence    string
	OpenLinkSequence    string
	RemoveItemSequence  string
}

// defaultKeyBindings are the keys the app starts with, before the settings
var defaultKeyBindings = APP_KEY_BINDINGS

var APP_KEY_BINDINGS = KeyBindings{
	Help:               '?',
	CommandLine:        ':',
	Escape:             tcell.KeyESC,
	ToggleServicesMenu: tcell.KeyCtrlSpace,
	ToggleServicePages: tcell.KeyCtrlP,
	Reset:              'r',
	ClearTable:         tcell.KeyCtrlX,
	Done:               tcell.KeyEnter,
	Find:               '/',
	HistoryModKey:      tcell.ModAlt,
	HistoryBack:        tcell.KeyLeft,
	HistoryForward:     tcell.KeyRight,
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
	TableScan:          's',
	TableQuery:         'q',
	Mark:               'm',
	Follow:             'f',
	TimeRange:          't',
	Unmask:             'U',
	Download:           'D',
	Filter:             'F',
	Sort:               'o',
	RemoveItem:         'x',
	TextCopy:           'y',
	ToggleRawView:      'J',
	TextViewPageUp:     tcell.KeyCtrlU,
	TextViewPageDown:   tcell.KeyCtrlD,
	TextViewWordLeft:   'b',
	TextViewWordRight:  'w',
	TextViewUndo:       'u',
	TextViewRedo:       tcell.KeyCtrlR,
	TextViewVisualMode: 'v',
	TextViewInsertMode: 'i',

	MoveTopSequence:     "gg",
	PageForwardSequence: "gt",
	PageBackSequence:    "gT",
	OpenLinkSequence:    "gx",
	RemoveItemSequence:  "dd",
}
