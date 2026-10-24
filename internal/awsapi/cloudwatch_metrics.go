package awsapi

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchMetricsApi interface {
	ListMetrics(
		ctx context.Context,
		dims []types.DimensionFilter,
		namespace string,
		metricName string,
		force bool,
	) ([]types.Metric, error)
	GetMetricData(
		ctx context.Context,
		metric types.Metric,
		stat string,
		periodSeconds int32,
		start time.Time,
		end time.Time,
	) (types.MetricDataResult, error)
}

type cloudWatchMetricsApi struct {
	baseApi
	metricsPaginator *cloudwatch.ListMetricsPaginator
}

func NewCloudWatchMetricsApi(
	logger *log.Logger,
) CloudWatchMetricsApi {
	return &cloudWatchMetricsApi{
		baseApi:          baseApi{logger: logger},
		metricsPaginator: nil,
	}
}

func (inst *cloudWatchMetricsApi) ListMetrics(
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

// GetMetricData returns the datapoints of one metric in ascending time order.
func (inst *cloudWatchMetricsApi) GetMetricData(
	ctx context.Context,
	metric types.Metric,
	stat string,
	periodSeconds int32,
	start time.Time,
	end time.Time,
) (types.MetricDataResult, error) {
	var result = types.MetricDataResult{}

	if err := requireArg("metric name", aws.ToString(metric.MetricName)); err != nil {
		return result, err
	}

	var paginator = cloudwatch.NewGetMetricDataPaginator(
		GetAwsApiClients().cloudwatch,
		&cloudwatch.GetMetricDataInput{
			StartTime: &start,
			EndTime:   &end,
			ScanBy:    types.ScanByTimestampAscending,
			MetricDataQueries: []types.MetricDataQuery{{
				Id:         aws.String("m1"),
				ReturnData: aws.Bool(true),
				MetricStat: &types.MetricStat{
					Metric: &metric,
					Period: aws.Int32(periodSeconds),
					Stat:   aws.String(stat),
				},
			}},
		},
	)

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return types.MetricDataResult{}, inst.logErr(err)
		}
		for _, series := range output.MetricDataResults {
			result.Timestamps = append(result.Timestamps, series.Timestamps...)
			result.Values = append(result.Values, series.Values...)
		}
	}

	return result, nil
}
