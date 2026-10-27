package core

import (
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
)

// appClipboard is where copied text goes: the clipboard of the system, and
// the one of the terminal the app is drawn in. It is only used on the UI
// goroutine.
type appClipboard struct {
	terminal    tcell.Screen
	writeSystem func(text string) error
	readSystem  func() (string, error)
	// systemFailed is set once copying to the system clipboard fails, which
	// is logged once, as it fails on every copy over SSH
	systemFailed bool
}

func newAppClipboard() appClipboard {
	return appClipboard{writeSystem: clipboard.WriteAll, readSystem: clipboard.ReadAll}
}

// SetTerminal sets the screen the app is drawn on, whose terminal is sent the
// text that is copied
func (inst *AppContext) SetTerminal(screen tcell.Screen) {
	inst.clipboard.terminal = screen
}

// SetSystemClipboard copies to the function in place of the clipboard of the
// system
func (inst *AppContext) SetSystemClipboard(copy func(text string) error) {
	inst.clipboard.writeSystem = copy
}

// CopyText puts the text on the clipboard. It is sent to the terminal with
// OSC 52, which reaches the clipboard of the machine the terminal runs on
// through SSH and tmux, and to the clipboard of the system with xclip, xsel,
// wl-copy or pbcopy, for terminals that do not take OSC 52.
func (inst *AppContext) CopyText(text string) {
	if inst.clipboard.terminal != nil {
		inst.clipboard.terminal.SetClipboard([]byte(text))
	}
	if err := inst.clipboard.writeSystem(text); err != nil && !inst.clipboard.systemFailed {
		inst.clipboard.systemFailed = true
		if inst.Logger != nil {
			inst.Logger.Printf("copying to the system clipboard failed, the terminal is sent the text: %v", err)
		}
	}
}

// PasteText is the text on the clipboard of the system. The clipboard of the
// terminal is not read, as most terminals do not allow it.
func (inst *AppContext) PasteText() string {
	var text, _ = inst.clipboard.readSystem()
	return text
}
