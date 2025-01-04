package awsapi

import (
	"aws-tui/internal/pkg/ui/core"
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
	config     aws.Config
	client     *lambda.Client
	allLambdas []types.FunctionConfiguration
}

func NewLambdaApi(
	config aws.Config,
	logger *log.Logger,
) *LambdaApi {
	return &LambdaApi{
		config:     config,
		logger:     logger,
		client:     lambda.NewFromConfig(config),
		allLambdas: []types.FunctionConfiguration{},
	}
}

func (inst *LambdaApi) ListLambdas(force bool) ([]types.FunctionConfiguration, error) {
	if len(inst.allLambdas) > 0 && !force {
		return inst.allLambdas, nil
	}

	inst.allLambdas = []types.FunctionConfiguration{}

	var paginator = lambda.NewListFunctionsPaginator(
		inst.client, &lambda.ListFunctionsInput{},
	)

	var apiError error = nil
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			apiError = err
			break
		}

		inst.allLambdas = append(inst.allLambdas, output.Functions...)
	}

	sort.Slice(inst.allLambdas, func(i, j int) bool {
		return aws.ToString(inst.allLambdas[i].FunctionName) < aws.ToString(inst.allLambdas[j].FunctionName)
	})

	return inst.allLambdas, apiError
}

func (inst *LambdaApi) FilterByName(name string) []types.FunctionConfiguration {
	return core.FuzzySearch(name, inst.allLambdas, func(v types.FunctionConfiguration) string {
		return aws.ToString(v.FunctionName)
	})
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

	output, err = inst.client.Invoke(context.TODO(),
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
