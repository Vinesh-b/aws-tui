package awsapi

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3BucketsApi struct {
	baseApi
	allbuckets       []types.Bucket
	objectsPaginator *s3.ListObjectsV2Paginator
}

func NewS3BucketsApi(
	logger *log.Logger,
) *S3BucketsApi {
	return &S3BucketsApi{
		baseApi:    baseApi{logger: logger},
		allbuckets: []types.Bucket{},
	}
}

func (inst *S3BucketsApi) ListBuckets(force bool) ([]types.Bucket, error) {
	if len(inst.allbuckets) > 0 && !force {
		return inst.allbuckets, nil
	}

	var paginator = s3.NewListBucketsPaginator(
		GetAwsApiClients().s3, &s3.ListBucketsInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.Bucket, error) {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		return output.Buckets, nil
	})

	sortByKey(result, func(b types.Bucket) *string { return b.Name })

	inst.allbuckets = result

	return result, inst.logErr(err)
}

func (inst *S3BucketsApi) ListObjects(
	bucketName string,
	prefix string,
	force bool,
) ([]types.Object, []types.CommonPrefix, error) {
	if err := requireArg("bucket name", bucketName); err != nil {
		return nil, nil, err
	}

	var objPrefix = &prefix
	if len(prefix) == 0 {
		objPrefix = nil
	}

	var client = GetAwsApiClients().s3
	if force || inst.objectsPaginator == nil {
		inst.objectsPaginator = s3.NewListObjectsV2Paginator(
			client, &s3.ListObjectsV2Input{
				Bucket:    aws.String(bucketName),
				MaxKeys:   aws.Int32(200),
				Delimiter: aws.String("/"),
				Prefix:    objPrefix,
			})
	}

	if !inst.objectsPaginator.HasMorePages() {
		return nil, nil, fmt.Errorf("No more pages found")
	}

	var output, err = inst.objectsPaginator.NextPage(context.TODO())
	if err != nil {
		return nil, nil, inst.logErr(err)
	}

	return output.Contents, output.CommonPrefixes, nil
}

func (inst *S3BucketsApi) DownloadFile(bucketName string, objectKey string, fileName string) error {
	if err := requireArg("bucket name", bucketName); err != nil {
		return err
	}

	if err := requireArg("object key", objectKey); err != nil {
		return err
	}

	if err := requireArg("file name", fileName); err != nil {
		return err
	}

	var client = GetAwsApiClients().s3

	result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		inst.logger.Printf("Failed to get object: %v:%v, Error: %v\n", bucketName, objectKey, err)
		return err
	}

	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		inst.logger.Printf("Failed to create file: %v, Error: %v\n", fileName, err)
		return err
	}

	defer file.Close()

	if _, err = io.Copy(file, result.Body); err != nil {
		inst.logger.Printf("Failed to write object body: %v, Error: %v\n", objectKey, err)
		return err
	}

	return nil
}

func (inst *S3BucketsApi) GetBucketPolicy(bucketArn string, force bool) (string, error) {
	var client = GetAwsApiClients().s3

	var output, err = client.GetBucketPolicy(context.TODO(),
		&s3.GetBucketPolicyInput{
			Bucket: aws.String(bucketArn),
		})

	if err != nil {
		return "", inst.logErr(err)
	}

	return aws.ToString(output.Policy), nil
}

func (inst *S3BucketsApi) GetBucketTags(
	bucketArn string, force bool,
) ([]types.Tag, error) {
	var client = GetAwsApiClients().s3

	var output, err = client.GetBucketTagging(context.TODO(),
		&s3.GetBucketTaggingInput{
			Bucket: aws.String(bucketArn),
		})

	if err != nil {
		return nil, inst.logErr(err)
	}

	return output.TagSet, nil
}

func (inst *S3BucketsApi) HeadBucket(
	bucketArn string, objectKey string, force bool,
) (s3.HeadBucketOutput, error) {
	var client = GetAwsApiClients().s3
	var empty = s3.HeadBucketOutput{}
	var output, err = client.HeadBucket(context.TODO(),
		&s3.HeadBucketInput{
			Bucket: aws.String(bucketArn),
		})

	if err != nil {
		return empty, inst.logErr(err)
	}

	return *output, nil
}

func (inst *S3BucketsApi) HeadObject(
	bucketArn string, objectKey string, force bool,
) (s3.HeadObjectOutput, error) {
	var client = GetAwsApiClients().s3
	var empty = s3.HeadObjectOutput{}
	var output, err = client.HeadObject(context.TODO(),
		&s3.HeadObjectInput{
			Bucket: aws.String(bucketArn),
			Key:    aws.String(objectKey),
		})

	if err != nil {
		return empty, inst.logErr(err)
	}

	return *output, nil
}
