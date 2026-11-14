package core

// NoShortcut is the shortcut of a service that has none in the services list
const NoShortcut rune = '󰯉'

// Service is how a service is added to the app: its package gives one of these
// and the app lists it. Everything the app needs to show the service in the
// services list, go to it with a command and build its page is here.
type Service struct {
	// Id is the name of the service, which is also its command in lower case
	// without spaces
	Id ViewId
	// Icon is shown before the name in the services list
	Icon string
	// Description is shown under the name in the services list
	Description string
	// Shortcut is the key that opens the service from the services list, which
	// is NoShortcut for none
	Shortcut rune
	// Aliases are short commands for the service, beside its name
	Aliases []string
	// Build creates the page of the service with new api objects, which use the
	// api clients the app context has at the time. It is called again when the
	// clients or the theme change.
	Build func(appCtx *AppContext) ServicePage
}
