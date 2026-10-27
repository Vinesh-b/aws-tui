package core

// Location is a page of a service that was shown
type Location struct {
	// Service is the name of the service in the app, as its menu shows it
	Service string
	// Page is the page shown in the service, which is empty for a service
	// without pages of its own
	Page string
}

// NavigationHistory keeps the locations that were shown, to go back and
// forward through like the history of a browser. It is only used on the UI
// goroutine.
type NavigationHistory struct {
	back       []Location
	forward    []Location
	current    Location
	hasCurrent bool
	limit      int
	// recording is false while the locations are shown by going back or
	// forward, which are not new visits
	recording bool
}

// NewNavigationHistory keeps up to limit locations to go back to
func NewNavigationHistory(limit int) *NavigationHistory {
	return &NavigationHistory{limit: max(limit, 1), recording: true}
}

// Visit makes the location the current one. The location that was current is
// kept to go back to, and those that were gone back from are dropped. Visiting
// the current location again changes nothing.
func (inst *NavigationHistory) Visit(location Location) {
	if !inst.recording || (inst.hasCurrent && inst.current == location) {
		return
	}

	if inst.hasCurrent {
		inst.back = append(inst.back, inst.current)
		if len(inst.back) > inst.limit {
			inst.back = inst.back[len(inst.back)-inst.limit:]
		}
	}
	inst.forward = nil
	inst.current = location
	inst.hasCurrent = true
}

// WithoutRecording runs the function without the locations it shows being
// visits, such as while showing a location that was gone back to.
func (inst *NavigationHistory) WithoutRecording(fn func()) {
	var recording = inst.recording
	inst.recording = false
	defer func() { inst.recording = recording }()
	fn()
}

// Back makes the location before the current one current and returns it. It is
// false when there is none.
func (inst *NavigationHistory) Back() (Location, bool) {
	if len(inst.back) == 0 {
		return Location{}, false
	}

	var last = len(inst.back) - 1
	inst.forward = append(inst.forward, inst.current)
	inst.current = inst.back[last]
	inst.back = inst.back[:last]
	return inst.current, true
}

// Forward makes the location that was gone back from current again and returns
// it. It is false when there is none.
func (inst *NavigationHistory) Forward() (Location, bool) {
	if len(inst.forward) == 0 {
		return Location{}, false
	}

	var last = len(inst.forward) - 1
	inst.back = append(inst.back, inst.current)
	inst.current = inst.forward[last]
	inst.forward = inst.forward[:last]
	return inst.current, true
}

func (inst *NavigationHistory) Current() (Location, bool) {
	return inst.current, inst.hasCurrent
}

func (inst *NavigationHistory) CanGoBack() bool {
	return len(inst.back) > 0
}

func (inst *NavigationHistory) CanGoForward() bool {
	return len(inst.forward) > 0
}

// Restart drops every location and starts again from the one given, such as
// when the pages the locations were on are built again with new data.
func (inst *NavigationHistory) Restart(location Location) {
	inst.back = nil
	inst.forward = nil
	inst.current = location
	inst.hasCurrent = true
}

// Clear drops every location, the current one too
func (inst *NavigationHistory) Clear() {
	inst.back = nil
	inst.forward = nil
	inst.current = Location{}
	inst.hasCurrent = false
}
