package xray

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

// noGroup is the option of the group picker while the filter is not that of a
// group, as it was entered
const noGroup = "-"

// defaultGroupName is the group of all the traces, which has no filter
const defaultGroupName = "Default"

// traceGroupViewWidth fits the label and most names of groups
const traceGroupViewWidth = 26

// TraceGroupView picks an X-Ray group, which is a filter expression kept with
// a name. Picking one puts its filter in and loads its traces. Entering a
// filter moves the picker to no group.
type TraceGroupView struct {
	*core.DropDown
	// ErrorMessageCallback reports the groups failing to load
	ErrorMessageCallback func(text string, a ...any)

	filter     *TraceFilterView
	serviceCtx *core.ServiceContext[awsapi.XRayApi]
	groups     []types.Group
	// options are the names of the picker, the first of which is no group
	options []string
	// settingGroup is set while the picker or the filter are changed by the
	// view, which is not a group being picked or a filter being entered
	settingGroup bool
}

func NewTraceGroupView(
	filter *TraceFilterView, serviceCtx *core.ServiceContext[awsapi.XRayApi],
) *TraceGroupView {
	var view = &TraceGroupView{
		DropDown:             core.NewDropDown(serviceCtx.Theme),
		ErrorMessageCallback: func(text string, a ...any) {},
		filter:               filter,
		serviceCtx:           serviceCtx,
	}
	view.SetLabel("Group ")
	view.showGroups("")
	filter.SetChangedFunc(func(string) {
		if !view.settingGroup {
			view.showGroup(0)
		}
	})
	return view
}

// sortedGroups are the groups by name, with the default group first
func sortedGroups(groups []types.Group) []types.Group {
	var sorted = slices.Clone(groups)
	slices.SortStableFunc(sorted, func(a, b types.Group) int {
		var aDefault, bDefault = aws.ToString(a.GroupName) == defaultGroupName, aws.ToString(b.GroupName) == defaultGroupName
		if aDefault != bDefault {
			if aDefault {
				return -1
			}
			return 1
		}
		return cmp.Compare(strings.ToLower(aws.ToString(a.GroupName)), strings.ToLower(aws.ToString(b.GroupName)))
	})
	return sorted
}

// showGroups puts the groups in the picker, with the one of the name picked,
// or no group if it is not one of them
func (inst *TraceGroupView) showGroups(picked string) {
	var options = []string{noGroup}
	var pickedIdx = 0
	for idx, group := range inst.groups {
		var name = aws.ToString(group.GroupName)
		options = append(options, name)
		if name == picked {
			pickedIdx = idx + 1
		}
	}

	inst.options = options
	inst.settingGroup = true
	defer func() { inst.settingGroup = false }()
	// The options are set with what picking one does, as setting them
	// replaces it
	inst.SetOptions(options, inst.selected)
	inst.SetCurrentOption(pickedIdx)
}

// selected picks the group of the option, unless it is no group or the view
// is showing it
func (inst *TraceGroupView) selected(text string, index int) {
	if !inst.settingGroup && index > 0 {
		inst.pick(inst.groups[index-1])
	}
}

// Options are the names of the picker, the first of which is no group
func (inst *TraceGroupView) Options() []string {
	return slices.Clone(inst.options)
}

// showGroup shows the option picked, without picking it
func (inst *TraceGroupView) showGroup(index int) {
	inst.settingGroup = true
	defer func() { inst.settingGroup = false }()
	inst.SetCurrentOption(index)
}

// pick puts the filter of the group in and loads its traces. The default
// group has no filter, so it loads them all.
func (inst *TraceGroupView) pick(group types.Group) {
	inst.settingGroup = true
	inst.filter.SetText(aws.ToString(group.FilterExpression))
	inst.settingGroup = false
	inst.filter.ApplyFilter()
}

// RefreshGroups loads the groups, keeping the one picked if it still is one
func (inst *TraceGroupView) RefreshGroups() {
	var groups []types.Group
	var loaded = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		groups, err = inst.serviceCtx.Api.GetGroups(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}
		loaded = true
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if !loaded {
			return
		}
		var _, picked = inst.GetCurrentOption()
		inst.groups = sortedGroups(groups)
		inst.showGroups(picked)
	})
}
