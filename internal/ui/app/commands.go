package app

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

// serviceAliases are short commands for services, beside their names
var serviceAliases = map[core.ViewId][]string{
	core.CLOUDWATCH_LOGS:    {"logs"},
	core.CLOUDWATCH_ALARMS:  {"alarms"},
	core.CLOUDWATCH_METRICS: {"metrics"},
	core.CLOUDFORMATION:     {"cfn"},
	core.DYNAMODB:           {"ddb"},
	core.S3BUCKETS:          {"s3"},
	core.STATE_MACHINES:     {"sfn"},
	core.SYSTEMS_MANAGER:    {"ssm"},
	core.PROFILE_SELECTION:  {"profile"},
}

// serviceName is the name of the service without the icon of its menu item
func serviceName(item *ServiceItem) string {
	return strings.TrimLeftFunc(item.MainText, func(r rune) bool {
		return unicode.Is(unicode.Co, r) || unicode.IsSpace(r)
	})
}

// serviceCommands are the commands that go to the service, its name in lower
// case without spaces and its aliases
func serviceCommands(item *ServiceItem) []string {
	var name = serviceName(item)
	var commands = []string{strings.ToLower(strings.ReplaceAll(name, " ", ""))}
	return append(commands, serviceAliases[core.ViewId(name)]...)
}

// messageLine shows what a command did, such as the command line
type messageLine interface {
	ShowMessage(text string)
	ShowError(text string)
}

// commands runs what is typed in the command line: :q quits, :w [file] saves
// the table that has the focus, and the name of a service goes to it
type commands struct {
	items  []*ServiceItem
	appCtx *core.AppContext
	line   messageLine
	// show shows the service, and quit stops the app
	show func(item *ServiceItem)
	quit func()
}

var (
	quitCommands  = []string{"q", "quit"}
	writeCommands = []string{"w", "write"}
)

// names are every command, in the order they are completed
func (inst *commands) names() []string {
	var names = slices.Concat(quitCommands, writeCommands)
	for _, item := range inst.items {
		names = append(names, serviceCommands(item)...)
	}
	return names
}

// Complete lists the commands that start with what is typed
func (inst *commands) Complete(text string) []string {
	var typed = strings.ToLower(strings.TrimSpace(text))
	if typed == "" || strings.ContainsRune(typed, ' ') {
		return nil
	}

	var entries []string
	for _, name := range inst.names() {
		if strings.HasPrefix(name, typed) && name != typed {
			entries = append(entries, name)
		}
	}
	return entries
}

// Run runs the command on the view that had the focus, and shows what it did
// or why it failed in the command line
func (inst *commands) Run(text string, focused tview.Primitive) {
	var command, argument, _ = strings.Cut(strings.TrimSpace(text), " ")
	command = strings.ToLower(command)
	argument = strings.TrimSpace(argument)

	switch {
	case slices.Contains(quitCommands, command):
		inst.quit()
	case slices.Contains(writeCommands, command):
		var err = inst.appCtx.SaveTable(focused, argument, func(savedPath string, err error) {
			if err != nil {
				inst.line.ShowError(err.Error())
				return
			}
			inst.line.ShowMessage("Saved to " + savedPath)
		})
		if err != nil {
			inst.line.ShowError(err.Error())
		}
	default:
		var item, err = inst.findService(command)
		if err != nil {
			inst.line.ShowError(err.Error())
			return
		}
		inst.show(item)
	}
}

// findService is the service the command names, or begins the name of when
// only one service does
func (inst *commands) findService(command string) (*ServiceItem, error) {
	var started []*ServiceItem
	for _, item := range inst.items {
		var names = serviceCommands(item)
		if slices.Contains(names, command) {
			return item, nil
		}
		if slices.ContainsFunc(names, func(name string) bool { return strings.HasPrefix(name, command) }) {
			started = append(started, item)
		}
	}

	switch len(started) {
	case 0:
		return nil, fmt.Errorf("not a command or service: %s", command)
	case 1:
		return started[0], nil
	}
	var names []string
	for _, item := range started {
		names = append(names, serviceName(item))
	}
	return nil, fmt.Errorf("%s could be %s", command, strings.Join(names, ", "))
}
