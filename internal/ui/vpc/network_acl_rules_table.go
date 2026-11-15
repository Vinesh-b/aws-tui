package vpc

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// defaultNetworkAclRuleNumber is the rule every network ACL ends with, which
// denies what no other rule matched
const defaultNetworkAclRuleNumber = 32767

// NetworkAclRulesTable shows the inbound or outbound rules of the selected
// network ACL in the order they are evaluated
type NetworkAclRulesTable struct {
	*core.SelectableTable[types.NetworkAclEntry]
	egress bool
}

// NewNetworkAclInboundRulesTable shows the rules for traffic into the subnets
func NewNetworkAclInboundRulesTable(serviceCtx *core.ServiceContext[awsapi.Ec2Api]) *NetworkAclRulesTable {
	return newNetworkAclRulesTable("Inbound Rules", "Source", false, serviceCtx)
}

// NewNetworkAclOutboundRulesTable shows the rules for traffic out of the subnets
func NewNetworkAclOutboundRulesTable(serviceCtx *core.ServiceContext[awsapi.Ec2Api]) *NetworkAclRulesTable {
	return newNetworkAclRulesTable("Outbound Rules", "Destination", true, serviceCtx)
}

func newNetworkAclRulesTable(
	title string, addressHeading string, egress bool, serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *NetworkAclRulesTable {
	var table = &NetworkAclRulesTable{
		SelectableTable: core.NewSelectableTable[types.NetworkAclEntry](
			title,
			core.TableRow{
				"Rule Number",
				"Protocol",
				"Port Range",
				addressHeading,
				"Allow/Deny",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(3, 1).SetFilterColumns(1, 2, 3),
		egress: egress,
	}
	table.SetNetworkAcl(types.NetworkAcl{})
	return table
}

func networkAclRuleNumber(entry types.NetworkAclEntry) string {
	var number = aws.ToInt32(entry.RuleNumber)
	if number == defaultNetworkAclRuleNumber {
		return "*"
	}
	return strconv.Itoa(int(number))
}

func networkAclProtocol(protocol string) string {
	switch protocol {
	case "-1":
		return "All"
	case "1":
		return "ICMP"
	case "6":
		return "TCP"
	case "17":
		return "UDP"
	case "58":
		return "ICMPv6"
	}
	return protocol
}

// icmpValue is an ICMP type or code, where -1 is any
func icmpValue(value *int32) string {
	if value == nil || *value == -1 {
		return "All"
	}
	return strconv.Itoa(int(*value))
}

func networkAclPortRange(entry types.NetworkAclEntry) string {
	switch {
	case entry.IcmpTypeCode != nil:
		return fmt.Sprintf("Type %s, Code %s", icmpValue(entry.IcmpTypeCode.Type), icmpValue(entry.IcmpTypeCode.Code))
	case entry.PortRange == nil:
		return "All"
	}
	var from, to = aws.ToInt32(entry.PortRange.From), aws.ToInt32(entry.PortRange.To)
	if from == to {
		return strconv.Itoa(int(from))
	}
	return fmt.Sprintf("%d - %d", from, to)
}

func networkAclAddress(entry types.NetworkAclEntry) string {
	if entry.CidrBlock != nil {
		return aws.ToString(entry.CidrBlock)
	}
	return aws.ToString(entry.Ipv6CidrBlock)
}

func networkAclRuleAction(action types.RuleAction) string {
	switch action {
	case types.RuleActionAllow:
		return "Allow"
	case types.RuleActionDeny:
		return "Deny"
	}
	return string(action)
}

// SetNetworkAcl shows the rules of the network ACL in its direction
func (inst *NetworkAclRulesTable) SetNetworkAcl(acl types.NetworkAcl) {
	var entries []types.NetworkAclEntry
	for _, entry := range acl.Entries {
		if aws.ToBool(entry.Egress) == inst.egress {
			entries = append(entries, entry)
		}
	}
	slices.SortStableFunc(entries, func(a, b types.NetworkAclEntry) int {
		return cmp.Compare(aws.ToInt32(a.RuleNumber), aws.ToInt32(b.RuleNumber))
	})

	var tableData []core.TableRow
	for _, entry := range entries {
		tableData = append(tableData, core.TableRow{
			networkAclRuleNumber(entry),
			networkAclProtocol(aws.ToString(entry.Protocol)),
			networkAclPortRange(entry),
			networkAclAddress(entry),
			networkAclRuleAction(entry.RuleAction),
		})
	}

	inst.SetTitleExtra(aws.ToString(acl.NetworkAclId))
	inst.SetData(tableData, entries, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}
