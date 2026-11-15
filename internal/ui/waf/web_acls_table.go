package waf

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	"github.com/gdamore/tcell/v2"
)

// WebACLsTable lists the regional web ACLs. The list only has their names, so
// the rest is loaded for the one selected.
type WebACLsTable struct {
	*core.SelectableTable[types.WebACLSummary]
	data           []types.WebACLSummary
	selectedWebACL types.WebACLSummary
	serviceCtx     *core.ServiceContext[awsapi.WafApi]
}

func NewWebACLsTable(
	serviceContext *core.ServiceContext[awsapi.WafApi],
) *WebACLsTable {
	var table = &WebACLsTable{
		SelectableTable: core.NewSelectableTable[types.WebACLSummary](
			"Web ACLs",
			core.TableRow{
				"Name",
				"Description",
				"Id",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateWebACLsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *WebACLsTable) populateWebACLsTable() {
	var tableData []core.TableRow
	for _, acl := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(acl.Name),
			aws.ToString(acl.Description),
			aws.ToString(acl.Id),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedWebACL = types.WebACLSummary{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshWebACLs loads the web ACLs
func (inst *WebACLsTable) RefreshWebACLs() {
	inst.refreshWebACLs(func() {})
}

func (inst *WebACLsTable) refreshWebACLs(loaded func()) {
	var acls []types.WebACLSummary
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		acls, err = inst.serviceCtx.Api.ListWebACLs(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = acls
		inst.populateWebACLsTable()
		loaded()
	})
}

// Open selects the web ACL of the ARN. The web ACLs are loaded again first if
// it isn't one of those loaded.
func (inst *WebACLsTable) Open(webAclArn string) {
	if inst.selectWebACL(webAclArn) {
		return
	}

	inst.refreshWebACLs(func() {
		if !inst.selectWebACL(webAclArn) {
			inst.ErrorMessageCallback("Web ACL %s not found", webAclArn)
		}
	})
}

// selectWebACL shows all the web ACLs, without a search, to select the one
// with the ARN
func (inst *WebACLsTable) selectWebACL(webAclArn string) bool {
	var matches = func(acl types.WebACLSummary) bool { return aws.ToString(acl.ARN) == webAclArn }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateWebACLsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *WebACLsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedWebACL = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *WebACLsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshWebACLs()
			return nil
		}
		return capture(event)
	})
}

func (inst *WebACLsTable) GetSelectedWebACL() types.WebACLSummary {
	return inst.selectedWebACL
}
