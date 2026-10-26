package iam

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

func policyType(policy awsapi.PrincipalPolicy) string {
	if policy.IsInline() {
		return "Inline"
	}
	return "Managed"
}

// PrincipalPoliciesView lists the policies of a role or user beside the
// document of the selected one, which is loaded when it is selected.
type PrincipalPoliciesView struct {
	*core.ResizableView
	PoliciesTable *IamChildTable[awsapi.PrincipalPolicy]
	DocumentView  *DocumentView
	loadDocument  func(ctx context.Context, principal string, policy awsapi.PrincipalPolicy) (string, error)
}

func NewPrincipalPoliciesView(
	serviceCtx *core.ServiceContext[awsapi.IamApi],
	listPolicies func(ctx context.Context, principal string) ([]awsapi.PrincipalPolicy, error),
	loadDocument func(ctx context.Context, principal string, policy awsapi.PrincipalPolicy) (string, error),
) *PrincipalPoliciesView {
	var policiesTable = NewIamChildTable(serviceCtx, "Policies", core.TableRow{"Name", "Type"},
		func(policy awsapi.PrincipalPolicy) core.TableRow {
			return core.TableRow{policy.Name, policyType(policy)}
		},
		listPolicies,
	)
	var documentView = NewDocumentView(serviceCtx, "Policy Document")

	const policiesTableSize = 4000
	const documentSize = 6000

	var view = &PrincipalPoliciesView{
		ResizableView: core.NewResizableView(
			policiesTable, policiesTableSize,
			documentView, documentSize,
			tview.FlexColumn,
		),
		PoliciesTable: policiesTable,
		DocumentView:  documentView,
		loadDocument:  loadDocument,
	}

	policiesTable.SetSelectionChangedFunc(func(row, column int) {
		view.showSelectedDocument()
	})

	return view
}

// SetErrorMessageCallback reports the failures of both the list and the
// documents
func (inst *PrincipalPoliciesView) SetErrorMessageCallback(callback func(text string, a ...any)) {
	inst.PoliciesTable.ErrorMessageCallback = callback
	inst.DocumentView.ErrorMessageCallback = callback
}

func (inst *PrincipalPoliciesView) Refresh(principal string) {
	inst.DocumentView.Clear()
	inst.PoliciesTable.Refresh(principal)
}

func (inst *PrincipalPoliciesView) showSelectedDocument() {
	var policy, ok = inst.PoliciesTable.GetSelectedItem()
	if !ok || len(policy.Name) == 0 {
		inst.DocumentView.Clear()
		return
	}

	var principal = inst.PoliciesTable.GetKey()
	var key = principal + "\x00" + policy.Name + "\x00" + policy.Arn
	inst.DocumentView.Show(key, policy.Name, func(ctx context.Context) (string, error) {
		return inst.loadDocument(ctx, principal, policy)
	})
}
