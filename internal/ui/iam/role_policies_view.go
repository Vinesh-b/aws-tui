package iam

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/rivo/tview"
)

type RolePoliciesTable struct {
	*core.SelectableTable[awsapi.RolePolicy]
	roleName   string
	serviceCtx *core.ServiceContext[awsapi.IamApi]
}

func NewRolePoliciesTable(
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *RolePoliciesTable {
	var table = &RolePoliciesTable{
		SelectableTable: core.NewSelectableTable[awsapi.RolePolicy](
			"Policies",
			core.TableRow{"Name", "Type"},
			serviceCtx.AppContext,
		),
		serviceCtx: serviceCtx,
	}

	table.populatePoliciesTable(nil)

	return table
}

func policyType(policy awsapi.RolePolicy) string {
	if policy.IsInline() {
		return "Inline"
	}
	return "Managed"
}

func (inst *RolePoliciesTable) populatePoliciesTable(policies []awsapi.RolePolicy) {
	var tableData []core.TableRow
	for _, policy := range policies {
		tableData = append(tableData, core.TableRow{policy.Name, policyType(policy)})
	}

	inst.SetTitleExtra(inst.roleName)
	inst.SetData(tableData, policies, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

// RefreshPolicies lists the policies of the role, which is empty to clear them
func (inst *RolePoliciesTable) RefreshPolicies(roleName string) {
	inst.roleName = roleName
	if len(roleName) == 0 {
		inst.populatePoliciesTable(nil)
		return
	}

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var policies []awsapi.RolePolicy

	dataLoader.AsyncLoadData(func() {
		var err error
		policies, err = inst.serviceCtx.Api.ListRolePolicies(inst.serviceCtx.Ctx, roleName)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The selection moved on to another role while these loaded. The loader
		// puts back the title from before it loaded, so it is set again.
		if inst.roleName != roleName {
			inst.RefreshTitle(0)
			return
		}
		inst.populatePoliciesTable(policies)
	})
}

// GetSelectedPolicy is the selected policy, if there is one
func (inst *RolePoliciesTable) GetSelectedPolicy() (awsapi.RolePolicy, bool) {
	var row, _ = inst.GetTable().GetSelection()
	if row < 1 || row >= inst.GetTable().GetRowCount() {
		return awsapi.RolePolicy{}, false
	}
	var policy = inst.GetPrivateData(row, 0)
	return policy, len(policy.Name) > 0
}

// RolePoliciesView lists the policies of a role beside the document of the
// selected one, which is loaded when it is selected.
type RolePoliciesView struct {
	*core.ResizableView
	PoliciesTable *RolePoliciesTable
	DocumentView  *core.SearchableTextView
	shownPolicy   awsapi.RolePolicy
	serviceCtx    *core.ServiceContext[awsapi.IamApi]
}

func NewRolePoliciesView(
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *RolePoliciesView {
	var policiesTable = NewRolePoliciesTable(serviceCtx)
	var documentView = core.NewSearchableTextView("Policy Document", serviceCtx.AppContext)
	documentView.SetJsonHighlighting(true)

	const policiesTableSize = 4000
	const documentSize = 6000

	var view = &RolePoliciesView{
		ResizableView: core.NewResizableView(
			policiesTable, policiesTableSize,
			documentView, documentSize,
			tview.FlexColumn,
		),
		PoliciesTable: policiesTable,
		DocumentView:  documentView,
		serviceCtx:    serviceCtx,
	}

	policiesTable.SetSelectionChangedFunc(func(row, column int) {
		view.refreshDocument()
	})

	return view
}

func (inst *RolePoliciesView) documentTitle() string {
	if len(inst.shownPolicy.Name) == 0 {
		return "Policy Document"
	}
	return "Policy Document ❬" + inst.shownPolicy.Name + "❭"
}

func (inst *RolePoliciesView) RefreshPolicies(roleName string) {
	inst.shownPolicy = awsapi.RolePolicy{}
	inst.DocumentView.SetTitle(inst.documentTitle())
	inst.DocumentView.SetText("", false)
	inst.PoliciesTable.RefreshPolicies(roleName)
}

func (inst *RolePoliciesView) refreshDocument() {
	var policy, ok = inst.PoliciesTable.GetSelectedPolicy()
	if !ok {
		inst.shownPolicy = awsapi.RolePolicy{}
		inst.DocumentView.SetText("", false)
		return
	}

	var roleName = inst.PoliciesTable.roleName
	inst.shownPolicy = policy
	inst.DocumentView.SetTitle(inst.documentTitle())

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var document string

	dataLoader.AsyncLoadData(func() {
		var err error
		document, err = inst.serviceCtx.Api.GetRolePolicyDocument(inst.serviceCtx.Ctx, roleName, policy)
		if err != nil {
			inst.PoliciesTable.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.DocumentView.Box, func() {
		// Another policy was selected while this one loaded. The loader puts
		// back the title from before it loaded, so it is set again.
		inst.DocumentView.SetTitle(inst.documentTitle())
		if inst.shownPolicy != policy || inst.PoliciesTable.roleName != roleName {
			return
		}
		var formatted, _ = utils.TryFormatToJson(document)
		inst.DocumentView.SetText(formatted, false)
	})
}
