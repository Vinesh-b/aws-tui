package core

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type VimMode = int

const (
	VimModeNormal VimMode = iota
	VimModeVisual
	VimModeInsert
)

type SearchableTextView struct {
	*SearchableView
	ErrorMessageCallback func(text string, a ...any)
	HelpView             *FloatingHelpView
	textArea             *tview.TextArea
	syntaxArea           *syntaxTextArea
	lineCount            int
	searchPositions      [][]int
	searchLines          []int
	nextSearchPosition   int
	title                string
	vimMode              VimMode
	readOnly             bool
}

func NewSearchableTextView(title string, appContext *AppContext) *SearchableTextView {
	var syntaxArea = newSyntaxTextArea()
	var textArea = syntaxArea.TextArea
	var searchableView = NewSearchableView(syntaxArea, appContext)

	var view = &SearchableTextView{
		SearchableView:       searchableView,
		ErrorMessageCallback: func(text string, a ...any) {},
		HelpView:             NewFloatingHelpView(appContext),
		textArea:             textArea,
		syntaxArea:           syntaxArea,
		lineCount:            0,
		searchPositions:      nil,
		searchLines:          nil,
		nextSearchPosition:   0,
		title:                title,
		vimMode:              VimModeNormal,
		readOnly:             true,
	}

	view.textArea.
		SetClipboard(
			func(s string) { clipboard.WriteAll(s) },
			func() string {
				var res, _ = clipboard.ReadAll()
				return res
			},
		).
		SetSelectedStyle(
			tcell.Style{}.Background(appContext.Theme.MoreContrastBackgroundColor),
		)

	view.SearchableView.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	view.textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		event = view.baseKeyInuptHandler(event)
		event = view.normalModeInputHandler(event)
		event = view.visualModeInputHandler(event)
		if !view.readOnly {
			event = view.insertModeInputHandler(event)
		}
		return event
	})

	view.SearchableView.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			view.searchPositions = nil
			view.searchLines = nil
			view.nextSearchPosition = 0
			var search = searchableView.GetSearchText()
			var text = textArea.GetText()
			if len(search) == 0 || len(text) == 0 {
				view.SearchableView.SetTitle(view.title)
				return
			}
			if expr, err := regexp.Compile(search); err == nil {
				view.searchPositions = expr.FindAllStringIndex(text, -1)
				view.searchLines = matchLines(text, view.searchPositions)
				view.updateSearchPosition()
			}
		}
	})

	view.AddRuneToggleOverlay("HELP", view.HelpView, APP_KEY_BINDINGS.Help, true)
	view.HelpView.View.
		AddRune(APP_KEY_BINDINGS.Find, "Search text").
		AddRune(APP_KEY_BINDINGS.NextSearch, "Jump to next search result").
		AddRune(APP_KEY_BINDINGS.PrevSearch, "Jump to previous search result").
		AddItem(strings.Join([]string{
			FormatRune(APP_KEY_BINDINGS.MoveUpRune),
			FormatRune(APP_KEY_BINDINGS.MoveDownRune),
			FormatRune(APP_KEY_BINDINGS.MoveLeftRune),
			FormatRune(APP_KEY_BINDINGS.MoveRightRune),
		}, ","), "Move Up, Down, Left, Right", nil).
		AddRune(APP_KEY_BINDINGS.TextViewWordRight, "Move forward one word").
		AddRune(APP_KEY_BINDINGS.TextViewWordLeft, "Move back one word").
		AddKey(APP_KEY_BINDINGS.TextViewPageUp, "Move page up").
		AddKey(APP_KEY_BINDINGS.TextViewPageDown, "Move page down").
		AddRune(APP_KEY_BINDINGS.MovePageTopRune, "Move to top").
		AddRune(APP_KEY_BINDINGS.MovePageBottomRune, "Move to bottom").
		AddItem("v", "Visual mode to select Text", nil).
		AddRune(APP_KEY_BINDINGS.TextCopy, "Copy selected text in visual mode")

	return view
}

func countRune(s string, r rune, limit int) int {
	var count = 0
	for _, c := range s {
		if limit == 0 {
			break
		}
		limit--

		if c == r {
			count++
		}
	}
	return count
}

// matchLines returns countRune(text, '\n', pos[0]) for every match, computed in
// a single pass while the text is at hand. This avoids copying the whole text
// out of the text area each time the search jumps to the next match.
func matchLines(text string, positions [][]int) []int {
	var lines = make([]int, len(positions))
	var next, count, runeIdx = 0, 0, 0

	for next < len(positions) && positions[next][0] == 0 {
		next++
	}
	for _, c := range text {
		if next >= len(positions) {
			break
		}
		if c == '\n' {
			count++
		}
		runeIdx++
		for next < len(positions) && positions[next][0] == runeIdx {
			lines[next] = count
			next++
		}
	}
	// Matches whose offset is past the rune count behave like countRune and
	// see every newline in the text.
	for ; next < len(positions); next++ {
		lines[next] = count
	}

	return lines
}

func (inst *SearchableTextView) middleTextOffset(currentRow int) int {
	var _, _, _, height = inst.textArea.GetInnerRect()
	var halfVisible = height / 2

	if inst.lineCount-currentRow <= halfVisible {
		return max(inst.lineCount-height+1, 0)
	} else {
		return max(currentRow-(halfVisible-1), 0)
	}
}

func (inst *SearchableTextView) updateSearchPosition() {
	if len(inst.searchPositions) == 0 {
		return
	}

	inst.SearchableView.SetTitle(fmt.Sprintf(
		"%s [%s: %d/%d]",
		inst.title,
		inst.GetSearchText(),
		inst.nextSearchPosition+1,
		len(inst.searchPositions),
	))

	var pos = inst.searchPositions[inst.nextSearchPosition]
	inst.textArea.Select(pos[0], pos[1])
	var selectedLine = inst.searchLines[inst.nextSearchPosition] - 1
	var scrollOffset = inst.middleTextOffset(selectedLine)
	inst.textArea.SetOffset(scrollOffset, 0)
}

func (inst *SearchableTextView) GetText() string {
	return inst.textArea.GetText()
}

// SetJsonHighlighting colours json keys, strings, numbers and literals. It is
// drawn per frame for the visible lines only, but it turns line wrapping off
// and can be disabled where that cost matters.
func (inst *SearchableTextView) SetJsonHighlighting(enabled bool) {
	inst.syntaxArea.setJsonHighlighting(enabled)
}

func (inst *SearchableTextView) JsonHighlighting() bool {
	return inst.syntaxArea.jsonHighlighting
}

func (inst *SearchableTextView) SetText(text string, cursorAtTheEnd bool) {
	inst.lineCount = countRune(text, '\n', -1)
	inst.syntaxArea.setText(text, cursorAtTheEnd)
}

func (inst *SearchableTextView) baseKeyInuptHandler(event *tcell.EventKey) *tcell.EventKey {
	if event == nil {
		return nil
	}

	switch event.Key() {
	case // Disable text area default edititing key events
		tcell.KeyCtrlY, tcell.KeyCtrlZ, tcell.KeyCtrlX, tcell.KeyCtrlV,
		tcell.KeyCtrlH, tcell.KeyCtrlD, tcell.KeyCtrlK, tcell.KeyCtrlW,
		tcell.KeyCtrlU, tcell.KeyBackspace2, tcell.KeyDelete, tcell.KeyTab,
		tcell.KeyEnter:
		return nil
	case tcell.KeyEsc:
		inst.vimMode = VimModeNormal
		var _, _, end_pos = inst.textArea.GetSelection()
		inst.textArea.Select(end_pos, end_pos)
		return nil
	}
	return event
}

func (inst *SearchableTextView) normalModeInputHandler(event *tcell.EventKey) *tcell.EventKey {

	if event == nil {
		return nil
	}

	if inst.vimMode != VimModeNormal {
		return event
	}

	switch event.Key() {
	case APP_KEY_BINDINGS.TextViewPageUp:
		return tcell.NewEventKey(tcell.KeyPgUp, 0, 0)
	case APP_KEY_BINDINGS.TextViewPageDown:
		return tcell.NewEventKey(tcell.KeyPgDn, 0, 0)
	case APP_KEY_BINDINGS.TextViewRedo:
		return tcell.NewEventKey(tcell.KeyCtrlY, 0, 0)
	}

	if unicode.IsControl(event.Rune()) {
		return event
	}

	var updateSearch = false

	switch event.Rune() {
	case 'v':
		inst.vimMode = VimModeVisual
		return nil
	case 'i':
		inst.vimMode = VimModeInsert
		return nil
	case APP_KEY_BINDINGS.MoveUpRune:
		return tcell.NewEventKey(tcell.KeyUp, 0, 0)
	case APP_KEY_BINDINGS.MoveDownRune:
		return tcell.NewEventKey(tcell.KeyDown, 0, 0)
	case APP_KEY_BINDINGS.MoveLeftRune:
		return tcell.NewEventKey(tcell.KeyLeft, 0, 0)
	case APP_KEY_BINDINGS.MoveRightRune:
		return tcell.NewEventKey(tcell.KeyRight, 0, 0)
	case APP_KEY_BINDINGS.MoveLineStartRune:
		return tcell.NewEventKey(tcell.KeyCtrlA, 0, 0)
	case APP_KEY_BINDINGS.MoveLineEndRune:
		return tcell.NewEventKey(tcell.KeyCtrlE, 0, 0)
	case APP_KEY_BINDINGS.MovePageTopRune:
		inst.textArea.Select(0, 0)
		return nil
	case APP_KEY_BINDINGS.MovePageBottomRune:
		var l = inst.textArea.GetTextLength()
		inst.textArea.Select(l, l)
		return nil
	case APP_KEY_BINDINGS.TextViewWordRight:
		return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl)
	case APP_KEY_BINDINGS.TextViewWordLeft:
		return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl)
	case APP_KEY_BINDINGS.NextSearch:
		if searchCount := len(inst.searchPositions); searchCount > 0 {
			updateSearch = true
			inst.nextSearchPosition = (inst.nextSearchPosition + 1) % searchCount
		}
	case APP_KEY_BINDINGS.PrevSearch:
		if searchCount := len(inst.searchPositions); searchCount > 0 {
			updateSearch = true
			inst.nextSearchPosition = (inst.nextSearchPosition - 1 + searchCount) % searchCount
		}
	case APP_KEY_BINDINGS.TextViewUndo:
		return tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0)
	}

	if updateSearch {
		inst.updateSearchPosition()
	}
	return nil
}

func (inst *SearchableTextView) visualModeInputHandler(event *tcell.EventKey) *tcell.EventKey {
	if event == nil {
		return nil
	}

	if inst.vimMode != VimModeVisual {
		return event
	}

	switch event.Key() {
	case APP_KEY_BINDINGS.TextViewPageUp:
		return tcell.NewEventKey(tcell.KeyPgUp, 0, 0)
	case APP_KEY_BINDINGS.TextViewPageDown:
		return tcell.NewEventKey(tcell.KeyPgDn, 0, 0)
	case APP_KEY_BINDINGS.TextViewRedo:
		return tcell.NewEventKey(tcell.KeyCtrlY, 0, 0)
	}

	if unicode.IsControl(event.Rune()) {
		return event
	}

	switch event.Rune() {
	case 'v':
		inst.vimMode = VimModeNormal
		return nil
	case APP_KEY_BINDINGS.TextCopy:
		return tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)
	case APP_KEY_BINDINGS.MoveUpRune:
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModShift)
	case APP_KEY_BINDINGS.MoveDownRune:
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModShift)
	case APP_KEY_BINDINGS.MoveLeftRune:
		return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModShift)
	case APP_KEY_BINDINGS.MoveRightRune:
		return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift)
	case APP_KEY_BINDINGS.MoveLineStartRune:
		return tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModShift)
	case APP_KEY_BINDINGS.MoveLineEndRune:
		return tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModShift)
	case APP_KEY_BINDINGS.MovePageTopRune:
		var _, _, end = inst.textArea.GetSelection()
		// Select always places the cursor at the end of selection, workaround TBD
		inst.textArea.Select(end, 0)
		return nil
	case APP_KEY_BINDINGS.MovePageBottomRune:
		var _, start, _ = inst.textArea.GetSelection()
		inst.textArea.Select(start, inst.textArea.GetTextLength())
		return nil
	case APP_KEY_BINDINGS.TextViewWordRight:
		return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl|tcell.ModShift)
	case APP_KEY_BINDINGS.TextViewWordLeft:
		return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl|tcell.ModShift)
	}
	return nil
}

func (inst *SearchableTextView) insertModeInputHandler(event *tcell.EventKey) *tcell.EventKey {
	if event == nil {
		return nil
	}
	if unicode.IsControl(event.Rune()) || inst.vimMode != VimModeInsert {
		return event
	}

	return event
}
