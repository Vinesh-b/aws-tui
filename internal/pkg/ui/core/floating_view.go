package core

import (
	"github.com/rivo/tview"
)

func FloatingViewRelative(title string, p tview.Primitive, width int, height int) *tview.Flex {
	var wrapper = tview.NewFlex().
		AddItem(p, 0, 1, true)

	wrapper.
		SetBorder(true).
		SetTitle(title)

	var maxBase = 100

	if width > maxBase {
		width = maxBase
	}
	if height > maxBase {
		height = maxBase
	}

	var horizontalPadding = (maxBase - width) / 2
	var verticalPadding = (maxBase - height) / 2

	var window = tview.NewFlex().
		AddItem(nil, 0, horizontalPadding, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, verticalPadding, false).
			AddItem(wrapper, 0, height, true).
			AddItem(nil, 0, verticalPadding, false),
			0, width, true,
		).
		AddItem(nil, 0, horizontalPadding, false)

	return window
}

func FloatingView(title string, p tview.Primitive, width int, height int) *tview.Flex {
	var wrapper = tview.NewFlex().
		AddItem(p, 0, 1, true)

	wrapper.
		SetBorder(true).
		SetTitle(title)

	var window = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(wrapper, height, 1, true).
			AddItem(nil, 0, 1, false),
			width, 8, true,
		).
		AddItem(nil, 0, 1, false)

	return window
}
