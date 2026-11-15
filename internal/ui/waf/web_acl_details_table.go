package waf

import (
	"context"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	"github.com/gdamore/tcell/v2"
)

// WebACLDetailsTable shows a web ACL as the list has it, then the web ACL
// loaded for it. Its raw view has all of the loaded web ACL, with the whole of
// each rule.
type WebACLDetailsTable struct {
	*core.DetailsTable
	summary    types.WebACLSummary
	webAcl     *types.WebACL
	loaded     func(webAcl *types.WebACL)
	serviceCtx *core.ServiceContext[awsapi.WafApi]
}

func NewWebACLDetailsTable(
	serviceContext *core.ServiceContext[awsapi.WafApi],
) *WebACLDetailsTable {
	var table = &WebACLDetailsTable{
		DetailsTable: core.NewDetailsTable("Web ACL Details", serviceContext.AppContext),
		loaded:       func(*types.WebACL) {},
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshDetails()
			return nil
		}
		return event
	})

	return table
}

func (inst *WebACLDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.ARN != nil {
		var summary = inst.summary
		tableData = []core.TableRow{
			{"Name", aws.ToString(summary.Name)},
			{"Description", aws.ToString(summary.Description)},
			{"Id", aws.ToString(summary.Id)},
			{"ARN", aws.ToString(summary.ARN)},
		}
		rawData = summary

		if acl := inst.webAcl; acl != nil {
			tableData = append(tableData,
				core.TableRow{"Default Action", defaultActionName(acl.DefaultAction)},
				core.TableRow{"Rules", strconv.Itoa(len(acl.Rules))},
				core.TableRow{"Capacity (WCUs)", strconv.FormatInt(acl.Capacity, 10)},
				core.TableRow{"Label Namespace", aws.ToString(acl.LabelNamespace)},
				core.TableRow{"Token Domains", strings.Join(acl.TokenDomains, ", ")},
				core.TableRow{"Firewall Manager", strconv.FormatBool(acl.ManagedByFirewallManager)},
			)

			if visibility := acl.VisibilityConfig; visibility != nil {
				tableData = append(tableData, core.DetailsSection("Visibility",
					core.TableRow{"Metric Name", aws.ToString(visibility.MetricName)},
					core.TableRow{"CloudWatch Metrics", strconv.FormatBool(visibility.CloudWatchMetricsEnabled)},
					core.TableRow{"Sampled Requests", strconv.FormatBool(visibility.SampledRequestsEnabled)},
				)...)
			}
			rawData = acl
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.Name))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetWebACLLoadedFunc runs the handler with each web ACL that is loaded for
// the one shown, or with nil when the one shown changes
func (inst *WebACLDetailsTable) SetWebACLLoadedFunc(handler func(webAcl *types.WebACL)) {
	inst.loaded = handler
}

// SetWebACL shows the web ACL as the list has it, until it is loaded
func (inst *WebACLDetailsTable) SetWebACL(summary types.WebACLSummary) {
	inst.summary = summary
	inst.webAcl = nil
	inst.populateDetailsTable()
	inst.loaded(nil)
}

// RefreshDetails loads the web ACL shown
func (inst *WebACLDetailsTable) RefreshDetails() {
	var webAclArn = aws.ToString(inst.summary.ARN)
	if len(webAclArn) == 0 {
		return
	}

	var webAcl *types.WebACL
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		webAcl, err = inst.serviceCtx.Api.GetWebACL(ctx, webAclArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another web ACL was selected while this one loaded
		if aws.ToString(inst.summary.ARN) != webAclArn || webAcl == nil {
			return
		}
		inst.webAcl = webAcl
		inst.populateDetailsTable()
		inst.loaded(webAcl)
	})
}
