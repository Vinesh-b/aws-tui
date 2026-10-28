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
