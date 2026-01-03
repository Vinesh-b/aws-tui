package awsapi

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3BucketsApi struct {
	logger           *log.Logger
	allbuckets       []types.Bucket
	objectsPaginator *s3.ListObjectsV2Paginator
	bucketsPaginator *s3.ListBucketsPaginator
}

func NewS3BucketsApi(
	logger *log.Logger,
) *S3BucketsApi {
	return &S3BucketsApi{
		logger:     logger,
		allbuckets: []types.Bucket{},
	}
}

func (inst *S3BucketsApi) ListBuckets(force bool) ([]types.Bucket, error) {
	if len(inst.allbuckets) > 0 && !force {
		return inst.allbuckets, nil
	}

	var client = GetAwsApiClients().s3
	if force || inst.bucketsPaginator == nil {
		inst.bucketsPaginator = s3.NewListBucketsPaginator(
			client,
			&s3.ListBucketsInput{},
		)
	}

	var err error = nil
	var output *s3.ListBucketsOutput
	for inst.bucketsPaginator.HasMorePages() {
		output, err = inst.bucketsPaginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		inst.allbuckets = append(inst.allbuckets, output.Buckets...)
	}

	sort.Slice(inst.allbuckets, func(i, j int) bool {
		return aws.ToString(inst.allbuckets[i].Name) < aws.ToString(inst.allbuckets[j].Name)
	})

	return inst.allbuckets, nil
}

func (inst *S3BucketsApi) ListObjects(
	bucketName string,
	prefix string,
	force bool,
) ([]types.Object, []types.CommonPrefix, error) {
	if len(bucketName) == 0 {
		return nil, nil, fmt.Errorf("Bucket name not set")
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
		inst.logger.Println(err)
		return nil, nil, err
	}

	return output.Contents, output.CommonPrefixes, nil
}

func (inst *S3BucketsApi) DownloadFile(bucketName string, objectKey string, fileName string) error {
	if len(bucketName) == 0 {
		return fmt.Errorf("Bucket name not set")
	}

	if len(objectKey) == 0 {
		return fmt.Errorf("Object key not set")
	}

	if len(fileName) == 0 {
		return fmt.Errorf("File name not set")
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
	body, err := io.ReadAll(result.Body)
	if err != nil {
		inst.logger.Printf("Failed to object body: %v, Error: %v\n", objectKey, err)
	}

	_, err = file.Write(body)
	return err
}
