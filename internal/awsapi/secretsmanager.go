package awsapi

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

type SecretsManagerApi interface {
	ListSecrets(
		ctx context.Context,
		reset bool,
	) ([]types.SecretListEntry, error)
	GetSecretValue(
		ctx context.Context,
		secretId string,
	) (*secretsmanager.GetSecretValueOutput, error)
}

type secretsManagerApi struct {
	baseApi
	listSecretsPaginator *secretsmanager.ListSecretsPaginator
}

func NewSecretsManagerApi(
	logger *log.Logger,
) SecretsManagerApi {
	return &secretsManagerApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *secretsManagerApi) ListSecrets(
	ctx context.Context,
	reset bool,
) ([]types.SecretListEntry, error) {
	var empty []types.SecretListEntry
	var client = GetAwsApiClients().secretsmanager

	if inst.listSecretsPaginator == nil || reset {
		inst.listSecretsPaginator = secretsmanager.NewListSecretsPaginator(
			client,
			&secretsmanager.ListSecretsInput{MaxResults: aws.Int32(50)},
		)
	}

	if !inst.listSecretsPaginator.HasMorePages() {
		return empty, fmt.Errorf("No more results")
	}

	var output, err = inst.listSecretsPaginator.NextPage(ctx)
	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.SecretList, nil
}

func (inst *secretsManagerApi) GetSecretValue(
	ctx context.Context,
	secretId string,
) (*secretsmanager.GetSecretValueOutput, error) {
	if err := requireArg("secret ID", secretId); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().secretsmanager.GetSecretValue(
		ctx,
		&secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}

	return output, nil
}
