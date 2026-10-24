package eventbridge

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

func TestEventBusDetailsTable__EmptyBusShowsBlankValues(t *testing.T) {
	var table = NewEventBusDetailsTable(uitest.NewServiceContext[awsapi.EventBridgeApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", ""},
		{"Description", ""},
		{"Arn", ""},
		{"LastModified", "0001-01-01 00:00:00"},
	})
}

func TestEventBusDetailsTable__PopulatesFromBus(t *testing.T) {
	var table = NewEventBusDetailsTable(uitest.NewServiceContext[awsapi.EventBridgeApi]())
	table.data = newEventBus("orders")

	table.populateEventBusDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Description", "orders bus"},
		{"Arn", "arn:aws:events:us-east-1:123456789012:event-bus/orders"},
		{"LastModified", "2024-01-02 03:04:05"},
	})
}

func newEventBus(name string) types.EventBus {
	var modified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.EventBus{
		Name:             aws.String(name),
		Description:      aws.String(name + " bus"),
		Arn:              aws.String("arn:aws:events:us-east-1:123456789012:event-bus/" + name),
		LastModifiedTime: &modified,
	}
}
