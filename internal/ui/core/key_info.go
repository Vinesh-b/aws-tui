package core

import (
	"strings"
)

// KeyBindingInfo is a key binding as the Settings page lists it: its name in
// the settings file, the group of keys it is in and what it does
type KeyBindingInfo struct {
	Name        string
	Group       string
	Description string
}

// keyBindingInfos are the key bindings in the order the Settings page lists
// them, in the groups of the README
var keyBindingInfos = []KeyBindingInfo{
	{"toggleServicesMenu", "Moving around", "Show or hide the services menu"},
	{"toggleServicePages", "Moving around", "Show or hide the pages of the service"},
	{"pageBackSequence", "Moving around", "Go to the page on the left"},
	{"pageForwardSequence", "Moving around", "Go to the page on the right"},
	{"historyModKey", "Moving around", "The modifier of the history keys"},
	{"historyBack", "Moving around", "Go back through the pages shown, with the history modifier"},
	{"historyForward", "Moving around", "Go forward through the pages shown, with the history modifier"},
	{"viewFocusUp", "Moving around", "Move to the pane above"},
	{"viewFocusDown", "Moving around", "Move to the pane below"},
	{"viewFocusLeft", "Moving around", "Move to the pane on the left"},
	{"viewFocusRight", "Moving around", "Move to the pane on the right"},
	{"viewResizeModKey", "Moving around", "The modifier that moves the split between panes, with the move keys"},
	{"viewResizeReset", "Moving around", "Put the splits back, with the split modifier"},
	{"escape", "Moving around", "Hide a floating view, or leave visual mode"},
	{"help", "Moving around", "Show the keys of the selected pane"},
	{"commandLine", "Moving around", "Open the command line"},

	{"moveUp", "Moving the cursor", "Move up"},
	{"moveDown", "Moving the cursor", "Move down"},
	{"moveLeft", "Moving the cursor", "Move left"},
	{"moveRight", "Moving the cursor", "Move right"},
	{"moveTopSequence", "Moving the cursor", "Go to the top"},
	{"moveTop", "Moving the cursor", "Go to the top, through the sequence while it is set"},
	{"moveBottom", "Moving the cursor", "Go to the bottom"},
	{"moveLineStart", "Moving the cursor", "Go to the start of the line"},
	{"moveLineEnd", "Moving the cursor", "Go to the end of the line"},

	{"done", "Tables", "Open the selected item"},
	{"reset", "Tables", "Load the table again"},
	{"clearTable", "Tables", "Clear the table"},
	{"find", "Tables", "Search"},
	{"nextSearch", "Tables", "Go to the next match"},
	{"prevSearch", "Tables", "Go to the previous match"},
	{"textCopy", "Tables", "Copy the cell, the value or the selected text"},
	{"tableQuery", "Tables", "Query the table, or the marked log groups with Insights"},
	{"tableScan", "Tables", "Scan the DynamoDB table"},
	{"mark", "Tables", "Mark a log group to query with Insights"},
	{"follow", "Tables", "Follow a log stream as new events arrive"},
	{"timeRange", "Tables", "Show the next time range of the events of a log stream"},
	{"unmask", "Tables", "Show or mask the data a data protection policy masks in the events of a log stream"},
	{"download", "Tables", "Download the selected S3 object to the current directory"},
	{"filter", "Tables", "Filter the events of a log stream with a filter pattern"},
	{"sort", "Tables", "Sort by the selected column, then the other way, then as loaded"},
	{"removeItem", "Tables", "Unmark a log group, or delete a saved query or DynamoDB item"},
	{"removeItemSequence", "Tables", "Unmark a log group, or delete a saved query or DynamoDB item"},

	{"toggleRawView", "Details", "Switch between the details and the API response as JSON"},
	{"openLinkSequence", "Details", "Open the resource the value links to"},

	{"textViewWordRight", "Text views", "Move a word forward"},
	{"textViewWordLeft", "Text views", "Move a word back"},
	{"textViewPageUp", "Text views", "Move a page up"},
	{"textViewPageDown", "Text views", "Move a page down"},
	{"textViewVisualMode", "Text views", "Select text in visual mode"},
	{"textViewInsertMode", "Text views", "Insert mode, which changes nothing as text views are read-only"},
	{"textViewUndo", "Text views", "Undo"},
	{"textViewRedo", "Text views", "Redo"},
}

// KeyBindingInfos are the key bindings, in the order to list them
func KeyBindingInfos() []KeyBindingInfo {
	return keyBindingInfos
}

// CurrentKeyBinding is the key of the named binding, written as in the
// settings file
func CurrentKeyBinding(name string) string {
	return formatKeyBinding(&APP_KEY_BINDINGS, name)
}

// DefaultKeyBinding is the key the named binding has by default, written as in
// the settings file
func DefaultKeyBinding(name string) string {
	var bindings = defaultKeyBindings
	return formatKeyBinding(&bindings, name)
}

func formatKeyBinding(bindings *KeyBindings, name string) string {
	var field = keyBindingFields(bindings)[name]
	switch {
	case field.rune != nil:
		return FormatRune(*field.rune)
	case field.key != nil:
		return FormatKey(*field.key)
	case field.mod != nil:
		return strings.Join(modifierNames(*field.mod), "+")
	case field.seq != nil:
		return *field.seq
	}
	return ""
}

// KeyBindingHint is what the named binding can be set to
func KeyBindingHint(name string) string {
	var bindings = APP_KEY_BINDINGS
	var field = keyBindingFields(&bindings)[name]
	switch {
	case field.rune != nil:
		return "One character, or space"
	case field.key != nil:
		return "A key such as Esc, Enter, PgUp, Left, Ctrl+X or Ctrl+Space"
	case field.mod != nil:
		return "Alt, Ctrl or Shift"
	case field.seq != nil:
		return "Two or more characters, or nothing to turn it off"
	}
	return ""
}
