package iam

import (
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/gdamore/tcell/v2"
)

type RoleListTable struct {
	*core.SelectableTable[types.Role]
	data         []types.Role
	filtered     []types.Role
	selectedRole types.Role
	serviceCtx   *core.ServiceContext[awsapi.IamApi]
}

func NewRoleListTable(
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *RoleListTable {
	var view = &RoleListTable{
		SelectableTable: core.NewSelectableTable[types.Role](
			"Roles",
			core.TableRow{
				"Name",
				"Path",
			},
			serviceCtx.AppContext,
		),
		data:         nil,
		selectedRole: types.Role{},
		serviceCtx:   serviceCtx,
	}

	view.populateRoleTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			view.FilterByName(view.GetSearchText())
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func roleName(role types.Role) string {
	return aws.ToString(role.RoleName)
}

func (inst *RoleListTable) populateRoleTable(data []types.Role) {
	var tableData []core.TableRow
	for _, role := range data {
		tableData = append(tableData, core.TableRow{
			roleName(role),
			aws.ToString(role.Path),
		})
	}

	inst.SetData(tableData, data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *RoleListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(name, inst.data, roleName)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateRoleTable(inst.filtered)
	})
}

func (inst *RoleListTable) RefreshRoles(reset bool) {
	inst.refreshRoles(reset, func() {})
}

func (inst *RoleListTable) refreshRoles(reset bool, loaded func()) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var roles []types.Role

	dataLoader.AsyncLoadData(func() {
		var err error
		roles, err = inst.serviceCtx.Api.ListRoles(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = roles
		inst.populateRoleTable(inst.data)
		loaded()
	})
}

// OpenRole selects the role with the name. The roles are loaded again first if
// it isn't one of those loaded.
func (inst *RoleListTable) OpenRole(name string) {
	if inst.selectRole(name) {
		return
	}

	inst.refreshRoles(true, func() {
		if !inst.selectRole(name) {
			inst.ErrorMessageCallback("Role %s not found", name)
		}
	})
}

// selectRole shows all the roles, without a search, to select the one with
// the name.
func (inst *RoleListTable) selectRole(name string) bool {
	var idx = slices.IndexFunc(inst.data, func(role types.Role) bool { return roleName(role) == name })
	if idx < 0 {
		return false
	}

	inst.populateRoleTable(inst.data)
	inst.Select(idx+1, 0)
	return true
}

func (inst *RoleListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedRole = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *RoleListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedRole = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *RoleListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshRoles(true)
			return nil
		}
		return capture(event)
	})
}

// GetSelectedRole is the name of the selected role, empty while none is
func (inst *RoleListTable) GetSelectedRole() string {
	return roleName(inst.selectedRole)
}
