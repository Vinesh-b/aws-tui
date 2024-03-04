package cloudwatch

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchMetricsApi struct {
	logger          *log.Logger
	config          aws.Config
	client          *cloudwatch.Client
	allMetrics      []types.Metric
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
		allMetrics:      nil,
		meticsPaginator: nil,
	}
}

func (inst *CloudWatchMetricsApi) ListMetrics(
	dims []types.DimensionFilter,
	namespace string,
	metricName string,
	force bool,
) []types.Metric {
	if !force && inst.allMetrics != nil {
		return inst.allMetrics
	}

	var queryNamespace = &namespace
	if namespace == "" {
		queryNamespace = nil
	}

	var queryMetricName = &metricName
	if metricName == "" {
		queryMetricName = nil
	}

	inst.allMetrics = nil
	inst.meticsPaginator = cloudwatch.NewListMetricsPaginator(
		inst.client,
		&cloudwatch.ListMetricsInput{
			Dimensions: dims,
			MetricName: queryMetricName,
			Namespace:  queryNamespace,
		},
	)

	for inst.meticsPaginator.HasMorePages() {
		var output, err = inst.meticsPaginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		for _, val := range output.Metrics {
			inst.allMetrics = append(inst.allMetrics, val)
		}
	}

	return inst.allMetrics
}
