package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	COMMAND_LINE_PAGE  = "COMMAND_LINE"
	commandLineInput   = "INPUT"
	commandLineMessage = "MESSAGE"
	commandLinePrompt  = ":"
)

// CommandLine is the line at the bottom of the app where a command is typed
// after a colon, as in Vim, such as :q or :w. What the command did is shown in
// the same line until the next key. It is only used on the UI goroutine.
type CommandLine struct {
	// Flex is the page the line is shown in, over the bottom line of the app
	*tview.Flex
	row      *tview.Pages
	input    *tview.InputField
	message  *tview.TextView
	appCtx   *AppContext
	pages    *tview.Pages
	previous tview.Primitive
	open     bool
	showing  bool
	run      func(command string, focused tview.Primitive)
}

// NewCommandLine makes the line and adds it as a hidden page of the pages of
// the app
func NewCommandLine(appCtx *AppContext, pages *tview.Pages) *CommandLine {
	var view = &CommandLine{
		Flex:    tview.NewFlex().SetDirection(tview.FlexRow),
		row:     tview.NewPages(),
		input:   tview.NewInputField().SetLabel(commandLinePrompt),
		message: tview.NewTextView(),
		appCtx:  appCtx,
		pages:   pages,
		run:     func(string, tview.Primitive) {},
	}

	view.row.
		AddPage(commandLineInput, view.input, true, true).
		AddPage(commandLineMessage, view.message, true, false)
	view.Flex.
		AddItem(nil, 0, 1, false).
		AddItem(view.row, 1, 0, true)
	view.Restyle(appCtx.Theme)

	view.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			view.submit()
		case tcell.KeyEscape:
			view.close()
		}
	})
	// Tab puts the completion in, and Enter runs it as in Vim. The input is
	// changed once it has handled the key, as it holds the completions while
	// it does and changing it lists them again. Queuing waits for the UI
	// goroutine, which is the one handling the key, so it is queued from
	// another.
	view.input.SetAutocompletedFunc(func(text string, index int, source int) bool {
		if source == tview.AutocompletedNavigate {
			return false
		}
		go appCtx.App.QueueUpdateDraw(func() {
			view.input.SetText(text)
			if source == tview.AutocompletedEnter {
				view.submit()
			}
		})
		return true
	})

	pages.AddPage(COMMAND_LINE_PAGE, view, true, false)
	return view
}

// submit closes the line and runs the command typed
func (inst *CommandLine) submit() {
	if !inst.open {
		return
	}
	var command = inst.input.GetText()
	var focused = inst.previous
	inst.close()
	if len(command) > 0 {
		inst.run(command, focused)
	}
}

// SetRunFunc sets what runs a command, which is given the view that had the
// focus when the line was opened
func (inst *CommandLine) SetRunFunc(run func(command string, focused tview.Primitive)) {
	inst.run = run
}

// SetCompleteFunc sets what lists the commands that complete what is typed
func (inst *CommandLine) SetCompleteFunc(complete func(text string) []string) {
	inst.input.SetAutocompleteFunc(complete)
}

// HandleEvent opens the line with its key outside the views text is typed
// into, and hides what a command showed on the next key. It returns the event
// to pass on, which is nil when the line was opened.
func (inst *CommandLine) HandleEvent(event *tcell.EventKey, focused tview.Primitive) *tcell.EventKey {
	if inst.showing {
		inst.hide()
	}
	if inst.open || isTextEntry(focused) {
		return event
	}
	if event.Key() == tcell.KeyRune && event.Rune() == APP_KEY_BINDINGS.CommandLine &&
		event.Modifiers() == tcell.ModNone {
		inst.Open(focused)
		return nil
	}
	return event
}

// Open shows the line to type a command, which runs on the focused view
func (inst *CommandLine) Open(focused tview.Primitive) {
	inst.previous = focused
	inst.open = true
	inst.showing = false
	inst.input.SetText("")
	inst.row.SwitchToPage(commandLineInput)
	inst.show()
	inst.appCtx.App.SetFocus(inst.input)
}

func (inst *CommandLine) IsOpen() bool {
	return inst.open
}

// ShowMessage shows what a command did, until the next key
func (inst *CommandLine) ShowMessage(text string) {
	inst.showText(text, inst.appCtx.Theme.PrimaryTextColour)
}

// ShowError shows why a command failed, until the next key
func (inst *CommandLine) ShowError(text string) {
	inst.showText(text, inst.appCtx.Theme.ErrorTextColour)
}

func (inst *CommandLine) showText(text string, colour tcell.Color) {
	inst.message.SetText(text).SetTextColor(colour)
	inst.row.SwitchToPage(commandLineMessage)
	inst.showing = true

	// Showing the page would move the focus to it
	var focused = inst.appCtx.App.GetFocus()
	inst.show()
	inst.appCtx.App.SetFocus(focused)
}

func (inst *CommandLine) show() {
	inst.pages.ShowPage(COMMAND_LINE_PAGE)
	inst.pages.SendToFront(COMMAND_LINE_PAGE)
}

// close hides the line and gives the focus back to the view it was opened on
func (inst *CommandLine) close() {
	inst.open = false
	inst.pages.HidePage(COMMAND_LINE_PAGE)
	if inst.previous != nil {
		inst.appCtx.App.SetFocus(inst.previous)
	}
}

// hide hides what a command showed, leaving the focus where it is
func (inst *CommandLine) hide() {
	inst.showing = false
	var focused = inst.appCtx.App.GetFocus()
	inst.pages.HidePage(COMMAND_LINE_PAGE)
	inst.appCtx.App.SetFocus(focused)
}

// Restyle gives the line the colours of the theme
func (inst *CommandLine) Restyle(theme *AppTheme) {
	inst.input.
		SetFieldBackgroundColor(theme.BackgroundColour).
		SetFieldTextColor(theme.PrimaryTextColour).
		SetLabelColor(theme.PrimaryTextColour).
		SetBackgroundColor(theme.BackgroundColour)
	inst.input.SetAutocompleteStyles(
		theme.ContrastBackgroundColor,
		tcell.StyleDefault.Foreground(theme.PrimaryTextColour).Background(theme.ContrastBackgroundColor),
		tcell.StyleDefault.Foreground(theme.InverseTextColour).Background(theme.PrimaryTextColour),
	)
	inst.message.SetBackgroundColor(theme.BackgroundColour)
}
