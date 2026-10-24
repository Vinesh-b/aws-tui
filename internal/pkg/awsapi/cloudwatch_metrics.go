package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchMetricsApi struct {
	baseApi
	metricsPaginator *cloudwatch.ListMetricsPaginator
}

func NewCloudWatchMetricsApi(
	logger *log.Logger,
) *CloudWatchMetricsApi {
	return &CloudWatchMetricsApi{
		baseApi:          baseApi{logger: logger},
		metricsPaginator: nil,
	}
}

func (inst *CloudWatchMetricsApi) ListMetrics(
	ctx context.Context,
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

	inst.metricsPaginator = cloudwatch.NewListMetricsPaginator(
		GetAwsApiClients().cloudwatch,
		&cloudwatch.ListMetricsInput{
			Dimensions: dims,
			MetricName: queryMetricName,
			Namespace:  queryNamespace,
		},
	)

	var result, err = collectPaginator(inst.metricsPaginator.HasMorePages, func() ([]types.Metric, error) {
		var output, err = inst.metricsPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Metrics, nil
	})

	sortByKey(result, func(m types.Metric) *string { return m.MetricName })

	return result, inst.logErr(err)
}
