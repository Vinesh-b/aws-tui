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
	// GetParametersByPath lists the parameters under the path. Secure values
	// are not decrypted, so the value of a SecureString is not its plain text.
	GetParametersByPath(
		ctx context.Context,
		path string, reset bool,
	) ([]types.Parameter, error)
	// GetParameterHistory lists the versions of the parameter. Secure values
	// are not decrypted, as they are not when the parameters are listed.
	GetParameterHistory(
		ctx context.Context,
		name string, reset bool,
	) ([]types.ParameterHistory, error)
	// GetParameterValue is the value of one version of the parameter, decrypted
	// if it is secure. A version of 0 is the latest one.
	GetParameterValue(
		ctx context.Context,
		name string, version int64,
	) (string, error)
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
	clients *AwsApiClients,
	logger *log.Logger,
) SystemsManagerApi {
	return &systemsManagerApi{
		baseApi: baseApi{clients: clients, logger: logger},
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

	var client = inst.clients.ssm
	if inst.getParamsByPathPaginator == nil || reset {
		inst.getParamsByPathPaginator = ssm.NewGetParametersByPathPaginator(
			client,
			&ssm.GetParametersByPathInput{
				Path:       aws.String(path),
				Recursive:  aws.Bool(true),
				MaxResults: aws.Int32(10),
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

	var client = inst.clients.ssm

	if inst.getParamHistoryPaginator == nil || reset {
		inst.getParamHistoryPaginator = ssm.NewGetParameterHistoryPaginator(
			client,
			&ssm.GetParameterHistoryInput{
				Name:       aws.String(name),
				MaxResults: aws.Int32(10),
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

func (inst *systemsManagerApi) GetParameterValue(
	ctx context.Context,
	name string, version int64,
) (string, error) {
	if err := requireArg("parameter name", name); err != nil {
		return "", err
	}

	// A version is asked for by adding it to the name
	var selector = name
	if version > 0 {
		selector = fmt.Sprintf("%s:%d", name, version)
	}

	var output, err = inst.clients.ssm.GetParameter(
		ctx,
		&ssm.GetParameterInput{
			Name:           aws.String(selector),
			WithDecryption: aws.Bool(true),
		},
	)
	if err != nil {
		return "", inst.logErr(err)
	}
	if output.Parameter == nil {
		return "", nil
	}

	return aws.ToString(output.Parameter.Value), nil
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

	var _, err = inst.clients.ssm.PutParameter(
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

	var _, err = inst.clients.ssm.PutParameter(
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

	var _, err = inst.clients.ssm.DeleteParameter(
		ctx,
		&ssm.DeleteParameterInput{Name: aws.String(name)},
	)
	if err != nil {
		return inst.logErr(err)
	}

	return nil
}
