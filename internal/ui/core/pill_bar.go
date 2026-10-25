package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

const (
	pillLeft  = "🭃"
	pillRight = "🭎"
)

// PillBar draws names as a row of rounded pills, with the current one
// highlighted, and finds the pill at a position for mouse clicks. It is drawn
// by the view that holds it, such as the bar of a tab view.
type PillBar struct {
	pills         []string
	current       int
	gap           int
	activeStyle   tcell.Style
	inactiveStyle tcell.Style
	background    tcell.Color
}

// NewPillBar makes a bar with the colours of the theme, taken when it is built
// like the ones of other views.
func NewPillBar(theme *AppTheme) *PillBar {
	return &PillBar{
		pills: []string{},
		activeStyle: tcell.StyleDefault.
			Background(theme.MoreContrastBackgroundColor).
			Foreground(theme.SecondaryTextColour),
		inactiveStyle: tcell.StyleDefault.
			Background(theme.ContrastBackgroundColor).
			Foreground(theme.PrimaryTextColour),
		background: theme.BackgroundColour,
	}
}

// SetPills sets the names shown, in order.
func (inst *PillBar) SetPills(names []string) *PillBar {
	inst.pills = append([]string{}, names...)
	return inst
}

func (inst *PillBar) GetPills() []string {
	return inst.pills
}

// SetCurrent sets the index of the highlighted pill.
func (inst *PillBar) SetCurrent(index int) *PillBar {
	inst.current = index
	return inst
}

func (inst *PillBar) GetCurrent() int {
	return inst.current
}

// SetGap sets the number of cells between pills.
func (inst *PillBar) SetGap(gap int) *PillBar {
	inst.gap = gap
	return inst
}

// Width is the number of cells the bar takes.
func (inst *PillBar) Width() int {
	var width = 0
	for idx, name := range inst.pills {
		if idx > 0 {
			width += inst.gap
		}
		width += pillWidth(name)
	}
	return width
}

// Draw draws the bar from x, cut off at maxX, and returns the x after it.
func (inst *PillBar) Draw(screen tcell.Screen, x int, y int, maxX int) int {
	for idx, name := range inst.pills {
		if idx > 0 {
			x = min(x+inst.gap, maxX)
		}
		var style = inst.inactiveStyle
		if idx == inst.current {
			style = inst.activeStyle
		}
		x = drawPill(screen, x, y, maxX, name, style, inst.background)
	}
	return x
}

// PillAt returns the index of the pill at offset cells from the left of the
// bar, and false when there is none, such as in a gap.
func (inst *PillBar) PillAt(offset int) (int, bool) {
	var start = 0
	for idx, name := range inst.pills {
		var end = start + pillWidth(name)
		if offset >= start && offset < end {
			return idx, true
		}
		start = end + inst.gap
	}
	return 0, false
}

func pillWidth(name string) int {
	return uniseg.StringWidth(pillLeft) + uniseg.StringWidth(name) + uniseg.StringWidth(pillRight)
}

// drawPill draws text in a pill of the background of style, with its rounded
// edges on the background around it, and returns the x after it.
func drawPill(
	screen tcell.Screen, x int, y int, maxX int, text string, style tcell.Style, background tcell.Color,
) int {
	var _, pillBackground, _ = style.Decompose()
	var edgeStyle = tcell.StyleDefault.Foreground(pillBackground).Background(background)
	x = drawText(screen, x, y, maxX, pillLeft, edgeStyle)
	x = drawText(screen, x, y, maxX, text, style)
	return drawText(screen, x, y, maxX, pillRight, edgeStyle)
}

// drawText draws text from x, up to maxX, and returns the x after it.
func drawText(screen tcell.Screen, x int, y int, maxX int, text string, style tcell.Style) int {
	var state = -1
	var cluster string
	for len(text) > 0 {
		var width int
		cluster, text, width, state = uniseg.FirstGraphemeClusterInString(text, state)
		if x+width > maxX {
			return maxX
		}
		var runes = []rune(cluster)
		screen.SetContent(x, y, runes[0], runes[1:], style)
		x += width
	}
	return x
}
