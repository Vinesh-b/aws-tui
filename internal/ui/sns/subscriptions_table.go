package sns

import (
	"sort"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// pendingConfirmation is what the api gives as the ARN of a subscription that
// has not been confirmed yet.
const pendingConfirmation = "PendingConfirmation"

type SubscriptionsTable struct {
	*core.SelectableTable[string]
	serviceCtx *core.ServiceContext[awsapi.SnsApi]
}

func NewSubscriptionsTable(
	serviceCtx *core.ServiceContext[awsapi.SnsApi],
) *SubscriptionsTable {
	var table = &SubscriptionsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Subscriptions",
			core.TableRow{"Protocol", "Endpoint", "Status", "Subscription Arn"},
			serviceCtx.AppContext,
		),
		serviceCtx: serviceCtx,
	}

	table.populateSubscriptionsTable(nil)

	return table
}

func subscriptionStatus(subscriptionArn string) string {
	if subscriptionArn == pendingConfirmation {
		return "Pending"
	}
	return "Confirmed"
}

func (inst *SubscriptionsTable) populateSubscriptionsTable(subscriptions []types.Subscription) {
	var sorted = append([]types.Subscription(nil), subscriptions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if protocol1, protocol2 := aws.ToString(sorted[i].Protocol), aws.ToString(sorted[j].Protocol); protocol1 != protocol2 {
			return protocol1 < protocol2
		}
		return aws.ToString(sorted[i].Endpoint) < aws.ToString(sorted[j].Endpoint)
	})

	var tableData []core.TableRow
	for _, subscription := range sorted {
		tableData = append(tableData, core.TableRow{
			aws.ToString(subscription.Protocol),
			aws.ToString(subscription.Endpoint),
			subscriptionStatus(aws.ToString(subscription.SubscriptionArn)),
			aws.ToString(subscription.SubscriptionArn),
		})
	}

	inst.SetData(tableData, nil, -1)
	inst.GetCell(0, 1).SetExpansion(1)
}

// RefreshSubscriptions loads the subscriptions of the topic. Without a topic
// the table is cleared.
func (inst *SubscriptionsTable) RefreshSubscriptions(topicArn string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var subscriptions []types.Subscription

	dataLoader.AsyncLoadData(func() {
		if topicArn == "" {
			return
		}

		var err error
		subscriptions, err = inst.serviceCtx.Api.ListTopicSubscriptions(inst.serviceCtx.Ctx, topicArn)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateSubscriptionsTable(subscriptions)
	})
}
