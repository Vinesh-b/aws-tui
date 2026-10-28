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
	PutSecretValue(
		ctx context.Context,
		secretId string, value string,
	) error
	DeleteSecret(
		ctx context.Context,
		secretId string,
	) error
}

// SecretRecoveryWindowDays is how long a deleted secret can still be restored
const SecretRecoveryWindowDays = 30

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

// PutSecretValue stores the value as the new current version of the secret
func (inst *secretsManagerApi) PutSecretValue(
	ctx context.Context,
	secretId string, value string,
) error {
	if err := requireArg("secret ID", secretId); err != nil {
		return err
	}
	if err := requireArg("secret value", value); err != nil {
		return err
	}

	var _, err = GetAwsApiClients().secretsmanager.PutSecretValue(
		ctx,
		&secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(secretId),
			SecretString: aws.String(value),
		},
	)
	if err != nil {
		return inst.logErr(err)
	}

	return nil
}

// DeleteSecret schedules the secret for deletion. It can be restored until the
// recovery window ends.
func (inst *secretsManagerApi) DeleteSecret(
	ctx context.Context,
	secretId string,
) error {
	if err := requireArg("secret ID", secretId); err != nil {
		return err
	}

	var _, err = GetAwsApiClients().secretsmanager.DeleteSecret(
		ctx,
		&secretsmanager.DeleteSecretInput{
			SecretId:             aws.String(secretId),
			RecoveryWindowInDays: aws.Int64(SecretRecoveryWindowDays),
		},
	)
	if err != nil {
		return inst.logErr(err)
	}

	return nil
}
