package awsapi

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaApi struct {
	baseApi
	allLambdas []types.FunctionConfiguration
}

func NewLambdaApi(
	logger *log.Logger,
) *LambdaApi {
	return &LambdaApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *LambdaApi) ListLambdas(force bool) ([]types.FunctionConfiguration, error) {
	var paginator = lambda.NewListFunctionsPaginator(
		GetAwsApiClients().lambda, &lambda.ListFunctionsInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.FunctionConfiguration, error) {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		return output.Functions, nil
	})

	sortByKey(result, func(f types.FunctionConfiguration) *string { return f.FunctionName })

	return result, inst.logErr(err)
}

func (inst *LambdaApi) InvokeLambda(
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
	output, err = client.Invoke(context.TODO(),
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

func (inst *LambdaApi) GetPolicy(lambdaArn string) (string, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return "", err
	}

	var client = GetAwsApiClients().lambda
	var output, err = client.GetPolicy(
		context.TODO(),
		&lambda.GetPolicyInput{
			FunctionName: aws.String(lambdaArn),
		},
	)
	if err != nil {
		return "", err
	}

	return aws.ToString(output.Policy), err
}

func (inst *LambdaApi) ListTags(lambdaArn string) (map[string]string, error) {
	if err := requireArg("lambda ARN", lambdaArn); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().lambda
	var output, err = client.ListTags(
		context.TODO(),
		&lambda.ListTagsInput{
			Resource: aws.String(lambdaArn),
		},
	)
	if err != nil {
		return nil, err
	}

	return output.Tags, err
}
