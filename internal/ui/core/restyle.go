package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type boxStyler interface {
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetBackgroundColor(color tcell.Color) *tview.Box
}

type flexLike interface {
	GetItemCount() int
	GetItem(index int) tview.Primitive
}

type pagesLike interface {
	GetPageNames(visibleOnly bool) []string
	GetPage(name string) tview.Primitive
}

// RestyleChrome gives views that were built with an earlier theme the colours
// of the current one, for views that cannot be built again, such as the ones
// that hold the log. It goes through the views inside flexes and pages, and
// sets the border, title and background of each, the text colour of text views
// and lists, the colours of input fields, and a FilterableList from the theme.
// Views it does not know are left as they are.
func RestyleChrome(primitive tview.Primitive, theme *AppTheme) {
	if primitive == nil {
		return
	}

	if styler, ok := primitive.(boxStyler); ok {
		styler.SetBorderColor(theme.BorderColour).
			SetTitleColor(theme.TitleColour).
			SetBackgroundColor(theme.BackgroundColour)
	}

	switch view := primitive.(type) {
	case *tview.TextView:
		view.SetTextColor(theme.PrimaryTextColour)
	case *tview.List:
		applyListTheme(view, theme)
	case *FilterableList:
		view.ApplyTheme(theme)
	case *tview.InputField:
		view.SetFieldBackgroundColor(theme.ContrastBackgroundColor).
			SetFieldTextColor(theme.PrimaryTextColour).
			SetLabelStyle(textStyle(theme.SecondaryTextColour, theme)).
			SetPlaceholderStyle(tcell.StyleDefault.
				Foreground(theme.PlaceholderTextColour).
				Background(theme.ContrastBackgroundColor))
	}

	if flex, ok := primitive.(flexLike); ok {
		for i := 0; i < flex.GetItemCount(); i++ {
			RestyleChrome(flex.GetItem(i), theme)
		}
	}
	if pages, ok := primitive.(pagesLike); ok {
		for _, name := range pages.GetPageNames(false) {
			RestyleChrome(pages.GetPage(name), theme)
		}
	}
}
