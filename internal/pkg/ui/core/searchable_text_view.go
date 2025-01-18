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
	textArea             *tview.TextArea
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
		textArea:             textArea,
		searchPositions:      nil,
		nextSearchPosition:   0,
		title:                title,
	}

	view.textArea.SetClipboard(
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

	var updateSearchPosition = func() {
		if len(view.searchPositions) == 0 {
			return
		}

		var pos = view.searchPositions[view.nextSearchPosition]
		view.SearchableView.SetTitle(fmt.Sprintf(
			"%s [%s: %d/%d]",
			view.title,
			view.GetSearchText(),
			view.nextSearchPosition+1,
			len(view.searchPositions),
		))
		var selectedLine = countRune(view.GetText(), '\n', pos[0])
		// Provide 5 lines of previous text for context
		view.textArea.SetOffset(selectedLine-5, 0)
		view.textArea.Select(pos[0], pos[1])

	}

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
			updateSearchPosition()
		}

		return nil
	})

	view.SearchableView.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			var search = searchableView.GetSearchText()
			var text = textArea.GetText()
			if len(search) == 0 || len(text) == 0 {
				return
			}
			if expr, err := regexp.Compile(search); err == nil {
				view.searchPositions = expr.FindAllStringIndex(text, -1)
				updateSearchPosition()
			}
		case APP_KEY_BINDINGS.Reset:
			view.textArea.Select(0, 0)
			view.searchPositions = nil
			view.nextSearchPosition = 0
		}
	})

	return view
}

func countRune(s string, r rune, limit int) int {
	var count = 0
	for _, c := range s {
		if limit <= 0 {
			break
		}
		limit--

		if c == r {
			count++
		}
	}
	return count
}

func (inst *SearchableTextView) GetText() string {
	return inst.textArea.GetText()
}

func (inst *SearchableTextView) SetText(text string, cursorAtTheEnd bool) {
	inst.textArea.SetText(text, cursorAtTheEnd)
}
