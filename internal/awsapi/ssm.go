package awsapi

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type SystemsManagerApi interface {
	GetParametersByPath(
		ctx context.Context,
		path string, reset bool,
	) ([]types.Parameter, error)
	GetParameterHistory(
		ctx context.Context,
		name string, reset bool,
	) ([]types.ParameterHistory, error)
	PutParameter(
		ctx context.Context,
		name string, value string, paramType types.ParameterType,
	) error
	CreateParameter(
		ctx context.Context,
		name string, value string, paramType types.ParameterType,
	) error
	DeleteParameter(
		ctx context.Context,
		name string,
	) error
}

type systemsManagerApi struct {
	baseApi
	getParamsByPathPaginator *ssm.GetParametersByPathPaginator
	getParamHistoryPaginator *ssm.GetParameterHistoryPaginator
}

func NewSystemsManagerApi(
	logger *log.Logger,
) SystemsManagerApi {
	return &systemsManagerApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *systemsManagerApi) GetParametersByPath(
	ctx context.Context,
	path string, reset bool,
) ([]types.Parameter, error) {
	var empty []types.Parameter

	if err := requireArg("parameter path", path); err != nil {
		return empty, err
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

	var output, err = inst.getParamsByPathPaginator.NextPage(ctx)
	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.Parameters, nil
}

func (inst *systemsManagerApi) GetParameterHistory(
	ctx context.Context,
	name string, reset bool,
) ([]types.ParameterHistory, error) {
	var empty []types.ParameterHistory

	if err := requireArg("parameter name", name); err != nil {
		return empty, err
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

	var output, err = inst.getParamHistoryPaginator.NextPage(ctx)
	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.Parameters, nil
}

// PutParameter stores the value as a new version of an existing parameter
func (inst *systemsManagerApi) PutParameter(
	ctx context.Context,
	name string, value string, paramType types.ParameterType,
) error {
	if err := requireArg("parameter name", name); err != nil {
		return err
	}
	if err := requireArg("parameter value", value); err != nil {
		return err
	}

	var _, err = GetAwsApiClients().ssm.PutParameter(
		ctx,
		&ssm.PutParameterInput{
			Name:      aws.String(name),
			Value:     aws.String(value),
			Type:      paramType,
			Overwrite: aws.Bool(true),
		},
	)
	if err != nil {
		return inst.logErr(err)
	}

	return nil
}

// CreateParameter adds a parameter that does not exist yet, and fails rather
// than replace one that does
func (inst *systemsManagerApi) CreateParameter(
	ctx context.Context,
	name string, value string, paramType types.ParameterType,
) error {
	if err := requireArg("parameter name", name); err != nil {
		return err
	}
	if err := requireArg("parameter value", value); err != nil {
		return err
	}

	var _, err = GetAwsApiClients().ssm.PutParameter(
		ctx,
		&ssm.PutParameterInput{
			Name:      aws.String(name),
			Value:     aws.String(value),
			Type:      paramType,
			Overwrite: aws.Bool(false),
		},
	)
	if err != nil {
		return inst.logErr(err)
	}

	return nil
}

// DeleteParameter deletes the parameter and its history for good
func (inst *systemsManagerApi) DeleteParameter(
	ctx context.Context,
	name string,
) error {
	if err := requireArg("parameter name", name); err != nil {
		return err
	}

	var _, err = GetAwsApiClients().ssm.DeleteParameter(
		ctx,
		&ssm.DeleteParameterInput{Name: aws.String(name)},
	)
	if err != nil {
		return inst.logErr(err)
	}

	return nil
}
