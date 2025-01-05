package core

import (
	"regexp"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchableTextView struct {
	*SearchableView
	textArea           *tview.TextArea
	searchPositions    [][]int
	nextSearchPosition int
}

func NewSearchableTextView(title string) *SearchableTextView {
	var textArea = tview.NewTextArea()
	var searchableView = NewSearchableView(textArea)

	var view = &SearchableTextView{
		SearchableView: searchableView,
		textArea:       textArea,
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
	view.textArea.
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
		switch event.Rune() {
		case APP_KEY_BINDINGS.TextViewCopy:
			return tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)
		case APP_KEY_BINDINGS.TextViewUp:
			return tcell.NewEventKey(tcell.KeyUp, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectUp:
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewDown:
			return tcell.NewEventKey(tcell.KeyDown, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectDown:
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, 0)
		case APP_KEY_BINDINGS.TextViewSelectLeft:
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModShift)
		case APP_KEY_BINDINGS.TextViewRight:
			return tcell.NewEventKey(tcell.KeyRight, 0, 0)
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
				view.nextSearchPosition = (view.nextSearchPosition + 1) % searchCount
				var pos = view.searchPositions[view.nextSearchPosition]
				view.textArea.Select(pos[0], pos[1])
			}
		case APP_KEY_BINDINGS.PrevSearch:
			if searchCount := len(view.searchPositions); searchCount > 0 {
				view.nextSearchPosition = (view.nextSearchPosition - 1 + searchCount) % searchCount
				var pos = view.searchPositions[view.nextSearchPosition]
				view.textArea.Select(pos[0], pos[1])
			}
		}
		return nil
	})

	view.SearchableView.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			var search = searchableView.GetSearchText()
			var text = textArea.GetText()
			var expr = regexp.MustCompile(search)
			view.searchPositions = expr.FindAllStringIndex(text, -1)
		case APP_KEY_BINDINGS.Reset:
			view.textArea.Select(0, 0)
			view.searchPositions = nil
			view.nextSearchPosition = 0
		}
	})

	return view
}

func (inst *SearchableTextView) GetText() string {
	return inst.textArea.GetText()
}

func (inst *SearchableTextView) SetText(text string, cursorAtTheEnd bool) {
	inst.textArea.SetText(text, cursorAtTheEnd)
}
