package awsapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaApi interface {
	ListLambdas(ctx context.Context, force bool) ([]types.FunctionConfiguration, error)
	InvokeLambda(
		ctx context.Context,
		name string,
		payload map[string]any,
	) (*lambda.InvokeOutput, error)
	// GetPolicy gives the resource-based policy of the lambda as json, which is
	// empty for lambdas that have none.
	GetPolicy(ctx context.Context, lambdaArn string) (string, error)
	ListTags(ctx context.Context, lambdaArn string) (map[string]string, error)
	// ListEventSourceMappings gives the event sources that invoke the lambda,
	// such as queues and streams, sorted by their ARN.
	ListEventSourceMappings(ctx context.Context, lambdaArn string) ([]types.EventSourceMappingConfiguration, error)
	// ListVersions gives $LATEST and then the published versions of the lambda,
	// newest first.
	ListVersions(ctx context.Context, lambdaArn string) ([]types.FunctionConfiguration, error)
	// ListAliases gives the aliases of the lambda sorted by name
	ListAliases(ctx context.Context, lambdaArn string) ([]types.AliasConfiguration, error)
}

type lambdaApi struct {
	baseApi
	allLambdas []types.FunctionConfiguration
}

func NewLambdaApi(
	logger *log.Logger,
) LambdaApi {
	return &lambdaApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *lambdaApi) ListLambdas(ctx context.Context, force bool) ([]types.FunctionConfiguration, error) {
	var paginator = lambda.NewListFunctionsPaginator(
		GetAwsApiClients().lambda, &lambda.ListFunctionsInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.FunctionConfiguration, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Functions, nil
	})

	sortByKey(result, func(f types.FunctionConfiguration) *string { return f.FunctionName })

	return result, inst.logErr(err)
}

func (inst *lambdaApi) InvokeLambda(
	ctx context.Context,
	name string,
	payload map[string]any,
) (*lambda.InvokeOutput, error) {
	var output *lambda.InvokeOutput = nil

	if err := requireArg("lambda name", name); err != nil {
		return output, err
	}

	var err error = nil
	var jsonPayload []byte

	jsonPayload, err = json.Marshal(payload)
	if err != nil {
		return nil, inst.logErr(err)
	}

	var client = GetAwsApiClients().lambda
	output, err = client.Invoke(ctx,
		&lambda.InvokeInput{
			FunctionName:   aws.String(name),
			Payload:        jsonPayload,
			LogType:        types.LogTypeTail,
			InvocationType: types.InvocationTypeRequestResponse,
		},
	)

	if err != nil {
		inst.logger.Println(err)
	}

	return output, err
}

func (inst *lambdaApi) GetPolicy(ctx context.Context, lambdaArn string) (string, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return "", err
	}

	var client = GetAwsApiClients().lambda
	var output, err = client.GetPolicy(
		ctx,
		&lambda.GetPolicyInput{
			FunctionName: aws.String(lambdaArn),
		},
	)

	// A lambda without a policy is not found
	var notFound *types.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return "", nil
	}
	if err != nil {
		return "", inst.logErr(err)
	}

	return aws.ToString(output.Policy), nil
}

func (inst *lambdaApi) ListEventSourceMappings(
	ctx context.Context, lambdaArn string,
) ([]types.EventSourceMappingConfiguration, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return nil, err
	}

	var paginator = lambda.NewListEventSourceMappingsPaginator(
		GetAwsApiClients().lambda,
		&lambda.ListEventSourceMappingsInput{FunctionName: aws.String(lambdaArn)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.EventSourceMappingConfiguration, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.EventSourceMappings, nil
	})

	sortByKey(result, func(m types.EventSourceMappingConfiguration) *string { return m.EventSourceArn })

	return result, inst.logErr(err)
}

func (inst *lambdaApi) ListTags(ctx context.Context, lambdaArn string) (map[string]string, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().lambda
	var output, err = client.ListTags(
		ctx,
		&lambda.ListTagsInput{
			Resource: aws.String(lambdaArn),
		},
	)
	if err != nil {
		return nil, err
	}

	return output.Tags, err
}

// latestVersion is the version of a lambda that changes until it is published
const latestVersion = "$LATEST"

// versionNumber orders $LATEST before every published version
func versionNumber(version string) int {
	if version == latestVersion {
		return math.MaxInt
	}
	var number, err = strconv.Atoi(version)
	if err != nil {
		return -1
	}
	return number
}

func (inst *lambdaApi) ListVersions(ctx context.Context, lambdaArn string) ([]types.FunctionConfiguration, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return nil, err
	}

	var paginator = lambda.NewListVersionsByFunctionPaginator(
		GetAwsApiClients().lambda,
		&lambda.ListVersionsByFunctionInput{FunctionName: aws.String(lambdaArn)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.FunctionConfiguration, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Versions, nil
	})

	sort.SliceStable(result, func(i, j int) bool {
		return versionNumber(aws.ToString(result[i].Version)) > versionNumber(aws.ToString(result[j].Version))
	})

	return result, inst.logErr(err)
}

func (inst *lambdaApi) ListAliases(ctx context.Context, lambdaArn string) ([]types.AliasConfiguration, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return nil, err
	}

	var paginator = lambda.NewListAliasesPaginator(
		GetAwsApiClients().lambda,
		&lambda.ListAliasesInput{FunctionName: aws.String(lambdaArn)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.AliasConfiguration, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Aliases, nil
	})

	sortByKey(result, func(a types.AliasConfiguration) *string { return a.Name })

	return result, inst.logErr(err)
}
