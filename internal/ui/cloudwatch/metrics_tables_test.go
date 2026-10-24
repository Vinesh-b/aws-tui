package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func newMetric(namespace, name string, dimensions ...types.Dimension) types.Metric {
	return types.Metric{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(name),
		Dimensions: dimensions,
	}
}

func newPopulatedMetricListTable(metrics ...types.Metric) *MetricListTable {
	var table = NewMetricsTable(uitest.NewServiceContext[awsapi.CloudWatchMetricsApi]())
	table.populateMetricsTable(metrics)
	return table
}

func TestMetricListTable__ShowsHeadingsAndOneRowPerMetric(t *testing.T) {
	var table = newPopulatedMetricListTable(
		newMetric("AWS/EC2", "CPUUtilization"),
		newMetric("AWS/S3", "BucketSizeBytes"),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Name", "Namespace"},
		{"CPUUtilization", "AWS/EC2"},
		{"BucketSizeBytes", "AWS/S3"},
	})
}

func TestMetricListTable__NameIncludesDimensions(t *testing.T) {
	var table = newPopulatedMetricListTable(
		newMetric("AWS/Lambda", "Invocations",
			types.Dimension{Name: aws.String("FunctionName"), Value: aws.String("orders")},
		),
		newMetric("AWS/ApplicationELB", "RequestCount",
			types.Dimension{Name: aws.String("LoadBalancer"), Value: aws.String("app/web/1")},
			types.Dimension{Name: aws.String("AvailabilityZone"), Value: aws.String("us-east-1a")},
		),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Name", "Namespace"},
		{"Invocations (FunctionName=orders)", "AWS/Lambda"},
		{"RequestCount (LoadBalancer=app/web/1, AvailabilityZone=us-east-1a)", "AWS/ApplicationELB"},
	})
}

func TestMetricListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedMetricListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name", "Namespace"}})
}

func TestMetricListTable__SelectingRowTracksSelectedMetric(t *testing.T) {
	var table = newPopulatedMetricListTable(
		newMetric("AWS/EC2", "CPUUtilization"),
		newMetric("AWS/S3", "BucketSizeBytes"),
	)

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedMetric().MetricName); got != "BucketSizeBytes" {
		t.Errorf("selected metric = %q, want %q", got, "BucketSizeBytes")
	}
}

func TestMetricListTable__SelectingHeadingKeepsPreviousSelection(t *testing.T) {
	var table = newPopulatedMetricListTable(
		newMetric("AWS/EC2", "CPUUtilization"),
		newMetric("AWS/S3", "BucketSizeBytes"),
	)
	table.Select(2, 0)

	table.Select(0, 0)

	if got := aws.ToString(table.GetSelectedMetric().MetricName); got != "BucketSizeBytes" {
		t.Errorf("selected metric = %q, want %q", got, "BucketSizeBytes")
	}
}

func TestMetricDetailsTable__ListsNamespaceNameAndDimensions(t *testing.T) {
	var table = NewMetricDetailsTable(uitest.NewServiceContext[awsapi.CloudWatchMetricsApi]())
	table.data = newMetric("AWS/EC2", "CPUUtilization",
		types.Dimension{Name: aws.String("InstanceId"), Value: aws.String("i-1")},
		types.Dimension{Name: aws.String("AutoScalingGroup"), Value: aws.String("asg-1")},
	)

	table.populateMetricDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Namespace", "AWS/EC2"},
		{"MetricName", "CPUUtilization"},
		{"Dimensions", ""},
		{"InstanceId", "i-1"},
		{"AutoScalingGroup", "asg-1"},
	})
}
