package apigateway

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

func TestMethodDetailsTable__ShowsTheMethod(t *testing.T) {
	var table = NewApiGatewayMethodDetailsTable(uitest.NewServiceContext[awsapi.ApiGatewayApi]())
	var method = types.Method{
		AuthorizationType: aws.String("AWS_IAM"),
		ApiKeyRequired:    aws.Bool(true),
		RequestParameters: map[string]bool{"method.request.querystring.page": false, "method.request.path.id": true},
		MethodIntegration: &types.Integration{
			Type:            types.IntegrationTypeAwsProxy,
			HttpMethod:      aws.String("POST"),
			Uri:             aws.String("arn:aws:apigateway:us-east-1:lambda:path/functions/orders/invocations"),
			TimeoutInMillis: 29000,
			IntegrationResponses: map[string]types.IntegrationResponse{
				"200": {SelectionPattern: aws.String("")},
			},
		},
		MethodResponses: map[string]types.MethodResponse{
			"200": {ResponseModels: map[string]string{"application/json": "Empty"}},
		},
	}

	table.SetMethod("/orders/{id}", "GET", &method)

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Method", "GET"},
		{"Path", "/orders/{id}"},
		{"Operation Name", ""},
		{"Authorization Type", "AWS_IAM"},
		{"Authorizer ID", ""},
		{"Authorization Scopes", ""},
		{"API Key Required", "true"},
		{"Request Validator ID", ""},
		{"Request Parameters"},
		{"method.request.path.id", "required"},
		{"method.request.querystring.page", "optional"},
		{"Integration"},
		{"Type", "AWS_PROXY"},
		{"HTTP Method", "POST"},
		{"URI", "arn:aws:apigateway:us-east-1:lambda:path/functions/orders/invocations"},
		{"Connection Type", ""},
		{"Connection ID", ""},
		{"Credentials", ""},
		{"Passthrough Behavior", ""},
		{"Content Handling", ""},
		{"Timeout (ms)", "29000"},
		{"Cache Namespace", ""},
		{"Integration Responses"},
		{"200", ""},
		{"Method Responses"},
		{"200", "application/json"},
	})
}

func TestMethodDetailsTable__ShowsNothingWithoutAMethod(t *testing.T) {
	var table = NewApiGatewayMethodDetailsTable(uitest.NewServiceContext[awsapi.ApiGatewayApi]())
	table.SetMethod("/", "GET", &types.Method{})

	table.SetMethod("", "", nil)

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}
