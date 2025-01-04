package awsapi

import (
	"aws-tui/internal/pkg/ui/core"
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
		allMetrics:      []types.Metric{},
		meticsPaginator: nil,
	}
}

func (inst *CloudWatchMetricsApi) ListMetrics(
	dims []types.DimensionFilter,
	namespace string,
	metricName string,
	force bool,
) ([]types.Metric, error) {
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

	inst.allMetrics = []types.Metric{}
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
		inst.allMetrics = append(inst.allMetrics, output.Metrics...)
	}

	sort.Slice(inst.allMetrics, func(i, j int) bool {
		return aws.ToString(inst.allMetrics[i].MetricName) < aws.ToString(inst.allMetrics[j].MetricName)
	})

	return inst.allMetrics, apiErr
}

func (inst *CloudWatchMetricsApi) FilterByName(name string) []types.Metric {
	return core.FuzzySearch(name, inst.allMetrics, func(v types.Metric) string {
		return aws.ToString(v.MetricName)
	})
}
