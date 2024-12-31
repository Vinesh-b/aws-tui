package core

import "github.com/rivo/tview"

type PaginatorView struct {
	*tview.Flex
	PageCounterView *tview.TextView
	PageNameView    *tview.TextView
}

func CreatePaginatorView(service string) PaginatorView {
	var pageCount = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetTextColor(TertiaryTextColor)

	var pageName = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetTextColor(TertiaryTextColor)

	var serviceName = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(TertiaryTextColor).
		SetText(service)

	var rootView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(serviceName, 0, 1, false).
		AddItem(pageName, 0, 1, false).
		AddItem(pageCount, 0, 1, false)
	rootView.SetBorderPadding(0, 0, 1, 1)

	return PaginatorView{
		Flex:            rootView,
		PageCounterView: pageCount,
		PageNameView:    pageName,
	}
}
