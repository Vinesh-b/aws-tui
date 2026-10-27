package services

import (
	"fmt"
	"strings"

	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.TextView
}

func (inst *HelpView) GetLastFocusedView() tview.Primitive {
	return inst.TextView
}

func NewHelpHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x005555))
	defer appCtx.Theme.ResetGlobalStyle()

	var textView = tview.NewTextView()
	var helpNavigation = &HelpView{TextView: textView}
	helpNavigation.SetBorderPadding(1, 1, 1, 1)

	helpNavigation.SetText(helpText())
	helpNavigation.
		SetTitle("Help").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	var serviceRootView = core.NewServiceRootView(string(core.HELP), appCtx)

	serviceRootView.AddAndSwitchToPage("Help Home", helpNavigation, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}

// helpSection is a heading and its keys, each with what it does
type helpSection struct {
	heading string
	keys    [][2]string
}

// helpText lists the keys as they are bound, so it follows the key bindings
// set in the settings file
func helpText() string {
	var keys = core.APP_KEY_BINDINGS
	var both = func(a string, b string) string { return a + " / " + b }
	// withMod is the runes pressed with the modifier, written as "Alt+k/j"
	var withMod = func(mod tcell.ModMask, first rune, others ...rune) string {
		var names = []string{core.FormatModRune(mod, first)}
		for _, r := range others {
			names = append(names, core.FormatRune(r))
		}
		return strings.Join(names, "/")
	}

	var sections = []helpSection{
		{"Navigation", [][2]string{
			{core.FormatKey(keys.ToggleServicesMenu), "Show or hide the services menu"},
			{core.FormatKey(keys.ToggleServicePages), "Show or hide the pages of the service"},
			{both(core.FormatModRune(keys.PageChangeModKey, keys.PageBack), core.FormatModRune(keys.PageChangeModKey, keys.PageForward)),
				"Go to the page on the left / right"},
			{both(core.FormatModKey(keys.HistoryModKey, keys.HistoryBack), core.FormatModKey(keys.HistoryModKey, keys.HistoryForward)),
				"Go back / forward through the pages shown"},
			{both(core.FormatKey(keys.ViewFocusUp), core.FormatKey(keys.ViewFocusDown)), "Move to the pane above / below"},
			{both(core.FormatKey(keys.ViewFocusLeft), core.FormatKey(keys.ViewFocusRight)), "Move to the pane on the left / right"},
			{withMod(keys.ViewResizeModKey, keys.MoveUpRune, keys.MoveDownRune, keys.MoveLeftRune, keys.MoveRightRune),
				"Move the split between two panes"},
			{core.FormatModRune(keys.ViewResizeModKey, keys.ViewResizeReset), "Put the splits back"},
			{core.FormatKey(keys.Escape), "Hide the floating view, such as a menu or a search"},
			{core.FormatRune(keys.Help), "Show the keys of the selected pane"},
			{core.FormatRune(keys.Quit), "Hide the services menu"},
		}},
		{"Tables", [][2]string{
			{core.FormatKey(keys.Done), "Open the selected item"},
			{core.FormatRune(keys.Reset), "Load the table again"},
			{core.FormatRune(keys.LoadMoreData), "Load more rows"},
			{core.FormatKey(keys.ClearTable), "Clear the table"},
			{core.FormatRune(keys.Find), "Search the table"},
			{both(core.FormatRune(keys.NextSearch), core.FormatRune(keys.PrevSearch)), "Go to the next / previous match"},
			{both(core.FormatRune(keys.MovePageTopRune), core.FormatRune(keys.MovePageBottomRune)), "Go to the first / last row"},
			{core.FormatRune(keys.TextCopy), "Copy the text of the cell"},
			{core.FormatRune(keys.SaveTable), "Save the table to a csv file"},
			{core.FormatRune(keys.TableQuery), "Query the table, such as DynamoDB or the marked log groups"},
			{core.FormatRune(keys.TableScan), "Scan the DynamoDB table"},
			{core.FormatRune(keys.Mark), "Mark the log group to query with Insights"},
			{core.FormatRune(keys.RemoveItem), "Unmark the log group, or delete the saved query"},
		}},
		{"Details", [][2]string{
			{core.FormatRune(keys.TextCopy), "Copy the value of the selected row"},
			{core.FormatRune(keys.ToggleRawView), "Switch between the details and the api response as json"},
			{core.FormatRune(keys.OpenLink), "Open the resource an underlined value links to"},
		}},
		{"Text views", [][2]string{
			{withMod(tcell.ModNone, keys.MoveUpRune, keys.MoveDownRune, keys.MoveLeftRune, keys.MoveRightRune),
				"Move up, down, left, right"},
			{both(core.FormatRune(keys.TextViewWordRight), core.FormatRune(keys.TextViewWordLeft)), "Move a word forward / back"},
			{both(core.FormatRune(keys.MoveLineStartRune), core.FormatRune(keys.MoveLineEndRune)), "Move to the start / end of the line"},
			{both(core.FormatKey(keys.TextViewPageUp), core.FormatKey(keys.TextViewPageDown)), "Move a page up / down"},
			{core.FormatRune(keys.TextViewVisualMode), "Select text in visual mode"},
			{core.FormatRune(keys.TextCopy), "Copy the selected text, in visual mode"},
			{core.FormatKey(keys.Escape), "Go back to normal mode"},
		}},
		{"Text inputs", [][2]string{
			{"Arrows", "Move the cursor"},
			{"Shift+Arrows", "Select text"},
			{"Ctrl+Q", "Copy the selected text (xclip is needed on Linux)"},
		}},
	}

	var text strings.Builder
	text.WriteString("The bar at the bottom of a service shows its name and pages.\n")
	for _, section := range sections {
		fmt.Fprintf(&text, "\n%s:\n", section.heading)
		for _, key := range section.keys {
			fmt.Fprintf(&text, " %-22s %s\n", key[0], key[1])
		}
	}
	return text.String()
}
