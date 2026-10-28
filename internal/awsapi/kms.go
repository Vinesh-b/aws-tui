package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// kmsDefaultPolicyName is the only name a key policy can have
const kmsDefaultPolicyName = "default"

type KmsApi interface {
	// ListKeys returns every key of the account in the region. The result is
	// kept until force asks for it again.
	ListKeys(ctx context.Context, force bool) ([]types.KeyListEntry, error)
	// ListAliases returns every alias, which point to the keys by their ID.
	ListAliases(ctx context.Context, force bool) ([]types.AliasListEntry, error)
	DescribeKey(ctx context.Context, keyId string) (*types.KeyMetadata, error)
	ListTags(ctx context.Context, force bool, keyId string) ([]types.Tag, error)
	// GetKeyPolicy returns the policy document of the key as json text.
	GetKeyPolicy(ctx context.Context, keyId string) (string, error)
}

type kmsApi struct {
	baseApi
	allKeys    []types.KeyListEntry
	allAliases []types.AliasListEntry
}

func NewKmsApi(
	logger *log.Logger,
) KmsApi {
	return &kmsApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *kmsApi) ListKeys(ctx context.Context, force bool) ([]types.KeyListEntry, error) {
	if len(inst.allKeys) > 0 && !force {
		return inst.allKeys, nil
	}

	var paginator = kms.NewListKeysPaginator(GetAwsApiClients().kms, &kms.ListKeysInput{})
	var keys, err = collectPaginator(paginator.HasMorePages, func() ([]types.KeyListEntry, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Keys, nil
	})
	sortByKey(keys, func(k types.KeyListEntry) *string { return k.KeyId })
	inst.allKeys = keys

	return keys, inst.logErr(err)
}

func (inst *kmsApi) ListAliases(ctx context.Context, force bool) ([]types.AliasListEntry, error) {
	if len(inst.allAliases) > 0 && !force {
		return inst.allAliases, nil
	}

	var paginator = kms.NewListAliasesPaginator(GetAwsApiClients().kms, &kms.ListAliasesInput{})
	var aliases, err = collectPaginator(paginator.HasMorePages, func() ([]types.AliasListEntry, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Aliases, nil
	})
	sortByKey(aliases, func(a types.AliasListEntry) *string { return a.AliasName })
	inst.allAliases = aliases

	return aliases, inst.logErr(err)
}

func (inst *kmsApi) DescribeKey(ctx context.Context, keyId string) (*types.KeyMetadata, error) {
	if err := requireArg("key ID", keyId); err != nil {
		return nil, err
	}

	var output, err = GetAwsApiClients().kms.DescribeKey(ctx,
		&kms.DescribeKeyInput{KeyId: aws.String(keyId)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.KeyMetadata, nil
}

func (inst *kmsApi) ListTags(ctx context.Context, force bool, keyId string) ([]types.Tag, error) {
	if err := requireArg("key ID", keyId); err != nil {
		return nil, err
	}

	var paginator = kms.NewListResourceTagsPaginator(GetAwsApiClients().kms,
		&kms.ListResourceTagsInput{KeyId: aws.String(keyId)},
	)
	var tags, err = collectPaginator(paginator.HasMorePages, func() ([]types.Tag, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Tags, nil
	})

	return tags, inst.logErr(err)
}

func (inst *kmsApi) GetKeyPolicy(ctx context.Context, keyId string) (string, error) {
	if err := requireArg("key ID", keyId); err != nil {
		return "", err
	}

	var output, err = GetAwsApiClients().kms.GetKeyPolicy(ctx,
		&kms.GetKeyPolicyInput{
			KeyId:      aws.String(keyId),
			PolicyName: aws.String(kmsDefaultPolicyName),
		},
	)
	if err != nil {
		return "", inst.logErr(err)
	}
	return aws.ToString(output.Policy), nil
}
