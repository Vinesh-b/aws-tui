package apigateway

import (
	"slices"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

// ApiGatewayMethodDetailsTable shows a method of a resource of a REST API. The
// resources come with their methods in full, so nothing is loaded.
type ApiGatewayMethodDetailsTable struct {
	*core.DetailsTable
}

func NewApiGatewayMethodDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.ApiGatewayApi],
) *ApiGatewayMethodDetailsTable {
	var table = &ApiGatewayMethodDetailsTable{
		DetailsTable: core.NewDetailsTable("Method Details", serviceCtx.AppContext),
	}

	table.SetMethod("", "", nil)

	return table
}

// mapRows is a row for each entry of the map, in key order.
func mapRows[V any](entries map[string]V, value func(V) string) []core.TableRow {
	var rows = make([]core.TableRow, 0, len(entries))
	for key, entry := range entries {
		rows = append(rows, core.TableRow{key, value(entry)})
	}
	slices.SortFunc(rows, func(a, b core.TableRow) int { return strings.Compare(a[0], b[0]) })
	return rows
}

func methodRows(path string, name string, method types.Method) []core.TableRow {
	var rows = []core.TableRow{
		{"Method", name},
		{"Path", path},
		{"Operation Name", aws.ToString(method.OperationName)},
		{"Authorization Type", aws.ToString(method.AuthorizationType)},
		{"Authorizer ID", aws.ToString(method.AuthorizerId)},
		{"Authorization Scopes", strings.Join(method.AuthorizationScopes, ", ")},
		{"API Key Required", strconv.FormatBool(aws.ToBool(method.ApiKeyRequired))},
		{"Request Validator ID", aws.ToString(method.RequestValidatorId)},
	}

	rows = append(rows, core.DetailsSection("Request Parameters",
		mapRows(method.RequestParameters, func(required bool) string {
			if required {
				return "required"
			}
			return "optional"
		})...)...)
	rows = append(rows, core.DetailsSection("Request Models",
		mapRows(method.RequestModels, func(model string) string { return model })...)...)

	if integration := method.MethodIntegration; integration != nil {
		rows = append(rows, core.DetailsSection("Integration",
			core.TableRow{"Type", string(integration.Type)},
			core.TableRow{"HTTP Method", aws.ToString(integration.HttpMethod)},
			core.TableRow{"URI", aws.ToString(integration.Uri)},
			core.TableRow{"Connection Type", string(integration.ConnectionType)},
			core.TableRow{"Connection ID", aws.ToString(integration.ConnectionId)},
			core.TableRow{"Credentials", aws.ToString(integration.Credentials)},
			core.TableRow{"Passthrough Behavior", aws.ToString(integration.PassthroughBehavior)},
			core.TableRow{"Content Handling", string(integration.ContentHandling)},
			core.TableRow{"Timeout (ms)", strconv.Itoa(int(integration.TimeoutInMillis))},
			core.TableRow{"Cache Namespace", aws.ToString(integration.CacheNamespace)},
		)...)
		rows = append(rows, core.DetailsSection("Integration Request Parameters",
			mapRows(integration.RequestParameters, func(source string) string { return source })...)...)
		rows = append(rows, core.DetailsSection("Integration Responses",
			mapRows(integration.IntegrationResponses, func(r types.IntegrationResponse) string {
				return aws.ToString(r.SelectionPattern)
			})...)...)
	}

	return append(rows, core.DetailsSection("Method Responses",
		mapRows(method.MethodResponses, func(r types.MethodResponse) string {
			return strings.Join(sortedKeys(r.ResponseModels), ", ")
		})...)...)
}

func sortedKeys[V any](entries map[string]V) []string {
	var keys = make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// SetMethod shows the method of the resource at the path, or nothing if there
// is no method.
func (inst *ApiGatewayMethodDetailsTable) SetMethod(path string, name string, method *types.Method) {
	var tableData []core.TableRow
	if method != nil {
		tableData = methodRows(path, name, *method)
		inst.SetTitleExtra(name + " " + path)
		inst.SetRawData(*method)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}
