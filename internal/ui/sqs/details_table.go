package sqs

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

type QueueDetailsTable struct {
	*core.DetailsTable
	queueUrl   string
	attributes map[string]string
	serviceCtx *core.ServiceContext[awsapi.SqsApi]
}

func NewQueueDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.SqsApi],
) *QueueDetailsTable {
	var table = &QueueDetailsTable{
		DetailsTable: core.NewDetailsTable("Queue Details", serviceCtx.AppContext),
		queueUrl:     "",
		attributes:   nil,
		serviceCtx:   serviceCtx,
	}

	table.populateQueueDetailsTable()

	return table
}

func (inst *QueueDetailsTable) encryption() string {
	switch {
	case len(inst.attributes["KmsMasterKeyId"]) > 0:
		return "KMS: " + inst.attributes["KmsMasterKeyId"]
	case inst.attributes["SqsManagedSseEnabled"] == "true":
		return "SQS managed"
	default:
		return "None"
	}
}

func (inst *QueueDetailsTable) populateQueueDetailsTable() {
	var attr = inst.attributes

	var tableData = []core.TableRow{
		{"Name", queueName(inst.queueUrl)},
		{"Type", queueType(inst.queueUrl)},
		{"URL", inst.queueUrl},
		{"ARN", attr["QueueArn"]},
		{"Encryption", inst.encryption()},
		{"Created", core.FormatUnixSeconds(attr["CreatedTimestamp"])},
		{"Last Modified", core.FormatUnixSeconds(attr["LastModifiedTimestamp"])},
	}

	tableData = append(tableData, core.DetailsSection("Messages",
		core.TableRow{"Available", attr["ApproximateNumberOfMessages"]},
		core.TableRow{"In Flight", attr["ApproximateNumberOfMessagesNotVisible"]},
		core.TableRow{"Delayed", attr["ApproximateNumberOfMessagesDelayed"]},
	)...)

	var config = []core.TableRow{
		{"Visibility Timeout (s)", attr["VisibilityTimeout"]},
		{"Message Retention (s)", attr["MessageRetentionPeriod"]},
		{"Delivery Delay (s)", attr["DelaySeconds"]},
		{"Receive Wait Time (s)", attr["ReceiveMessageWaitTimeSeconds"]},
		{"Max Message Size (bytes)", attr["MaximumMessageSize"]},
		{"Redrive Policy", attr["RedrivePolicy"]},
	}
	if queueType(inst.queueUrl) == "FIFO" {
		config = append(config, core.TableRow{"Content Based Deduplication", attr["ContentBasedDeduplication"]})
	}
	tableData = append(tableData, core.DetailsSection("Configuration", config...)...)

	inst.SetTitleExtra(queueName(inst.queueUrl))
	inst.SetRawData(inst.attributes)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *QueueDetailsTable) RefreshDetails(queueUrl string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var attributes map[string]string

	dataLoader.AsyncLoadData(func() {
		var err error
		attributes, err = inst.serviceCtx.Api.GetQueueAttributes(inst.serviceCtx.Ctx, queueUrl)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.queueUrl = queueUrl
		inst.attributes = attributes
		inst.populateQueueDetailsTable()
	})
}
