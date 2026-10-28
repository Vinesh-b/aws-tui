package apigateway

import (
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

// ApiGatewayDetailsTable shows the selected REST API. Everything it shows, tags
// included, comes with the API in the list, so nothing is loaded.
type ApiGatewayDetailsTable struct {
	*core.DetailsTable
	data *types.RestApi
}

func NewApiGatewayDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.ApiGatewayApi],
) *ApiGatewayDetailsTable {
	var table = &ApiGatewayDetailsTable{
		DetailsTable: core.NewDetailsTable("REST API Details", serviceCtx.AppContext),
	}

	table.populateDetailsTable()

	return table
}

func apiRows(api types.RestApi) []core.TableRow {
	var vpcEndpoints string
	if api.EndpointConfiguration != nil {
		vpcEndpoints = strings.Join(api.EndpointConfiguration.VpcEndpointIds, ", ")
	}
	var compression string
	if api.MinimumCompressionSize != nil {
		compression = strconv.Itoa(int(*api.MinimumCompressionSize))
	}

	var rows = []core.TableRow{
		{"Name", aws.ToString(api.Name)},
		{"API ID", aws.ToString(api.Id)},
		{"Description", aws.ToString(api.Description)},
		{"Created", core.FormatTime(api.CreatedDate)},
		{"Version", aws.ToString(api.Version)},
		{"Status", string(api.ApiStatus)},
		{"Status Message", aws.ToString(api.ApiStatusMessage)},
		{"Root Resource ID", aws.ToString(api.RootResourceId)},
		{"Endpoint Type", endpointTypes(api)},
		{"VPC Endpoints", vpcEndpoints},
		{"Execute API Endpoint Disabled", strconv.FormatBool(api.DisableExecuteApiEndpoint)},
		{"API Key Source", string(api.ApiKeySource)},
		{"Minimum Compression Size", compression},
		{"Binary Media Types", strings.Join(api.BinaryMediaTypes, ", ")},
	}

	var tagRows = core.TagRows(tagPairs(api.Tags), func(p [2]string) (string, string) { return p[0], p[1] })
	return append(rows, core.DetailsSection("Tags", tagRows...)...)
}

func tagPairs(tags map[string]string) [][2]string {
	var pairs = make([][2]string, 0, len(tags))
	for k, v := range tags {
		pairs = append(pairs, [2]string{k, v})
	}
	return pairs
}

func (inst *ApiGatewayDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = apiRows(*inst.data)
		inst.SetTitleExtra(aws.ToString(inst.data.Name))
		inst.SetRawData(*inst.data)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetApi shows the API
func (inst *ApiGatewayDetailsTable) SetApi(api types.RestApi) {
	inst.data = &api
	inst.populateDetailsTable()
}
