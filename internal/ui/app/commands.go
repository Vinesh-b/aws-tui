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
	core.SECRETS_MANAGER:    {"secrets"},
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

// pageCommand is the name of a page of a service as it is typed, in lower case
// without spaces
func pageCommand(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

// startCommand is the command that goes to the service, on the page if there is
// one
func startCommand(item *ServiceItem, page string) string {
	var command = serviceCommands(item)[0]
	if page == "" {
		return command
	}
	return command + " " + pageCommand(page)
}

// duplicateArgs are the arguments that start the app with the profile and
// region, on the page the command goes to. Whatever is empty is left out, so
// the new instance finds it the way a new start does.
func duplicateArgs(profile string, region string, start string) []string {
	var args []string
	for _, flag := range [][2]string{{"-profile", profile}, {"-region", region}, {"-open", start}} {
		if flag[1] != "" {
			args = append(args, flag[0], flag[1])
		}
	}
	return args
}

// commands runs what is typed in the command line: :q quits, :w [file] saves
// the table that has the focus, :dup opens the app again in a new terminal, and
// the name of a service goes to it, as does the name followed by one of its
// pages, such as :logs streams
type commands struct {
	items  []*ServiceItem
	appCtx *core.AppContext
	line   messageLine
	// show shows the service, on the page if there is one, and quit stops the
	// app
	show func(item *ServiceItem, page string)
	quit func()
	// duplicate opens the app in a new terminal where this one is
	duplicate func() error
}

var (
	quitCommands  = []string{"q", "quit"}
	writeCommands = []string{"w", "write"}
	// duplicateCommands open the app in a new terminal, like duplicating a tab
	duplicateCommands = []string{"dup", "duplicate"}
)

// names are every command, in the order they are completed
func (inst *commands) names() []string {
	var names = slices.Concat(quitCommands, writeCommands, duplicateCommands)
	for _, item := range inst.items {
		names = append(names, serviceCommands(item)...)
	}
	return names
}

// Complete lists the commands that start with what is typed
func (inst *commands) Complete(text string) []string {
	var typed = strings.ToLower(strings.TrimLeft(text, " "))
	if command, typedPage, found := strings.Cut(typed, " "); found {
		return inst.completePage(command, strings.TrimLeft(typedPage, " "))
	}
	typed = strings.TrimSpace(typed)
	if typed == "" {
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

// completePage lists the pages of the service the command names that start
// with what is typed, each with the command in front as it replaces the text
func (inst *commands) completePage(command string, typedPage string) []string {
	var item, err = inst.findService(command)
	if err != nil {
		return nil
	}
	var paged, ok = item.ServicePage.(core.PagedServicePage)
	if !ok {
		return nil
	}

	typedPage = pageCommand(typedPage)
	var entries []string
	for _, page := range paged.ServicePages() {
		var name = pageCommand(page)
		if strings.HasPrefix(name, typedPage) && name != typedPage {
			entries = append(entries, command+" "+name)
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
	case slices.Contains(duplicateCommands, command):
		if err := inst.duplicate(); err != nil {
			inst.line.ShowError(err.Error())
			return
		}
		inst.line.ShowMessage("Opened a new window")
	default:
		var item, err = inst.findService(command)
		if err != nil {
			inst.line.ShowError(err.Error())
			return
		}
		var page, pageErr = inst.findPage(item, argument)
		if pageErr != nil {
			inst.line.ShowError(pageErr.Error())
			return
		}
		inst.show(item, page)
	}
}

// findPage is the page of the service the typed name is or begins the name of
// when only one page does, which is empty when no page is typed
func (inst *commands) findPage(item *ServiceItem, typed string) (string, error) {
	typed = pageCommand(typed)
	if typed == "" {
		return "", nil
	}
	var paged, ok = item.ServicePage.(core.PagedServicePage)
	if !ok {
		return "", fmt.Errorf("%s has no pages", serviceName(item))
	}

	var started []string
	for _, page := range paged.ServicePages() {
		if pageCommand(page) == typed {
			return page, nil
		}
		if strings.HasPrefix(pageCommand(page), typed) {
			started = append(started, page)
		}
	}

	switch len(started) {
	case 0:
		return "", fmt.Errorf("%s has no page %s", serviceName(item), typed)
	case 1:
		return started[0], nil
	}
	return "", fmt.Errorf("%s could be %s", typed, strings.Join(started, ", "))
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
