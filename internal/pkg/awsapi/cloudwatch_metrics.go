package awsapi

import (
	"context"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchMetricsApi struct {
	logger          *log.Logger
	config          aws.Config
	client          *cloudwatch.Client
	meticsPaginator *cloudwatch.ListMetricsPaginator
}

func NewCloudWatchMetricsApi(
	config aws.Config,
	logger *log.Logger,
) *CloudWatchMetricsApi {
	return &CloudWatchMetricsApi{
		config:          config,
		logger:          logger,
		client:          cloudwatch.NewFromConfig(config),
		meticsPaginator: nil,
	}
}

func (inst *CloudWatchMetricsApi) ListMetrics(
	dims []types.DimensionFilter,
	namespace string,
	metricName string,
	force bool,
) ([]types.Metric, error) {
	var queryNamespace = &namespace
	if namespace == "" {
		queryNamespace = nil
	}

	var queryMetricName = &metricName
	if metricName == "" {
		queryMetricName = nil
	}

	var result = []types.Metric{}
	inst.meticsPaginator = cloudwatch.NewListMetricsPaginator(
		inst.client,
		&cloudwatch.ListMetricsInput{
			Dimensions: dims,
			MetricName: queryMetricName,
			Namespace:  queryNamespace,
		},
	)

	var apiErr error = nil
	for inst.meticsPaginator.HasMorePages() {
		var output, err = inst.meticsPaginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			apiErr = err
			break
		}
		result = append(result, output.Metrics...)
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].MetricName) < aws.ToString(result[j].MetricName)
	})

	return result, apiErr
}
