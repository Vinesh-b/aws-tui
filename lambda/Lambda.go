package lambda

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaApi struct {
	logger     *log.Logger
	config     aws.Config
	client     *lambda.Client
	allLambdas map[string]types.FunctionConfiguration
}

func NewLambdaApi(
	config aws.Config,
	logger *log.Logger,
) *LambdaApi {
	return &LambdaApi{
		config:     config,
		logger:     logger,
		client:     lambda.NewFromConfig(config),
		allLambdas: make(map[string]types.FunctionConfiguration),
	}
}

func (inst *LambdaApi) ListLambdas(force bool) map[string]types.FunctionConfiguration {
	if len(inst.allLambdas) > 0 && !force {
		return inst.allLambdas
	}

    inst.allLambdas = make(map[string]types.FunctionConfiguration)

	var paginator = lambda.NewListFunctionsPaginator(
		inst.client, &lambda.ListFunctionsInput{},
	)

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		for _, val := range output.Functions {
			inst.allLambdas[*val.FunctionName] = val
		}
	}
	return inst.allLambdas
}

func (inst *LambdaApi) FilterByName(name string) map[string]types.FunctionConfiguration {

	if len(inst.allLambdas) < 1 {
		inst.ListLambdas(true)
	}

	var foundLambdas = make(map[string]types.FunctionConfiguration)

	for _, info := range inst.allLambdas {
		found := strings.Contains(*info.FunctionName, name)
		if found {
			foundLambdas[*info.FunctionName] = info
		}
	}
	return foundLambdas
}
