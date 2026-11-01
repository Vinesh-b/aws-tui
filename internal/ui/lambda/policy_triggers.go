package lambda

import (
	"encoding/json"
	"slices"
	"strings"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

const apiGatewayPrincipal = "apigateway.amazonaws.com"

// apiGatewayTrigger is a statement of the resource-based policy that lets API
// Gateway invoke the lambda. API Gateway has no event source mapping, so the
// policy is the only place its triggers are found.
type apiGatewayTrigger struct {
	Sid string
	// SourceArn is the execute-api ARN of the methods allowed to invoke the
	// lambda, which is empty when any API can
	SourceArn     string
	SourceAccount string
	Statement     json.RawMessage
}

// apiGatewayMethod is the api, stage, method and path in an execute-api ARN,
// such as arn:aws:execute-api:us-east-1:1:abc123/prod/GET/orders/*. Any part
// may be a "*" wildcard.
type apiGatewayMethod struct {
	ApiId  string
	Stage  string
	Method string
	Path   string
}

func parseApiGatewayMethod(sourceArn string) apiGatewayMethod {
	var resource = sourceArn
	if parsed, err := arn.Parse(sourceArn); err == nil {
		resource = parsed.Resource
	}

	var parts = strings.SplitN(resource, "/", 4)
	var method = apiGatewayMethod{ApiId: parts[0]}
	if len(parts) > 1 {
		method.Stage = parts[1]
	}
	if len(parts) > 2 {
		method.Method = parts[2]
	}
	if len(parts) > 3 {
		method.Path = "/" + parts[3]
	}
	return method
}

// name is the api and what is called on it, such as "abc123 GET /orders"
func (inst apiGatewayTrigger) name() string {
	if inst.SourceArn == "" {
		return "Any API"
	}
	var method = parseApiGatewayMethod(inst.SourceArn)
	return strings.Join(slices.DeleteFunc(
		[]string{method.ApiId, method.Method, method.Path},
		func(part string) bool { return part == "" },
	), " ")
}

// stringOrList is a policy value that may be one string or a list of them
type stringOrList []string

func (inst *stringOrList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*inst = []string{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*inst = list
	return nil
}

type policyStatement struct {
	Sid       string
	Effect    string
	Principal json.RawMessage
	Condition map[string]map[string]stringOrList
}

func (inst policyStatement) allowsService(service string) bool {
	if inst.Effect != "Allow" {
		return false
	}
	var principal struct{ Service stringOrList }
	if err := json.Unmarshal(inst.Principal, &principal); err != nil {
		return false
	}
	return slices.Contains(principal.Service, service)
}

// conditionValue is the first value of a condition key in any operator, such
// as ArnLike or ArnEquals. Condition keys are not case-sensitive.
func (inst policyStatement) conditionValue(key string) string {
	var operators = make([]string, 0, len(inst.Condition))
	for operator := range inst.Condition {
		operators = append(operators, operator)
	}
	slices.Sort(operators)

	for _, operator := range operators {
		for name, values := range inst.Condition[operator] {
			if strings.EqualFold(name, key) && len(values) > 0 {
				return values[0]
			}
		}
	}
	return ""
}

// apiGatewayTriggers are the statements of the policy that let API Gateway
// invoke the lambda, sorted by the methods they allow. A policy that can't be
// read has none.
func apiGatewayTriggers(policy string) []apiGatewayTrigger {
	var document struct {
		Statement json.RawMessage
	}
	if err := json.Unmarshal([]byte(policy), &document); err != nil || len(document.Statement) == 0 {
		return nil
	}

	// The statement is a list, or one statement on its own
	var rawStatements []json.RawMessage
	if err := json.Unmarshal(document.Statement, &rawStatements); err != nil {
		rawStatements = []json.RawMessage{document.Statement}
	}

	var triggers []apiGatewayTrigger
	for _, raw := range rawStatements {
		var statement policyStatement
		if err := json.Unmarshal(raw, &statement); err != nil || !statement.allowsService(apiGatewayPrincipal) {
			continue
		}
		triggers = append(triggers, apiGatewayTrigger{
			Sid:           statement.Sid,
			SourceArn:     statement.conditionValue("AWS:SourceArn"),
			SourceAccount: statement.conditionValue("AWS:SourceAccount"),
			Statement:     raw,
		})
	}

	slices.SortStableFunc(triggers, func(a, b apiGatewayTrigger) int {
		return strings.Compare(a.SourceArn, b.SourceArn)
	})
	return triggers
}

func apiGatewayTriggerRows(trigger apiGatewayTrigger) []core.TableRow {
	var method = parseApiGatewayMethod(trigger.SourceArn)
	return []core.TableRow{
		{"Source", trigger.name()},
		{"Type", "API Gateway"},
		{"Source ARN", trigger.SourceArn},
		{"API ID", method.ApiId},
		{"Stage", method.Stage},
		{"Method", method.Method},
		{"Path", method.Path},
		{"Source Account", trigger.SourceAccount},
		{"Statement ID", trigger.Sid},
	}
}
