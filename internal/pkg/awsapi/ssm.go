package awsapi

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type SystemsManagerApi struct {
	logger                   *log.Logger
	getParamsByPathPaginator *ssm.GetParametersByPathPaginator
	getParamHistoryPaginator *ssm.GetParameterHistoryPaginator
}

func NewSystemsManagerApi(
	logger *log.Logger,
) *SystemsManagerApi {
	return &SystemsManagerApi{
		logger: logger,
	}
}

func (inst *SystemsManagerApi) GetParametersByPath(
	path string, reset bool,
) ([]types.Parameter, error) {
	var empty []types.Parameter

	if len(path) == 0 {
		return empty, fmt.Errorf("Parameter path not set")
	}

	var client = GetAwsApiClients().ssm
	if inst.getParamsByPathPaginator == nil || reset {
		inst.getParamsByPathPaginator = ssm.NewGetParametersByPathPaginator(
			client,
			&ssm.GetParametersByPathInput{
				Path:           aws.String(path),
				Recursive:      aws.Bool(true),
				WithDecryption: aws.Bool(true),
				MaxResults:     aws.Int32(10),
			},
		)
	}

	if !inst.getParamsByPathPaginator.HasMorePages() {
		return empty, fmt.Errorf("No more results")
	}

	var output, err = inst.getParamsByPathPaginator.NextPage(context.Background())
	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.Parameters, nil
}

func (inst *SystemsManagerApi) GetParameterHistory(
	name string, reset bool,
) ([]types.ParameterHistory, error) {
	var empty []types.ParameterHistory

	if len(name) == 0 {
		return empty, fmt.Errorf("Parameter name not set")
	}

	var client = GetAwsApiClients().ssm

	if inst.getParamHistoryPaginator == nil || reset {
		inst.getParamHistoryPaginator = ssm.NewGetParameterHistoryPaginator(
			client,
			&ssm.GetParameterHistoryInput{
				Name:           aws.String(name),
				WithDecryption: aws.Bool(true),
				MaxResults:     aws.Int32(10),
			},
		)
	}

	if !inst.getParamHistoryPaginator.HasMorePages() {
		return empty, fmt.Errorf("No more results")
	}

	var output, err = inst.getParamHistoryPaginator.NextPage(context.Background())
	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.Parameters, nil
}
