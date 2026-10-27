package cloudwatch

import (
	"slices"

	"aws-tui/internal/utils"
)

// LogGroupMarks are the log groups marked on the Groups page to be queried by
// Insights. They are only changed and read on the UI goroutine.
type LogGroupMarks struct {
	groups    utils.StringSet
	listeners []func()
}

func NewLogGroupMarks() *LogGroupMarks {
	return &LogGroupMarks{groups: utils.StringSet{}}
}

// OnChange calls the handler after each change of the marks
func (inst *LogGroupMarks) OnChange(handler func()) {
	inst.listeners = append(inst.listeners, handler)
}

func (inst *LogGroupMarks) changed() {
	for _, listener := range inst.listeners {
		listener()
	}
}

func (inst *LogGroupMarks) Has(logGroup string) bool {
	var _, ok = inst.groups[logGroup]
	return ok
}

func (inst *LogGroupMarks) Add(logGroup string) {
	if len(logGroup) == 0 || inst.Has(logGroup) {
		return
	}
	inst.groups[logGroup] = struct{}{}
	inst.changed()
}

func (inst *LogGroupMarks) Remove(logGroup string) {
	if !inst.Has(logGroup) {
		return
	}
	delete(inst.groups, logGroup)
	inst.changed()
}

func (inst *LogGroupMarks) Toggle(logGroup string) {
	if inst.Has(logGroup) {
		inst.Remove(logGroup)
	} else {
		inst.Add(logGroup)
	}
}

func (inst *LogGroupMarks) Clear() {
	if len(inst.groups) == 0 {
		return
	}
	inst.groups = utils.StringSet{}
	inst.changed()
}

func (inst *LogGroupMarks) Len() int {
	return len(inst.groups)
}

// All are the marked log groups sorted by name
func (inst *LogGroupMarks) All() []string {
	var groups = make([]string, 0, len(inst.groups))
	for group := range inst.groups {
		groups = append(groups, group)
	}
	slices.Sort(groups)
	return groups
}
