package s3

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3BucketsApi struct {
	logger           *log.Logger
	config           aws.Config
	client           *s3.Client
	allbuckets       map[string]types.Bucket
	objectsPaginator *s3.ListObjectsV2Paginator
}

func NewS3BucketsApi(
	config aws.Config,
	logger *log.Logger,
) *S3BucketsApi {
	return &S3BucketsApi{
		config:     config,
		logger:     logger,
		client:     s3.NewFromConfig(config),
		allbuckets: make(map[string]types.Bucket),
	}
}

func (inst *S3BucketsApi) ListBuckets(force bool) map[string]types.Bucket {
	if len(inst.allbuckets) > 0 && !force {
		return inst.allbuckets
	}

	inst.allbuckets = make(map[string]types.Bucket)

	var output, err = inst.client.ListBuckets(
		context.TODO(), &s3.ListBucketsInput{},
	)

	if err != nil {
		inst.logger.Println(err)
		return inst.allbuckets
	}

	for _, bucket := range output.Buckets {
		inst.allbuckets[*bucket.Name] = bucket
	}

	return inst.allbuckets
}

func (inst *S3BucketsApi) FilterByName(name string) map[string]types.Bucket {

	if len(inst.allbuckets) < 1 {
		inst.ListBuckets(true)
	}

	var foundBuckets = make(map[string]types.Bucket)

	for _, info := range inst.allbuckets {
		found := strings.Contains(*info.Name, name)
		if found {
			foundBuckets[*info.Name] = info
		}
	}
	return foundBuckets
}

func (inst *S3BucketsApi) ListObjects(
	bucketName string,
	prefix string,
	force bool,
) ([]types.Object, []types.CommonPrefix) {
	var objPrefix = &prefix
	if len(prefix) == 0 {
		objPrefix = nil
	}
	if force || inst.objectsPaginator == nil {
		inst.objectsPaginator = s3.NewListObjectsV2Paginator(
			inst.client, &s3.ListObjectsV2Input{
				Bucket:    aws.String(bucketName),
				MaxKeys:   aws.Int32(200),
				Delimiter: aws.String("/"),
				Prefix:    objPrefix,
			})
	}

	if !inst.objectsPaginator.HasMorePages() {
		return nil, nil
	}

	var output, err = inst.objectsPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return nil, nil
	}

	return output.Contents, output.CommonPrefixes
}
