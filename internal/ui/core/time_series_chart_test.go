package core

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

var chartStart = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

func chartPoints(values ...float64) []ChartPoint {
	var points []ChartPoint
	for i, v := range values {
		points = append(points, ChartPoint{Time: chartStart.Add(time.Duration(i) * time.Minute), Value: v})
	}
	return points
}

func newTestChart() *TimeSeriesChart {
	var chart = NewTimeSeriesChart("chart", newTestAppContext())
	chart.SetBorder(false)
	chart.FormatTime = func(t time.Time, span time.Duration) string {
		if t.Equal(chartStart) {
			return "a"
		}
		return "b"
	}
	return chart
}

func drawChart(t *testing.T, chart *TimeSeriesChart, width, height int) []string {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	chart.SetRect(0, 0, width, height)
	chart.Draw(screen)

	var rows []string
	for row := range height {
		var line strings.Builder
		for col := range width {
			var str, _, _ = screen.Get(col, row)
			if str == "" {
				str = " "
			}
			line.WriteString(str)
		}
		rows = append(rows, line.String())
	}
	return rows
}

func assertRows(t *testing.T, got []string, want []string) {
	t.Helper()
	for i, wantRow := range want {
		if got[i] != wantRow {
			t.Errorf("row %d = %q, want %q", i, got[i], wantRow)
		}
	}
}

func TestTimeSeriesChart__DrawsBarsAxesAndLabels(t *testing.T) {
	var chart = newTestChart()
	chart.SetData(chartPoints(0, 4, 8, 2))

	var rows = drawChart(t, chart, 6, 5)

	assertRows(t, rows, []string{
		"8│  █ ",
		"4│ ▄█ ",
		"0│ ██▆",
		" └────",
		"  a  b",
	})
}

func TestTimeSeriesChart__ShowsTheLargestValueWhenPointsShareAColumn(t *testing.T) {
	var chart = newTestChart()
	chart.SetData(chartPoints(1, 8, 1, 1, 1, 1, 1, 1))

	var rows = drawChart(t, chart, 6, 5)

	// 8 points over 4 columns: the spike in the first pair fills column 0
	assertRows(t, rows, []string{"8│█   "})
}

func TestTimeSeriesChart__NonZeroValuesAlwaysShowABar(t *testing.T) {
	var chart = newTestChart()
	chart.SetData(chartPoints(0.001, 1000, 1000, 1000))

	var rows = drawChart(t, chart, 12, 5)

	if !strings.Contains(rows[2], "0│▁") {
		t.Errorf("bottom row = %q, want a sliver for the tiny value", rows[2])
	}
}

func TestTimeSeriesChart__MessageIsCentredWhenThereAreNoPoints(t *testing.T) {
	var chart = newTestChart()
	chart.SetMessage("No data")

	var rows = drawChart(t, chart, 11, 5)

	if got := strings.TrimSpace(rows[2]); got != "No data" || strings.Index(rows[2], "No data") != 2 {
		t.Errorf("message row = %q", rows[2])
	}
}

func TestTimeSeriesChart__SetDataClearsTheMessage(t *testing.T) {
	var chart = newTestChart()
	chart.SetMessage("Loading")

	chart.SetData(chartPoints(1))

	if chart.GetMessage() != "" || len(chart.GetPoints()) != 1 {
		t.Errorf("message=%q points=%d", chart.GetMessage(), len(chart.GetPoints()))
	}
}

func TestTimeSeriesChart__TooSmallToDrawDoesNotPanic(t *testing.T) {
	var chart = newTestChart()
	chart.SetData(chartPoints(1, 2, 3))

	drawChart(t, chart, 3, 2)
	drawChart(t, chart, 1, 1)
}

func TestTimeSeriesChart__SinglePointAndConstantValuesDraw(t *testing.T) {
	var chart = newTestChart()

	chart.SetData(chartPoints(5))
	drawChart(t, chart, 10, 6)
	chart.SetData(chartPoints(0, 0, 0))
	drawChart(t, chart, 10, 6)
}

func TestFormatCompactNumber__AbbreviatesLargeValues(t *testing.T) {
	var tests = map[float64]string{
		0:          "0",
		42:         "42",
		0.5:        "0.50",
		1500:       "1.5k",
		2_500_000:  "2.5M",
		3e9:        "3.0G",
		-1500:      "-1.5k",
		12.3456789: "12.35",
	}
	for value, want := range tests {
		if got := FormatCompactNumber(value); got != want {
			t.Errorf("FormatCompactNumber(%v) = %q, want %q", value, got, want)
		}
	}
}

func TestChartSummary__DescribesLastMinAndMax(t *testing.T) {
	var got = ChartSummary(chartPoints(3, 9, 1, 4), FormatCompactNumber)

	if got != "last 4, min 1, max 9" {
		t.Errorf("summary = %q", got)
	}
	if ChartSummary(nil, FormatCompactNumber) != "" {
		t.Error("empty data should have no summary")
	}
}

func TestTimeSeriesChart__DrawnCellsUseTheThemeBackground(t *testing.T) {
	var themeBackground = tcell.NewHexColor(0x2E3440)
	var appCtx = newTestAppContext()
	appCtx.Theme.BackgroundColour = themeBackground
	var chart = NewTimeSeriesChart("chart", appCtx)
	chart.SetBorder(false)

	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(12, 6)
	chart.SetRect(0, 0, 12, 6)
	chart.SetData(chartPoints(0, 3, 5, 8))
	chart.Draw(screen)

	for row := range 6 {
		for col := range 12 {
			var str, style, _ = screen.Get(col, row)
			if strings.TrimSpace(str) == "" {
				continue
			}
			if _, bg, _ := style.Decompose(); bg != themeBackground {
				t.Errorf("cell (%d,%d) %q background = %v, want the theme background", col, row, str, bg)
			}
		}
	}
}
