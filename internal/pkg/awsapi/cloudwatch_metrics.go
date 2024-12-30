package awsapi

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchMetricsApi struct {
	logger          *log.Logger
	config          aws.Config
	client          *cloudwatch.Client
	allMetrics      map[string]types.Metric
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
		allMetrics:      make(map[string]types.Metric),
		meticsPaginator: nil,
	}
}

func (inst *CloudWatchMetricsApi) ListMetrics(
	dims []types.DimensionFilter,
	namespace string,
	metricName string,
	force bool,
) (map[string]types.Metric, error) {
	if len(inst.allMetrics) > 0 && !force {
		return inst.allMetrics, nil
	}

	var queryNamespace = &namespace
	if namespace == "" {
		queryNamespace = nil
	}

	var queryMetricName = &metricName
	if metricName == "" {
		queryMetricName = nil
	}

	inst.allMetrics = make(map[string]types.Metric)
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

		for _, val := range output.Metrics {
			inst.allMetrics[*val.MetricName] = val
		}
	}

	return inst.allMetrics, apiErr
}

func (inst *CloudWatchMetricsApi) FilterByName(name string) map[string]types.Metric {

	if len(inst.allMetrics) < 1 {
		return nil
	}

	var foundMetrics = make(map[string]types.Metric)

	for _, val := range inst.allMetrics {
		found := strings.Contains(*val.MetricName, name)
		if found {
			foundMetrics[*val.MetricName] = val
		}
	}
	return foundMetrics
}
