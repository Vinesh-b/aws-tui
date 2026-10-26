package cloudwatch

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func TestLogsHomeView__OpeningLogGroupShowsItsStreams(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeCloudWatchLogsApi{streams: []types.LogStream{{LogStreamName: aws.String("2024/01/02/[$LATEST]abc")}}}
	var page = NewLogsHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener, ok = page.(core.ResourceOpener)
	if !ok || !opener.OpensArnService("logs") {
		t.Fatal("logs page should open logs resources")
	}
	var resource, _ = arn.Parse("arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/orders:*")

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	uitest.AwaitScreenText(t, appCtx.App, screen, "❬/aws/states/orders❭")
	uitest.AwaitScreenText(t, appCtx.App, screen, "2024/01/02/[$LATEST]abc")
}

func alarmWithDimension(namespace string, name string, value string) cwTypes.MetricAlarm {
	return cwTypes.MetricAlarm{
		AlarmArn:   aws.String("arn:aws:cloudwatch:us-east-1:123456789012:alarm:errors"),
		Namespace:  aws.String(namespace),
		Dimensions: []cwTypes.Dimension{{Name: aws.String(name), Value: aws.String(value)}},
	}
}

func TestAlarmDetailsTable__DimensionsLinkToTheirResources(t *testing.T) {
	var tests = []struct {
		namespace, name, value string
		want                   string
	}{
		{"AWS/Lambda", "FunctionName", "orders", "arn:aws:lambda:us-east-1:123456789012:function:orders"},
		{"AWS/SQS", "QueueName", "orders-dlq", "arn:aws:sqs:us-east-1:123456789012:orders-dlq"},
		{"AWS/EC2", "InstanceId", "i-123", ""},
		{"AWS/Lambda", "Resource", "orders:prod", ""},
	}

	for _, tt := range tests {
		var table = newAlarmDetailsTable()
		table.data = alarmWithDimension(tt.namespace, tt.name, tt.value)

		table.populateAlarmDetailsTable()

		const dimensionRow = 11
		if got := table.GetCellText(dimensionRow, 0); got != tt.name {
			t.Fatalf("row %d = %q, want %q", dimensionRow, got, tt.name)
		}
		var link, _ = table.GetLink(dimensionRow)
		if link != tt.want {
			t.Errorf("%s %s: link = %q, want %q", tt.namespace, tt.name, link, tt.want)
		}
	}
}
