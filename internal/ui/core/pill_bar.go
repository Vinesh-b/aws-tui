package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

const (
	pillLeft  = "🭃"
	pillRight = "🭎"
)

// The marks a scrolled bar shows on the side that has pills out of view
const (
	scrolledLeftMark  = "‹ "
	scrolledRightMark = " ›"
)

// pillPosition is where a pill was drawn, in cells from the start of the bar
type pillPosition struct {
	index int
	start int
	end   int
}

// PillBar draws names as a row of rounded pills, with the current one
// highlighted, and finds the pill at a position for mouse clicks. It is drawn
// by the view that holds it, such as the bar of a tab view. When the pills
// don't fit, the bar scrolls to keep the current one in view.
type PillBar struct {
	pills         []string
	current       int
	gap           int
	activeStyle   tcell.Style
	inactiveStyle tcell.Style
	background    tcell.Color
	// first is the first pill drawn when the bar is scrolled
	first int
	// drawn is where the pills were drawn the last time, for clicks. Before the
	// bar is drawn clicks find the pills as if they all fit.
	drawn []pillPosition
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
	inst.first = 0
	inst.drawn = nil
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

// Width is the number of cells the bar takes when every pill fits.
func (inst *PillBar) Width() int {
	return inst.widthOf(0, len(inst.pills)-1)
}

// widthOf is the number of cells the pills from first to last take
func (inst *PillBar) widthOf(first int, last int) int {
	var width = 0
	for idx := first; idx <= last; idx++ {
		if idx > first {
			width += inst.gap
		}
		width += pillWidth(inst.pills[idx])
	}
	return width
}

// scrollTo moves the first pill so the current one fits in the width, and
// shows as many of the pills before it as fit after the last one is in view.
func (inst *PillBar) scrollTo(width int) {
	var last = len(inst.pills) - 1
	var current = max(0, min(inst.current, last))

	inst.first = max(0, min(inst.first, current))
	for inst.first < current && inst.widthOf(inst.first, current) > width {
		inst.first++
	}
	for inst.first > 0 && inst.widthOf(inst.first-1, last) <= width {
		inst.first--
	}
}

// Draw draws the bar from x, cut off at maxX, and returns the x after it.
func (inst *PillBar) Draw(screen tcell.Screen, x int, y int, maxX int) int {
	inst.drawn = nil
	if len(inst.pills) == 0 {
		return x
	}

	var start = x
	var markStyle = inst.inactiveStyle.Background(inst.background)
	var markWidth = uniseg.StringWidth(scrolledLeftMark) + uniseg.StringWidth(scrolledRightMark)

	var first = 0
	var scrolled = inst.Width() > maxX-x
	if scrolled {
		inst.scrollTo(maxX - x - markWidth)
		first = inst.first
		if first > 0 {
			x = drawText(screen, x, y, maxX, scrolledLeftMark, markStyle)
		}
	} else {
		inst.first = 0
	}

	// A scrolled bar draws only the pills that fit, and leaves room to show
	// there are more
	var limit = maxX
	if scrolled {
		limit = maxX - uniseg.StringWidth(scrolledRightMark)
	}

	var last = first - 1
	for idx := first; idx < len(inst.pills); idx++ {
		var name = inst.pills[idx]
		var pillX = x
		if idx > first {
			pillX += inst.gap
		}
		if scrolled && idx > first && pillX+pillWidth(name) > limit {
			break
		}

		var style = inst.inactiveStyle
		if idx == inst.current {
			style = inst.activeStyle
		}
		var end = drawPill(screen, min(pillX, maxX), y, limit, name, style, inst.background)
		inst.drawn = append(inst.drawn, pillPosition{index: idx, start: pillX - start, end: end - start})
		x = end
		last = idx
	}

	if scrolled && last < len(inst.pills)-1 {
		x = drawText(screen, x, y, maxX, scrolledRightMark, markStyle)
	}
	return x
}

// PillAt returns the index of the pill at offset cells from the left of the
// bar, and false when there is none, such as in a gap.
func (inst *PillBar) PillAt(offset int) (int, bool) {
	var positions = inst.drawn
	if positions == nil {
		var start = 0
		for idx, name := range inst.pills {
			var end = start + pillWidth(name)
			positions = append(positions, pillPosition{index: idx, start: start, end: end})
			start = end + inst.gap
		}
	}

	for _, position := range positions {
		if offset >= position.start && offset < position.end {
			return position.index, true
		}
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
