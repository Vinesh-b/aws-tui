package core

import (
	"fmt"
	"regexp"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchableTextView struct {
	*SearchableView
	ErrorMessageCallback func(text string, a ...any)
	HelpView             *FloatingHelpView
	textArea             *tview.TextArea
	lineCount            int
	searchPositions      [][]int
	nextSearchPosition   int
	title                string
}

func NewSearchableTextView(title string, app *tview.Application) *SearchableTextView {
	var textArea = tview.NewTextArea()
	var searchableView = NewSearchableView(textArea, app)

	var view = &SearchableTextView{
		SearchableView:       searchableView,
		ErrorMessageCallback: func(text string, a ...any) {},
		HelpView:             NewFloatingHelpView(),
		textArea:             textArea,
		lineCount:            0,
		searchPositions:      nil,
		nextSearchPosition:   0,
		title:                title,
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
			tcell.Style{}.Background(MoreContrastBackgroundColor),
		)

	view.SearchableView.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	view.textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case // Disable text area default edititing key events
			tcell.KeyCtrlY, tcell.KeyCtrlZ, tcell.KeyCtrlX, tcell.KeyCtrlV,
			tcell.KeyCtrlH, tcell.KeyCtrlD, tcell.KeyCtrlK, tcell.KeyCtrlW,
			tcell.KeyCtrlU, tcell.KeyBackspace2, tcell.KeyDelete, tcell.KeyTab,
			tcell.KeyEnter:
			return nil
		}
		if unicode.IsControl(event.Rune()) {
			return event
		}
		var updateSearch = false

		switch event.Rune() {
		case APP_KEY_BINDINGS.TextViewCopy:
			return tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)
		case APP_KEY_BINDINGS.MoveUpRune:
			return tcell.NewEventKey(tcell.KeyUp, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectUp:
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.MoveDownRune:
			return tcell.NewEventKey(tcell.KeyDown, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectDown:
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.MoveLeftRune:
			return tcell.NewEventKey(tcell.KeyLeft, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.MoveRightRune:
			return tcell.NewEventKey(tcell.KeyRight, 0, 0)
		case APP_KEY_BINDINGS.TextViewPageUp:
			return tcell.NewEventKey(tcell.KeyPgUp, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectPageUp:
			return tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewPageDown:
			return tcell.NewEventKey(tcell.KeyPgDn, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectPageDown:
			return tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewSelectRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewWordRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl)
		case APP_KEY_BINDINGS.TextViewWordSelectRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl|tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewWordLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl)
		case APP_KEY_BINDINGS.TextViewWordSelectLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl|tcell.ModShift)
		case APP_KEY_BINDINGS.NextSearch:
			if searchCount := len(view.searchPositions); searchCount > 0 {
				updateSearch = true
				view.nextSearchPosition = (view.nextSearchPosition + 1) % searchCount
			}
		case APP_KEY_BINDINGS.PrevSearch:
			if searchCount := len(view.searchPositions); searchCount > 0 {
				updateSearch = true
				view.nextSearchPosition = (view.nextSearchPosition - 1 + searchCount) % searchCount
			}
		}

		if updateSearch {
			view.updateSearchPosition()
		}

		return nil
	})

	view.SearchableView.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			view.searchPositions = nil
			view.nextSearchPosition = 0
			var search = searchableView.GetSearchText()
			var text = textArea.GetText()
			if len(search) == 0 || len(text) == 0 {
				view.SearchableView.SetTitle(view.title)
				return
			}
			if expr, err := regexp.Compile(search); err == nil {
				view.searchPositions = expr.FindAllStringIndex(text, -1)
				view.updateSearchPosition()
			}
		}
	})

	view.AddRuneToggleOverlay("HELP", view.HelpView, '?')
	view.HelpView.View.
		AddItem("Ctrl-F", "Search Text", nil).
		AddItem("f", "Jump to next search result", nil).
		AddItem("F", "Jump to previous search result", nil).
		AddItem("k,j,h,l", "Move Up, Down, Left, Right", nil).
		AddItem("w", "Move forward one word", nil).
		AddItem("b", "Move back one word", nil).
		AddItem("u", "Move page up", nil).
		AddItem("d", "Move page down", nil).
		AddItem("Shift + Move key", "Select Text", nil).
		AddItem("y", "Copy selected Text", nil)

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
	var selectedLine = countRune(inst.GetText(), '\n', pos[0]) - 1
	var scrollOffset = inst.middleTextOffset(selectedLine)
	inst.textArea.SetOffset(scrollOffset, 0)
}

func (inst *SearchableTextView) GetText() string {
	return inst.textArea.GetText()
}

func (inst *SearchableTextView) SetText(text string, cursorAtTheEnd bool) {
	inst.lineCount = countRune(text, '\n', -1)
	inst.textArea.SetText(text, cursorAtTheEnd)
}
