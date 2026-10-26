package sns

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

type TopicDetailsTable struct {
	*core.DetailsTable
	topicArn   string
	attributes map[string]string
	serviceCtx *core.ServiceContext[awsapi.SnsApi]
}

func NewTopicDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.SnsApi],
) *TopicDetailsTable {
	var table = &TopicDetailsTable{
		DetailsTable: core.NewDetailsTable("Topic Details", serviceCtx.AppContext),
		topicArn:     "",
		attributes:   nil,
		serviceCtx:   serviceCtx,
	}

	table.populateTopicDetailsTable()

	return table
}

func (inst *TopicDetailsTable) encryption() string {
	if len(inst.attributes["KmsMasterKeyId"]) > 0 {
		return "KMS: " + inst.attributes["KmsMasterKeyId"]
	}
	return "None"
}

func (inst *TopicDetailsTable) populateTopicDetailsTable() {
	var attr = inst.attributes

	var tableData = []core.TableRow{
		{"Name", topicName(inst.topicArn)},
		{"Type", topicType(inst.topicArn)},
		{"ARN", inst.topicArn},
		{"Display Name", attr["DisplayName"]},
		{"Owner", attr["Owner"]},
		{"Encryption", inst.encryption()},
		{"Delivery Policy", attr["EffectiveDeliveryPolicy"]},
	}

	if topicType(inst.topicArn) == "FIFO" {
		tableData = append(tableData,
			core.TableRow{"Content Based Deduplication", attr["ContentBasedDeduplication"]},
		)
	}

	tableData = append(tableData, core.DetailsSection("Subscriptions",
		core.TableRow{"Confirmed", attr["SubscriptionsConfirmed"]},
		core.TableRow{"Pending", attr["SubscriptionsPending"]},
		core.TableRow{"Deleted", attr["SubscriptionsDeleted"]},
	)...)

	inst.SetTitleExtra(topicName(inst.topicArn))
	inst.SetRawData(inst.attributes)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *TopicDetailsTable) RefreshDetails(topicArn string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var attributes map[string]string

	dataLoader.AsyncLoadData(func() {
		var err error
		attributes, err = inst.serviceCtx.Api.GetTopicAttributes(inst.serviceCtx.Ctx, topicArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.topicArn = topicArn
		inst.attributes = attributes
		inst.populateTopicDetailsTable()
	})
}

// ClearDetails empties the table, e.g. after the topic was deleted.
func (inst *TopicDetailsTable) ClearDetails() {
	inst.topicArn = ""
	inst.attributes = nil
	inst.populateTopicDetailsTable()
}
