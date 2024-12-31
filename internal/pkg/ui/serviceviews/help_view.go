package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/ui/core"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.TextView
}

func (inst *HelpView) GetLastFocusedView() tview.Primitive {
	return inst.TextView
}

func NewHelpHomeView(
	app *tview.Application,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0x005555))
	defer core.ResetGlobalStyle()

	var textView = tview.NewTextView()
	var helpNavigation = &HelpView{TextView: textView}

	helpNavigation.SetText(`
Navigation:
The section at the bottom of a service page will display the
Service-Name, Page-Name and Page-Number.
 - ESC to go back to the main menu
 - Ctrl-Space to toggle floating services menu
 - Ctrl-F to toggle floating search input
 - Ctrl-J to move one pane up
 - Ctrl-K to move one pane down
 - Ctrl-H to go to page left
 - Ctrl-L to go to page right
 - Alt-J to move a horizontal pane split down
 - Alt-K to move a horizontal pane split up

Data Loading:
 - Ctrl-R to force refresh of the selected pane
 - Ctrl-N to load more data for the slected pane
 - Enter to select an item a table and load more info

Text Area:
 - Arrow keys to move cursor in a text area
 - Shift key + Arrow to select text
 - Ctrl-Q to copy text (On linux xclip is required to copy to system clipboard)
`)
	helpNavigation.
		SetTitle("Help").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	var serviceRootView = core.NewServiceRootView(app, string(HELP))

	serviceRootView.AddAndSwitchToPage("Help Home", helpNavigation, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
