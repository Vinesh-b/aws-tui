package vpc

import (
	"errors"
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
	"aws-tui/internal/ui/vpc/vpcfake"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var networkAclHeadings = core.TableRow{
	"Name", "Network ACL Id", "VPC Id", "Default", "Inbound Rules", "Outbound Rules", "Subnets", "Owner Id",
}

var inboundRuleHeadings = core.TableRow{"Rule Number", "Protocol", "Port Range", "Source", "Allow/Deny"}
var outboundRuleHeadings = core.TableRow{"Rule Number", "Protocol", "Port Range", "Destination", "Allow/Deny"}
var associationHeadings = core.TableRow{"Subnet Id", "Association Id"}

func networkAclEntry(number int32, egress bool, protocol string, cidr string, action types.RuleAction) types.NetworkAclEntry {
	return types.NetworkAclEntry{
		RuleNumber: aws.Int32(number),
		Egress:     aws.Bool(egress),
		Protocol:   aws.String(protocol),
		CidrBlock:  aws.String(cidr),
		RuleAction: action,
	}
}

func withPorts(entry types.NetworkAclEntry, from int32, to int32) types.NetworkAclEntry {
	entry.PortRange = &types.PortRange{From: aws.Int32(from), To: aws.Int32(to)}
	return entry
}

func newNetworkAcl(id string, vpcId string, tags []types.Tag) types.NetworkAcl {
	return types.NetworkAcl{
		NetworkAclId: aws.String(id),
		VpcId:        aws.String(vpcId),
		IsDefault:    aws.Bool(false),
		OwnerId:      aws.String("123456789012"),
		Tags:         tags,
	}
}

// newWebNetworkAcl has rules each way, out of order as EC2 returns them, and
// two subnet associations
func newWebNetworkAcl() types.NetworkAcl {
	var acl = newNetworkAcl("acl-web", "vpc-1", nameTag("web-nacl"))
	acl.Entries = []types.NetworkAclEntry{
		networkAclEntry(defaultNetworkAclRuleNumber, false, "-1", "0.0.0.0/0", types.RuleActionDeny),
		withPorts(networkAclEntry(110, false, "6", "10.0.0.0/16", types.RuleActionAllow), 1024, 65535),
		withPorts(networkAclEntry(100, false, "6", "0.0.0.0/0", types.RuleActionAllow), 443, 443),
		{
			RuleNumber:    aws.Int32(120),
			Egress:        aws.Bool(false),
			Protocol:      aws.String("58"),
			Ipv6CidrBlock: aws.String("::/0"),
			RuleAction:    types.RuleActionAllow,
			IcmpTypeCode:  &types.IcmpTypeCode{Type: aws.Int32(128), Code: aws.Int32(-1)},
		},
		networkAclEntry(defaultNetworkAclRuleNumber, true, "-1", "0.0.0.0/0", types.RuleActionDeny),
		networkAclEntry(100, true, "-1", "0.0.0.0/0", types.RuleActionAllow),
	}
	acl.Associations = []types.NetworkAclAssociation{
		{NetworkAclAssociationId: aws.String("aclassoc-2"), SubnetId: aws.String("subnet-b")},
		{NetworkAclAssociationId: aws.String("aclassoc-1"), SubnetId: aws.String("subnet-a")},
	}
	return acl
}

func newNetworkAclsTest(t *testing.T, api *vpcfake.Api) *NetworkAclsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var view = NewNetworkAclsPageView(
		NewNetworkAclsTable(serviceCtx),
		NewNetworkAclDetailsTable(serviceCtx),
		NewNetworkAclInboundRulesTable(serviceCtx),
		NewNetworkAclOutboundRulesTable(serviceCtx),
		NewNetworkAclAssociationsTable(serviceCtx),
		serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func (inst *NetworkAclsPageView) app() *core.AppContext {
	return inst.NetworkAclsTable.serviceCtx.AppContext
}

func (inst *NetworkAclsPageView) reset(t *testing.T) {
	t.Helper()
	uitest.OnUI(t, inst.app().App, func() { uitest.PressTableRune(inst.NetworkAclsTable, core.APP_KEY_BINDINGS.Reset) })
}

func TestNetworkAclsTable__ResetLoadsTheNetworkAcls(t *testing.T) {
	var defaultAcl = newNetworkAcl("acl-default", "vpc-2", nil)
	defaultAcl.IsDefault = aws.Bool(true)
	var api = vpcfake.New(vpcfake.WithNetworkAcls([]types.NetworkAcl{newWebNetworkAcl(), defaultAcl}))
	var view = newNetworkAclsTest(t, api)

	view.reset(t)

	uitest.AwaitRows(t, view.app().App, view.NetworkAclsTable, []core.TableRow{
		networkAclHeadings,
		{"web-nacl", "acl-web", "vpc-1", "false", "4", "2", "2", "123456789012"},
		{"", "acl-default", "vpc-2", "true", "0", "0", "0", "123456789012"},
	})
}

func TestNetworkAclsPage__SelectedNetworkAclIsShownInEveryTab(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithNetworkAcls([]types.NetworkAcl{newWebNetworkAcl()}))
	var view = newNetworkAclsTest(t, api)

	view.reset(t)

	uitest.AwaitDetailsRows(t, view.app().App, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "web-nacl"},
		{"Network ACL ID", "acl-web"},
		{"VPC ID", "vpc-1"},
		{"Default", "false"},
		{"Owner ID", "123456789012"},
		{"Inbound Rules", "4"},
		{"Outbound Rules", "2"},
		{"Associated Subnets", "2"},
		{"Tags"},
		{"Name", "web-nacl"},
	})
	uitest.AwaitRows(t, view.app().App, view.InboundRulesTable, []core.TableRow{
		inboundRuleHeadings,
		{"100", "TCP", "443", "0.0.0.0/0", "Allow"},
		{"110", "TCP", "1024 - 65535", "10.0.0.0/16", "Allow"},
		{"120", "ICMPv6", "Type 128, Code All", "::/0", "Allow"},
		{"*", "All", "All", "0.0.0.0/0", "Deny"},
	})
	uitest.AwaitRows(t, view.app().App, view.OutboundRulesTable, []core.TableRow{
		outboundRuleHeadings,
		{"100", "All", "All", "0.0.0.0/0", "Allow"},
		{"*", "All", "All", "0.0.0.0/0", "Deny"},
	})
	uitest.AwaitRows(t, view.app().App, view.AssociationsTable, []core.TableRow{
		associationHeadings,
		{"subnet-a", "aclassoc-1"},
		{"subnet-b", "aclassoc-2"},
	})
	uitest.OnUI(t, view.app().App, func() {
		if got := view.InboundRulesTable.GetTitle(); got != "Inbound Rules ❬4❭ ❬acl-web❭" {
			t.Errorf("inbound rules title = %q", got)
		}
	})
}

func TestNetworkAclsPage__MovingTheSelectionShowsTheNextNetworkAcl(t *testing.T) {
	var other = newNetworkAcl("acl-other", "vpc-2", nameTag("other-nacl"))
	other.Entries = []types.NetworkAclEntry{
		withPorts(networkAclEntry(200, false, "17", "10.1.0.0/16", types.RuleActionDeny), 53, 53),
	}
	other.Associations = []types.NetworkAclAssociation{
		{NetworkAclAssociationId: aws.String("aclassoc-9"), SubnetId: aws.String("subnet-z")},
	}
	var api = vpcfake.New(vpcfake.WithNetworkAcls([]types.NetworkAcl{newWebNetworkAcl(), other}))
	var view = newNetworkAclsTest(t, api)
	view.reset(t)
	uitest.Eventually(t, view.app().App, func() bool { return view.AssociationsTable.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, view.app().App, func() { view.NetworkAclsTable.Select(2, 0) })

	uitest.AwaitRows(t, view.app().App, view.InboundRulesTable, []core.TableRow{
		inboundRuleHeadings,
		{"200", "UDP", "53", "10.1.0.0/16", "Deny"},
	})
	uitest.AwaitRows(t, view.app().App, view.OutboundRulesTable, []core.TableRow{outboundRuleHeadings})
	uitest.AwaitRows(t, view.app().App, view.AssociationsTable, []core.TableRow{
		associationHeadings,
		{"subnet-z", "aclassoc-9"},
	})
	uitest.OnUI(t, view.app().App, func() {
		if got := uitest.DetailsRowOf(view.DetailsTable.DetailsTable, "Name"); got < 0 ||
			view.DetailsTable.GetCellText(got, 1) != "other-nacl" {
			t.Errorf("details are not of acl-other")
		}
	})
}

func TestNetworkAclsPage__RefreshWithNoNetworkAclsClearsTheTabs(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithNetworkAcls([]types.NetworkAcl{newWebNetworkAcl()}))
	var view = newNetworkAclsTest(t, api)
	view.reset(t)
	uitest.Eventually(t, view.app().App, func() bool { return view.AssociationsTable.GetTable().GetRowCount() == 3 })

	api.SetNetworkAcls(nil)
	view.reset(t)

	uitest.AwaitRows(t, view.app().App, view.NetworkAclsTable, []core.TableRow{networkAclHeadings})
	uitest.Eventually(t, view.app().App, func() bool {
		return uitest.DetailsRowOf(view.DetailsTable.DetailsTable, "Network ACL ID") < 0
	}, "the details are cleared")
	uitest.AwaitRows(t, view.app().App, view.InboundRulesTable, []core.TableRow{inboundRuleHeadings})
	uitest.AwaitRows(t, view.app().App, view.OutboundRulesTable, []core.TableRow{outboundRuleHeadings})
	uitest.AwaitRows(t, view.app().App, view.AssociationsTable, []core.TableRow{associationHeadings})
}

func TestNetworkAclsTable__RefreshReportsTheApiError(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithErr(errors.New("not authorized to describe network acls")))
	var view = newNetworkAclsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.NetworkAclsTable.ErrorMessageCallback = callback

	view.reset(t)

	if got := uitest.AwaitError(t, received); got != "not authorized to describe network acls" {
		t.Errorf("message = %q", got)
	}
}

func TestHomeView__NetworkAclsPageLoadsTheNetworkAcls(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = vpcfake.New(vpcfake.WithNetworkAcls([]types.NetworkAcl{newWebNetworkAcl()}))
	var page = NewVpcHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	var paged, ok = page.(core.PagedServicePage)
	if !ok {
		t.Fatal("the VPC home view has no pages")
	}
	if pages := paged.ServicePages(); !slices.Contains(pages, "Network ACLs") {
		t.Fatalf("pages = %v, want a Network ACLs page", pages)
	}
	uitest.OnUI(t, appCtx.App, func() { paged.ShowServicePage("Network ACLs") })
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "web-nacl")
	if got := api.NetworkAclListings(); got != 1 {
		t.Errorf("listings = %d, want 1", got)
	}
}
