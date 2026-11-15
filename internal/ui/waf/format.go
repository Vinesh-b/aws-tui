package waf

import (
	"fmt"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// defaultRateWindowSec is the window a rate-based rule counts requests over
// when it doesn't set one
const defaultRateWindowSec = 300

// wafArnName is the name in a WAF ARN, or the ARN when it isn't one
func wafArnName(resourceArn string) string {
	var named, err = awsapi.ParseWafArn(resourceArn)
	if err != nil {
		return resourceArn
	}
	return named.Name
}

func defaultActionName(action *types.DefaultAction) string {
	switch {
	case action == nil:
		return ""
	case action.Allow != nil:
		return "Allow"
	case action.Block != nil:
		return "Block"
	}
	return ""
}

func ruleActionName(action types.RuleAction) string {
	switch {
	case action.Allow != nil:
		return "Allow"
	case action.Block != nil:
		return "Block"
	case action.Count != nil:
		return "Count"
	case action.Captcha != nil:
		return "CAPTCHA"
	case action.Challenge != nil:
		return "Challenge"
	case action.Monetize != nil:
		return "Monetize"
	}
	return ""
}

// ruleAction is what the rule does with the requests it matches. A rule that
// uses a rule group has the actions of the group's rules, unless it overrides
// them to count.
func ruleAction(rule types.Rule) string {
	switch {
	case rule.Action != nil:
		return ruleActionName(*rule.Action)
	case rule.OverrideAction == nil:
		return ""
	case rule.OverrideAction.Count != nil:
		return "Override to Count"
	case rule.OverrideAction.None != nil:
		return "Use Rule Actions"
	}
	return ""
}

func rateAggregateKey(key types.RateBasedStatementAggregateKeyType) string {
	switch key {
	case types.RateBasedStatementAggregateKeyTypeIp:
		return "IP"
	case types.RateBasedStatementAggregateKeyTypeForwardedIp:
		return "forwarded IP"
	case types.RateBasedStatementAggregateKeyTypeCustomKeys:
		return "custom keys"
	case types.RateBasedStatementAggregateKeyTypeConstant:
		return "all requests"
	}
	return string(key)
}

// joinStatements is the statements joined by the operator, with those that
// combine others in brackets
func joinStatements(statements []types.Statement, operator string) string {
	var parts []string
	for _, statement := range statements {
		var summary = statementSummary(&statement)
		if isCompound(&statement) {
			summary = "(" + summary + ")"
		}
		parts = append(parts, summary)
	}
	return strings.Join(parts, " "+operator+" ")
}

func isCompound(statement *types.Statement) bool {
	return statement.AndStatement != nil || statement.OrStatement != nil
}

// statementSummary is a line about what the statement of a rule matches. The
// details of how it inspects requests are in the web ACL's raw view.
func statementSummary(statement *types.Statement) string {
	switch {
	case statement == nil:
		return ""
	case statement.ManagedRuleGroupStatement != nil:
		var group = statement.ManagedRuleGroupStatement
		var summary = "Managed rule group " + aws.ToString(group.VendorName) + "/" + aws.ToString(group.Name)
		if version := aws.ToString(group.Version); len(version) > 0 {
			summary += " " + version
		}
		return summary
	case statement.RuleGroupReferenceStatement != nil:
		return "Rule group " + wafArnName(aws.ToString(statement.RuleGroupReferenceStatement.ARN))
	case statement.IPSetReferenceStatement != nil:
		return "IP set " + wafArnName(aws.ToString(statement.IPSetReferenceStatement.ARN))
	case statement.RegexPatternSetReferenceStatement != nil:
		return "Regex pattern set " + wafArnName(aws.ToString(statement.RegexPatternSetReferenceStatement.ARN))
	case statement.RateBasedStatement != nil:
		var rate = statement.RateBasedStatement
		var window = rate.EvaluationWindowSec
		if window == 0 {
			window = defaultRateWindowSec
		}
		var summary = fmt.Sprintf("Rate limit %d per %ds by %s",
			aws.ToInt64(rate.Limit), window, rateAggregateKey(rate.AggregateKeyType))
		if rate.ScopeDownStatement != nil {
			summary += " where " + statementSummary(rate.ScopeDownStatement)
		}
		return summary
	case statement.GeoMatchStatement != nil:
		var codes []string
		for _, code := range statement.GeoMatchStatement.CountryCodes {
			codes = append(codes, string(code))
		}
		return "Country is " + strings.Join(codes, ", ")
	case statement.AsnMatchStatement != nil:
		var asns []string
		for _, asn := range statement.AsnMatchStatement.AsnList {
			asns = append(asns, strconv.FormatInt(asn, 10))
		}
		return "ASN is " + strings.Join(asns, ", ")
	case statement.LabelMatchStatement != nil:
		return "Has label " + aws.ToString(statement.LabelMatchStatement.Key)
	case statement.ByteMatchStatement != nil:
		return "String match"
	case statement.RegexMatchStatement != nil:
		return "Regex match"
	case statement.SizeConstraintStatement != nil:
		return "Size constraint"
	case statement.SqliMatchStatement != nil:
		return "SQL injection"
	case statement.XssMatchStatement != nil:
		return "Cross-site scripting"
	case statement.AndStatement != nil:
		return joinStatements(statement.AndStatement.Statements, "AND")
	case statement.OrStatement != nil:
		return joinStatements(statement.OrStatement.Statements, "OR")
	case statement.NotStatement != nil:
		var inner = statement.NotStatement.Statement
		if inner != nil && isCompound(inner) {
			return "NOT (" + statementSummary(inner) + ")"
		}
		return "NOT " + statementSummary(inner)
	}
	return ""
}

// resourceTypeName is the name the console gives the type of resource
func resourceTypeName(resourceType types.ResourceType) string {
	switch resourceType {
	case types.ResourceTypeApplicationLoadBalancer:
		return "Application Load Balancer"
	case types.ResourceTypeApiGateway:
		return "API Gateway Stage"
	case types.ResourceTypeAppsync:
		return "AppSync API"
	case types.ResourceTypeCognitioUserPool:
		return "Cognito User Pool"
	case types.ResourceTypeAppRunnerService:
		return "App Runner Service"
	case types.ResourceTypeVerifiedAccessInstance:
		return "Verified Access Instance"
	}
	return string(resourceType)
}
