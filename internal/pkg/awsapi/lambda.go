package awsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaApi struct {
	logger     *log.Logger
	allLambdas []types.FunctionConfiguration
}

func NewLambdaApi(
	logger *log.Logger,
) *LambdaApi {
	return &LambdaApi{
		logger: logger,
	}
}

func (inst *LambdaApi) ListLambdas(force bool) ([]types.FunctionConfiguration, error) {
	var client = GetAwsApiClients().lambda
	var paginator = lambda.NewListFunctionsPaginator(
		client, &lambda.ListFunctionsInput{},
	)

	var result = []types.FunctionConfiguration{}
	var apiError error = nil
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			apiError = err
			break
		}

		result = append(result, output.Functions...)
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].FunctionName) < aws.ToString(result[j].FunctionName)
	})

	return result, apiError
}

func (inst *LambdaApi) InvokeLambda(
	name string,
	payload map[string]any,
) (*lambda.InvokeOutput, error) {
	var output *lambda.InvokeOutput = nil

	if len(name) == 0 {
		return output, fmt.Errorf("lambda name not set")
	}

	var err error = nil
	var jsonPayload []byte

	jsonPayload, err = json.Marshal(payload)
	if err != nil {
		inst.logger.Println(err)
		return nil, err
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
	if len(lambdaArn) == 0 {
		return "", fmt.Errorf("lambda ARN not set")
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
	if len(lambdaArn) == 0 {
		return nil, fmt.Errorf("lambda ARN not set")
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
