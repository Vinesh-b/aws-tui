package awsapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controlTypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
)

// accountId is the account of the credentials, which S3 Control needs with
// every request
func (inst *s3BucketsApi) accountId(ctx context.Context) (string, error) {
	var identity, err = inst.clients.Sts.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return aws.ToString(identity.Account), nil
}

func (inst *s3BucketsApi) ListAccessPoints(ctx context.Context) ([]s3controlTypes.AccessPoint, error) {
	var account, err = inst.accountId(ctx)
	if err != nil {
		return nil, inst.logErr(err)
	}

	var paginator = s3control.NewListAccessPointsPaginator(
		inst.clients.s3control, &s3control.ListAccessPointsInput{AccountId: aws.String(account)},
	)

	var result, listErr = collectPaginator(paginator.HasMorePages, func() ([]s3controlTypes.AccessPoint, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.AccessPointList, nil
	})

	sortByKey(result, func(a s3controlTypes.AccessPoint) *string { return a.Name })

	return result, inst.logErr(listErr)
}

// accessPointOfArn is the account and name of an access point ARN, whose
// resource is "accesspoint/<name>"
func accessPointOfArn(accessPointArn string) (string, string, error) {
	var parsed, err = arn.Parse(accessPointArn)
	if err != nil {
		return "", "", fmt.Errorf("invalid access point ARN %q: %w", accessPointArn, err)
	}
	var name, found = strings.CutPrefix(parsed.Resource, "accesspoint/")
	if !found || len(name) == 0 || len(parsed.AccountID) == 0 {
		return "", "", fmt.Errorf("invalid access point ARN %q", accessPointArn)
	}
	return parsed.AccountID, name, nil
}

func (inst *s3BucketsApi) GetAccessPoint(
	ctx context.Context, accessPointArn string,
) (*s3control.GetAccessPointOutput, error) {
	if err := requireArg("access point ARN", accessPointArn); err != nil {
		return nil, err
	}
	var account, name, err = accessPointOfArn(accessPointArn)
	if err != nil {
		return nil, err
	}

	var output, getErr = inst.clients.s3control.GetAccessPoint(ctx, &s3control.GetAccessPointInput{
		AccountId: aws.String(account),
		Name:      aws.String(name),
	})
	if getErr != nil {
		return nil, inst.logErr(getErr)
	}
	return output, nil
}

// noAccessPointPolicyCode is the error S3 gives for an access point without a
// policy
const noAccessPointPolicyCode = "NoSuchAccessPointPolicy"

func (inst *s3BucketsApi) GetAccessPointPolicy(ctx context.Context, accessPointArn string) (string, error) {
	if err := requireArg("access point ARN", accessPointArn); err != nil {
		return "", err
	}
	var account, name, err = accessPointOfArn(accessPointArn)
	if err != nil {
		return "", err
	}

	var output, getErr = inst.clients.s3control.GetAccessPointPolicy(ctx, &s3control.GetAccessPointPolicyInput{
		AccountId: aws.String(account),
		Name:      aws.String(name),
	})
	if getErr != nil {
		var apiErr smithy.APIError
		if errors.As(getErr, &apiErr) && apiErr.ErrorCode() == noAccessPointPolicyCode {
			return "", nil
		}
		return "", inst.logErr(getErr)
	}
	return aws.ToString(output.Policy), nil
}
