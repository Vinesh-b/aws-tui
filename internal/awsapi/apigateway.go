package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

type ApiGatewayApi interface {
	// ListRestApis returns every REST API of the account in the region.
	ListRestApis(ctx context.Context, force bool) ([]types.RestApi, error)
	// ListResources returns every resource (path) of a REST API with the
	// methods that are set up on it.
	ListResources(ctx context.Context, force bool, restApiId string) ([]types.Resource, error)
	// TestInvokeMethod runs a method of a resource the way a request to it would,
	// without a stage. For an integration that does something, that happens.
	// pathWithQueryString is the path of the request with the values of its path
	// parameters, and the query string after a "?".
	TestInvokeMethod(
		ctx context.Context,
		restApiId string,
		resourceId string,
		httpMethod string,
		pathWithQueryString string,
		body string,
		headers map[string]string,
	) (*apigateway.TestInvokeMethodOutput, error)
}

type apiGatewayApi struct {
	baseApi
}

func NewApiGatewayApi(
	logger *log.Logger,
) ApiGatewayApi {
	return &apiGatewayApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *apiGatewayApi) ListRestApis(ctx context.Context, force bool) ([]types.RestApi, error) {
	var result, err = collectTokenPages(func(position *string) ([]types.RestApi, *string, error) {
		var output, err = GetAwsApiClients().apigateway.GetRestApis(ctx,
			&apigateway.GetRestApisInput{Position: position},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Items, output.Position, nil
	})

	sortByKey(result, func(a types.RestApi) *string { return a.Name })

	return result, inst.logErr(err)
}

func (inst *apiGatewayApi) ListResources(ctx context.Context, force bool, restApiId string) ([]types.Resource, error) {
	if err := requireArg("restApiId", restApiId); err != nil {
		return nil, err
	}

	var result, err = collectTokenPages(func(position *string) ([]types.Resource, *string, error) {
		var output, err = GetAwsApiClients().apigateway.GetResources(ctx,
			&apigateway.GetResourcesInput{
				RestApiId: &restApiId,
				Embed:     []string{"methods"},
				Position:  position,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Items, output.Position, nil
	})

	sortByKey(result, func(r types.Resource) *string { return r.Path })

	return result, inst.logErr(err)
}

func (inst *apiGatewayApi) TestInvokeMethod(
	ctx context.Context,
	restApiId string,
	resourceId string,
	httpMethod string,
	pathWithQueryString string,
	body string,
	headers map[string]string,
) (*apigateway.TestInvokeMethodOutput, error) {
	if err := requireArg("restApiId", restApiId); err != nil {
		return nil, err
	}
	if err := requireArg("resourceId", resourceId); err != nil {
		return nil, err
	}
	if err := requireArg("httpMethod", httpMethod); err != nil {
		return nil, err
	}

	var input = &apigateway.TestInvokeMethodInput{
		RestApiId:  &restApiId,
		ResourceId: &resourceId,
		HttpMethod: &httpMethod,
		Headers:    headers,
	}
	if len(pathWithQueryString) > 0 {
		input.PathWithQueryString = &pathWithQueryString
	}
	if len(body) > 0 {
		input.Body = &body
	}

	var output, err = GetAwsApiClients().apigateway.TestInvokeMethod(ctx, input)

	return output, inst.logErr(err)
}
