package core

import (
	"slices"
	"strings"
)

// KeyBindingInfo is a key binding as the Settings page lists it: its name in
// the settings file, the group of keys it is in and what it does
type KeyBindingInfo struct {
	Name  string
	Group string
	// Service is the service whose pages use the key, or "" for a key every
	// service uses
	Service     string
	Description string
}

// Section is the group the Settings page lists the key under: "General" with
// the group of the key, or the service
func (info KeyBindingInfo) Section() string {
	if info.Service != "" {
		return info.Service
	}
	return "General: " + info.Group
}

// forService marks the bindings as the keys of one service. A key only a
// service uses is added to a list of its own below, and the Settings page then
// lists it under that service.
func forService(service string, infos ...KeyBindingInfo) []KeyBindingInfo {
	for i := range infos {
		infos[i].Service = service
	}
	return infos
}

// keyBindingInfos are the key bindings in the order the Settings page lists
// them, in the groups of the README
var keyBindingInfos = slices.Concat(generalKeyBindingInfos,
	forService("CloudWatch", serviceKeyBindingInfosCloudWatch...),
	forService("DynamoDB", serviceKeyBindingInfosDynamoDB...),
	forService("S3", serviceKeyBindingInfosS3...),
)

var generalKeyBindingInfos = []KeyBindingInfo{
	{Name: "toggleServicesMenu", Group: "Moving around", Description: "Show or hide the services menu"},
	{Name: "toggleServicePages", Group: "Moving around", Description: "Show or hide the pages of the service"},
	{Name: "pageBackSequence", Group: "Moving around", Description: "Go to the page on the left"},
	{Name: "pageForwardSequence", Group: "Moving around", Description: "Go to the page on the right"},
	{Name: "historyModKey", Group: "Moving around", Description: "The modifier of the history keys"},
	{Name: "historyBack", Group: "Moving around", Description: "Go back through the pages shown, with the history modifier"},
	{Name: "historyForward", Group: "Moving around", Description: "Go forward through the pages shown, with the history modifier"},
	{Name: "viewFocusUp", Group: "Moving around", Description: "Move to the pane above"},
	{Name: "viewFocusDown", Group: "Moving around", Description: "Move to the pane below"},
	{Name: "viewFocusLeft", Group: "Moving around", Description: "Move to the pane on the left"},
	{Name: "viewFocusRight", Group: "Moving around", Description: "Move to the pane on the right"},
	{Name: "viewResizeModKey", Group: "Moving around", Description: "The modifier that moves the split between panes, with the move keys"},
	{Name: "viewResizeReset", Group: "Moving around", Description: "Put the splits back, with the split modifier"},
	{Name: "escape", Group: "Moving around", Description: "Hide a floating view, or leave visual mode"},
	{Name: "help", Group: "Moving around", Description: "Show the keys of the selected pane"},
	{Name: "commandLine", Group: "Moving around", Description: "Open the command line"},

	{Name: "moveUp", Group: "Moving the cursor", Description: "Move up"},
	{Name: "moveDown", Group: "Moving the cursor", Description: "Move down"},
	{Name: "moveLeft", Group: "Moving the cursor", Description: "Move left"},
	{Name: "moveRight", Group: "Moving the cursor", Description: "Move right"},
	{Name: "moveTopSequence", Group: "Moving the cursor", Description: "Go to the top"},
	{Name: "moveTop", Group: "Moving the cursor", Description: "Go to the top, through the sequence while it is set"},
	{Name: "moveBottom", Group: "Moving the cursor", Description: "Go to the bottom"},
	{Name: "moveLineStart", Group: "Moving the cursor", Description: "Go to the start of the line"},
	{Name: "moveLineEnd", Group: "Moving the cursor", Description: "Go to the end of the line"},

	{Name: "done", Group: "Tables", Description: "Open the selected item"},
	{Name: "reset", Group: "Tables", Description: "Load the table again"},
	{Name: "clearTable", Group: "Tables", Description: "Clear the table"},
	{Name: "find", Group: "Tables", Description: "Search"},
	{Name: "nextSearch", Group: "Tables", Description: "Go to the next match"},
	{Name: "prevSearch", Group: "Tables", Description: "Go to the previous match"},
	{Name: "textCopy", Group: "Tables", Description: "Copy the cell, the value or the selected text"},
	{Name: "tableQuery", Group: "Tables", Description: "Query the table, or the marked log groups with Insights"},
	{Name: "sort", Group: "Tables", Description: "Sort by the selected column, then the other way, then as loaded"},
	{Name: "removeItem", Group: "Tables", Description: "Unmark a log group, or delete a saved query or DynamoDB item"},
	{Name: "removeItemSequence", Group: "Tables", Description: "Unmark a log group, or delete a saved query or DynamoDB item"},

	{Name: "toggleRawView", Group: "Details", Description: "Switch between the details and the API response as JSON"},
	{Name: "openLinkSequence", Group: "Details", Description: "Open the resource the value links to"},

	{Name: "textViewWordRight", Group: "Text views", Description: "Move a word forward"},
	{Name: "textViewWordLeft", Group: "Text views", Description: "Move a word back"},
	{Name: "textViewPageUp", Group: "Text views", Description: "Move a page up"},
	{Name: "textViewPageDown", Group: "Text views", Description: "Move a page down"},
	{Name: "textViewVisualMode", Group: "Text views", Description: "Select text in visual mode"},
	{Name: "textViewInsertMode", Group: "Text views", Description: "Insert mode, which changes nothing as text views are read-only"},
	{Name: "textViewUndo", Group: "Text views", Description: "Undo"},
	{Name: "textViewRedo", Group: "Text views", Description: "Redo"},
}

var serviceKeyBindingInfosCloudWatch = []KeyBindingInfo{
	{Name: "mark", Group: "Tables", Description: "Mark a log group to query with Insights"},
	{Name: "follow", Group: "Tables", Description: "Follow a log stream as new events arrive"},
	{Name: "timeRange", Group: "Tables", Description: "Show the next time range of the events of a log stream"},
	{Name: "unmask", Group: "Tables", Description: "Show or mask the data a data protection policy masks in the events of a log stream"},
	{Name: "filter", Group: "Tables", Description: "Filter the events of a log stream with a filter pattern"},
}

var serviceKeyBindingInfosDynamoDB = []KeyBindingInfo{
	{Name: "tableScan", Group: "Tables", Description: "Scan the DynamoDB table"},
}

var serviceKeyBindingInfosS3 = []KeyBindingInfo{
	{Name: "download", Group: "Tables", Description: "Download the selected S3 object to the current directory"},
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
