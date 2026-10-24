package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChartPoint struct {
	Time  time.Time
	Value float64
}

// barGlyphs fill a cell from the bottom in eighths.
var barGlyphs = [...]rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// TimeSeriesChart draws points as a bar chart with value and time axes. When
// there are more points than columns, each column shows the largest value
// of the points it covers so spikes stay visible.
type TimeSeriesChart struct {
	*tview.Box
	points      []ChartPoint
	message     string
	barStyle    tcell.Style
	axisStyle   tcell.Style
	FormatValue func(value float64) string
	FormatTime  func(t time.Time, span time.Duration) string
}

func NewTimeSeriesChart(title string, appContext *AppContext) *TimeSeriesChart {
	var chart = &TimeSeriesChart{
		Box:         tview.NewBox(),
		barStyle:    tcell.StyleDefault.Foreground(appContext.Theme.TertiaryTextColour),
		axisStyle:   tcell.StyleDefault.Foreground(appContext.Theme.PrimaryTextColour),
		FormatValue: FormatCompactNumber,
		FormatTime:  formatChartTime,
	}

	chart.Box.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	return chart
}

func FormatCompactNumber(value float64) string {
	var abs = math.Abs(value)
	switch {
	case abs >= 1e9:
		return strconv.FormatFloat(value/1e9, 'f', 1, 64) + "G"
	case abs >= 1e6:
		return strconv.FormatFloat(value/1e6, 'f', 1, 64) + "M"
	case abs >= 1e3:
		return strconv.FormatFloat(value/1e3, 'f', 1, 64) + "k"
	case value == math.Trunc(value):
		return strconv.FormatFloat(value, 'f', 0, 64)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func formatChartTime(t time.Time, span time.Duration) string {
	if span > 24*time.Hour {
		return t.Local().Format("01-02 15:04")
	}
	return t.Local().Format("15:04")
}

// SetData shows the points, which must be in ascending time order.
func (inst *TimeSeriesChart) SetData(points []ChartPoint) {
	inst.points = points
	inst.message = ""
}

// SetMessage replaces the chart with a centred message.
func (inst *TimeSeriesChart) SetMessage(message string) {
	inst.points = nil
	inst.message = message
}

func (inst *TimeSeriesChart) GetPoints() []ChartPoint {
	return inst.points
}

func (inst *TimeSeriesChart) GetMessage() string {
	return inst.message
}

func (inst *TimeSeriesChart) valueRange() (low float64, high float64) {
	low, high = 0, 0
	for _, p := range inst.points {
		low = math.Min(low, p.Value)
		high = math.Max(high, p.Value)
	}
	if high == low {
		high = low + 1
	}
	return low, high
}

func (inst *TimeSeriesChart) Draw(screen tcell.Screen) {
	inst.Box.DrawForSubclass(screen, inst)
	var x, y, width, height = inst.GetInnerRect()

	if len(inst.points) == 0 {
		var offset = max((width-utf8.RuneCountInString(inst.message))/2, 0)
		inst.print(screen, x+offset, y+height/2, x+width, inst.message)
		return
	}

	var low, high = inst.valueRange()
	var labels = [...]string{
		inst.FormatValue(high),
		inst.FormatValue((high + low) / 2),
		inst.FormatValue(low),
	}
	var labelWidth = 0
	for _, label := range labels {
		labelWidth = max(labelWidth, utf8.RuneCountInString(label))
	}

	var plotX = x + labelWidth + 1
	var plotWidth = width - labelWidth - 1
	var plotHeight = height - 2
	if plotWidth < 2 || plotHeight < 1 {
		return
	}

	inst.drawAxes(screen, x, y, labelWidth, plotX, plotWidth, plotHeight, labels[:])
	inst.drawBars(screen, plotX, y, plotWidth, plotHeight, low, high)
}

func (inst *TimeSeriesChart) drawAxes(
	screen tcell.Screen, x, y, labelWidth, plotX, plotWidth, plotHeight int, labels []string,
) {
	var axisX = plotX - 1
	for row := range plotHeight {
		screen.SetContent(axisX, y+row, '│', nil, inst.axisStyle)
	}
	screen.SetContent(axisX, y+plotHeight, '└', nil, inst.axisStyle)
	for col := range plotWidth {
		screen.SetContent(plotX+col, y+plotHeight, '─', nil, inst.axisStyle)
	}

	var labelRows = [...]int{0, plotHeight / 2, plotHeight - 1}
	for i, label := range labels {
		var pad = labelWidth - utf8.RuneCountInString(label)
		inst.print(screen, x+pad, y+labelRows[i], plotX, label)
	}

	var first, last = inst.points[0].Time, inst.points[len(inst.points)-1].Time
	var span = last.Sub(first)
	var firstLabel, lastLabel = inst.FormatTime(first, span), inst.FormatTime(last, span)
	var timeRow = y + plotHeight + 1
	inst.print(screen, plotX, timeRow, plotX+plotWidth, firstLabel)
	var lastLen = utf8.RuneCountInString(lastLabel)
	if plotWidth >= utf8.RuneCountInString(firstLabel)+lastLen+1 {
		inst.print(screen, plotX+plotWidth-lastLen, timeRow, plotX+plotWidth, lastLabel)
	}
}

// print writes text without going past maxX.
func (inst *TimeSeriesChart) print(screen tcell.Screen, x, y, maxX int, text string) {
	for i, r := range []rune(text) {
		if x+i >= maxX {
			return
		}
		screen.SetContent(x+i, y, r, nil, inst.axisStyle)
	}
}

func (inst *TimeSeriesChart) drawBars(
	screen tcell.Screen, plotX, plotY, plotWidth, plotHeight int, low, high float64,
) {
	var count = len(inst.points)
	for col := range plotWidth {
		var from = col * count / plotWidth
		var to = max((col+1)*count/plotWidth, from+1)

		var value = inst.points[from].Value
		for _, p := range inst.points[from:to] {
			value = math.Max(value, p.Value)
		}

		var eighths = int(math.Round((value - low) / (high - low) * float64(plotHeight*8)))
		if value > low && eighths == 0 {
			eighths = 1
		}

		for row := range plotHeight {
			var filled = min(max(eighths-row*8, 0), 8)
			if filled == 0 {
				break
			}
			var cellY = plotY + plotHeight - 1 - row
			screen.SetContent(plotX+col, cellY, barGlyphs[filled-1], nil, inst.barStyle)
		}
	}
}

// ChartSummary describes the range of the values, for use in a title.
func ChartSummary(points []ChartPoint, format func(float64) string) string {
	if len(points) == 0 {
		return ""
	}

	var low, high = points[0].Value, points[0].Value
	for _, p := range points {
		low = math.Min(low, p.Value)
		high = math.Max(high, p.Value)
	}

	return strings.TrimSpace(fmt.Sprintf(
		"last %s, min %s, max %s",
		format(points[len(points)-1].Value), format(low), format(high),
	))
}
