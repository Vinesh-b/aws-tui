package sqs

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newQueueAttributes() map[string]string {
	return map[string]string{
		"QueueArn":                              "arn:aws:sqs:us-east-1:123456789012:orders",
		"ApproximateNumberOfMessages":           "5",
		"ApproximateNumberOfMessagesNotVisible": "2",
		"ApproximateNumberOfMessagesDelayed":    "1",
		"VisibilityTimeout":                     "30",
		"MessageRetentionPeriod":                "345600",
		"DelaySeconds":                          "0",
		"ReceiveMessageWaitTimeSeconds":         "10",
		"MaximumMessageSize":                    "262144",
		"SqsManagedSseEnabled":                  "true",
		"RedrivePolicy":                         `{"maxReceiveCount":3}`,
		"CreatedTimestamp":                      "1704164645",
		"LastModifiedTimestamp":                 "1704164706",
	}
}

func newDetailsTable() *QueueDetailsTable {
	return NewQueueDetailsTable(uitest.NewServiceContext[awsapi.SqsApi]())
}

func TestQueueDetailsTable__EmptyQueueShowsBlankValues(t *testing.T) {
	var table = newDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "."},
		{"Type", "Standard"},
		{"URL", ""},
		{"ARN", ""},
		{"Encryption", "None"},
		{"Created", ""},
		{"Last Modified", ""},
		{"Messages"},
		{"Available", ""},
		{"In Flight", ""},
		{"Delayed", ""},
		{"Configuration"},
		{"Visibility Timeout (s)", ""},
		{"Message Retention (s)", ""},
		{"Delivery Delay (s)", ""},
		{"Receive Wait Time (s)", ""},
		{"Max Message Size (bytes)", ""},
		{"Redrive Policy", ""},
	})
}

func TestQueueDetailsTable__PopulatesFromAttributes(t *testing.T) {
	var table = newDetailsTable()
	table.queueUrl = queueUrl("orders")
	table.attributes = newQueueAttributes()

	table.populateQueueDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Type", "Standard"},
		{"URL", queueUrl("orders")},
		{"ARN", "arn:aws:sqs:us-east-1:123456789012:orders"},
		{"Encryption", "SQS managed"},
		{"Created", "2024-01-02 03:04:05"},
		{"Last Modified", "2024-01-02 03:05:06"},
		{"Messages"},
		{"Available", "5"},
		{"In Flight", "2"},
		{"Delayed", "1"},
		{"Configuration"},
		{"Visibility Timeout (s)", "30"},
		{"Message Retention (s)", "345600"},
		{"Delivery Delay (s)", "0"},
		{"Receive Wait Time (s)", "10"},
		{"Max Message Size (bytes)", "262144"},
		{"Redrive Policy", `{"maxReceiveCount":3}`},
	})
}

func TestQueueDetailsTable__KmsKeyTakesPrecedenceInEncryption(t *testing.T) {
	var table = newDetailsTable()
	table.queueUrl = queueUrl("orders")
	table.attributes = map[string]string{"KmsMasterKeyId": "alias/orders", "SqsManagedSseEnabled": "true"}

	table.populateQueueDetailsTable()

	if got := table.GetCellText(4, 1); got != "KMS: alias/orders" {
		t.Errorf("encryption = %q, want %q", got, "KMS: alias/orders")
	}
}

func TestQueueDetailsTable__FifoQueueShowsDeduplicationRow(t *testing.T) {
	var table = newDetailsTable()
	table.queueUrl = queueUrl("orders.fifo")
	table.attributes = map[string]string{"ContentBasedDeduplication": "true"}

	table.populateQueueDetailsTable()

	const fifoRow = 18
	if got := table.GetCellText(fifoRow, 0); got != "Content Based Deduplication" {
		t.Errorf("row %d label = %q, want %q", fifoRow, got, "Content Based Deduplication")
	}
	if got := table.GetCellText(fifoRow, 1); got != "true" {
		t.Errorf("row %d value = %q, want %q", fifoRow, got, "true")
	}
}
