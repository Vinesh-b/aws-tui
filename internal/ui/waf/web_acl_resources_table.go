package waf

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

// WebACLResourcesTable shows the resources the selected web ACL protects.
// Finding them takes a call for each type of resource, so like the policy of
// an S3 access point, nothing is loaded until its tab is opened.
type WebACLResourcesTable struct {
	*core.SelectableTable[awsapi.WebACLResource]
	serviceCtx *core.ServiceContext[awsapi.WafApi]
	webAclArn  string
	loadedFor  string
	visible    bool
}

func NewWebACLResourcesTable(serviceCtx *core.ServiceContext[awsapi.WafApi]) *WebACLResourcesTable {
	var table = &WebACLResourcesTable{
		SelectableTable: core.NewSelectableTable[awsapi.WebACLResource](
			"Associated Resources",
			core.TableRow{
				"Resource Type",
				"ARN",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceCtx,
	}

	table.showResources(nil)
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.loadedFor = ""
			table.Load()
			return nil
		}
		return event
	})

	return table
}

func (inst *WebACLResourcesTable) showResources(resources []awsapi.WebACLResource) {
	var tableData []core.TableRow
	for _, resource := range resources {
		tableData = append(tableData, core.TableRow{
			resourceTypeName(resource.Type),
			resource.Arn,
		})
	}

	inst.SetTitleExtra(wafArnName(inst.webAclArn))
	inst.SetData(tableData, resources, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}

// SetWebACL selects the web ACL whose resources are shown, without loading
// them. Load loads them once the tab is open.
func (inst *WebACLResourcesTable) SetWebACL(webAclArn string) {
	if webAclArn == inst.webAclArn {
		return
	}
	inst.webAclArn = webAclArn
	inst.loadedFor = ""
	inst.showResources(nil)
}

func (inst *WebACLResourcesTable) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the resources of the web ACL selected, if the tab is open and
// they aren't loaded yet
func (inst *WebACLResourcesTable) Load() {
	var webAclArn = inst.webAclArn
	if !inst.visible || len(webAclArn) == 0 || inst.loadedFor == webAclArn {
		return
	}
	inst.loadedFor = webAclArn

	var resources []awsapi.WebACLResource
	var loadFailed = false
	// It takes a call for each type of resource, one after another
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 30)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		resources, err = inst.serviceCtx.Api.ListWebACLResources(ctx, webAclArn)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another web ACL was selected while this one loaded
		if inst.loadedFor != webAclArn {
			return
		}
		// What was found before an error is shown, and opening the tab again
		// tries again
		if loadFailed {
			inst.loadedFor = ""
		}
		inst.showResources(resources)
	})
}
