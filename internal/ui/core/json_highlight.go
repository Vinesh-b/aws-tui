package core

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

type jsonTokenKind int

const (
	jsonKey jsonTokenKind = iota
	jsonString
	jsonNumber
	jsonLiteral
)

var jsonTokenColours = map[jsonTokenKind]tcell.Color{
	jsonKey:     tcell.NewHexColor(0x61AFEF),
	jsonString:  tcell.NewHexColor(0x98C379),
	jsonNumber:  tcell.NewHexColor(0xD19A66),
	jsonLiteral: tcell.NewHexColor(0xC678DD),
}

// jsonSpan covers the runes [start, end) of a line.
type jsonSpan struct {
	start, end int
	kind       jsonTokenKind
}

// lexJsonLine tokenises one line of pretty printed json. Strings cannot span
// lines in json, so each line can be lexed on its own.
func lexJsonLine(line []rune) []jsonSpan {
	var spans []jsonSpan
	var length = len(line)

	for i := 0; i < length; {
		var c = line[i]
		switch {
		case c == '"':
			var end = i + 1
			for end < length && line[end] != '"' {
				if line[end] == '\\' {
					end++
				}
				end++
			}
			end = min(end+1, length)

			var kind = jsonString
			var next = end
			for next < length && line[next] == ' ' {
				next++
			}
			if next < length && line[next] == ':' {
				kind = jsonKey
			}
			spans = append(spans, jsonSpan{i, end, kind})
			i = end
		case c == '-' || (c >= '0' && c <= '9'):
			var end = i + 1
			for end < length && strings.ContainsRune("0123456789.eE+-", line[end]) {
				end++
			}
			spans = append(spans, jsonSpan{i, end, jsonNumber})
			i = end
		case c >= 'a' && c <= 'z':
			var end = i + 1
			for end < length && line[end] >= 'a' && line[end] <= 'z' {
				end++
			}
			switch string(line[i:end]) {
			case "true", "false", "null":
				spans = append(spans, jsonSpan{i, end, jsonLiteral})
			}
			i = end
		default:
			i++
		}
	}

	return spans
}

func runeDisplayWidth(r rune) int {
	if r < 0x7F {
		if r < 0x20 {
			return 0
		}
		return 1
	}
	return uniseg.StringWidth(string(r))
}

// syntaxTextArea recolours the cells of a text area after it has drawn them.
// Only the visible lines are lexed, and nothing is done unless highlighting is
// enabled. Rows map directly to lines, so highlighting turns line wrapping off.
type syntaxTextArea struct {
	*tview.TextArea
	jsonHighlighting bool
	lines            []string
}

func newSyntaxTextArea() *syntaxTextArea {
	return &syntaxTextArea{TextArea: tview.NewTextArea()}
}

func (inst *syntaxTextArea) setJsonHighlighting(enabled bool) {
	inst.jsonHighlighting = enabled
	if enabled {
		inst.TextArea.SetWrap(false)
		inst.lines = strings.Split(inst.TextArea.GetText(), "\n")
	} else {
		inst.TextArea.SetWrap(true)
		inst.lines = nil
	}
}

// refreshLines lexes the current text, for text that is edited
func (inst *syntaxTextArea) refreshLines() {
	if inst.jsonHighlighting {
		inst.lines = strings.Split(inst.TextArea.GetText(), "\n")
	}
}

func (inst *syntaxTextArea) setText(text string, cursorAtTheEnd bool) {
	inst.TextArea.SetText(text, cursorAtTheEnd)
	if inst.jsonHighlighting {
		inst.lines = strings.Split(text, "\n")
	}
}

func (inst *syntaxTextArea) Draw(screen tcell.Screen) {
	inst.TextArea.Draw(screen)
	if inst.jsonHighlighting {
		inst.paintJson(screen)
	}
}

func (inst *syntaxTextArea) paintJson(screen tcell.Screen) {
	var x, y, width, height = inst.GetInnerRect()
	var rowOffset, colOffset = inst.GetOffset()

	for row := 0; row < height && rowOffset+row < len(inst.lines); row++ {
		var line = []rune(inst.lines[rowOffset+row])
		var col, idx = 0, 0

		for _, span := range lexJsonLine(line) {
			for ; idx < span.start; idx++ {
				col += runeDisplayWidth(line[idx])
			}
			for ; idx < span.end; idx++ {
				var runeWidth = runeDisplayWidth(line[idx])
				var screenCol = x + col - colOffset
				col += runeWidth

				if runeWidth == 0 || screenCol < x || screenCol+runeWidth > x+width {
					continue
				}
				var str, style, _ = screen.Get(screenCol, y+row)
				screen.Put(screenCol, y+row, str, style.Foreground(jsonTokenColours[span.kind]))
			}
			if x+col-colOffset >= x+width {
				break
			}
		}
	}
}
