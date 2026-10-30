package sns

import (
	"fmt"
	"sort"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// pendingConfirmation is what the api gives as the ARN of a subscription that
// has not been confirmed yet.
const subscribeOverlayId = "SUBSCRIBE"
const unsubscribeOverlayId = "UNSUBSCRIBE"

// unsubscribeKey opens the unsubscribe confirmation for the selected row.
const unsubscribeKey = 'u'

const pendingConfirmation = "PendingConfirmation"

type SubscriptionsTable struct {
	*core.SelectableTable[string]
	SubscribeView   *FloatingSubscribeView
	UnsubscribeView *core.FloatingConfirmView
	topicArn        string
	// toUnsubscribe is what the unsubscribe view was shown for
	toUnsubscribe subscriptionRef
	serviceCtx    *core.ServiceContext[awsapi.SnsApi]
}

type subscriptionRef struct {
	topicArn string
	arn      string
	protocol string
	endpoint string
}

func NewSubscriptionsTable(
	serviceCtx *core.ServiceContext[awsapi.SnsApi],
) *SubscriptionsTable {
	var table = &SubscriptionsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Subscriptions",
			core.TableRow{"Protocol", "Endpoint", "Status", "Subscription Arn"},
			serviceCtx.AppContext,
		).SetColumnExpansion(1, 1),
		SubscribeView:   NewFloatingSubscribeView(serviceCtx.AppContext),
		UnsubscribeView: core.NewFloatingConfirmView(serviceCtx.AppContext, "Unsubscribe", "Unsubscribe"),
		serviceCtx:      serviceCtx,
	}

	table.AddRuneToggleOverlay(subscribeOverlayId, table.SubscribeView, subscribeKey, false)
	table.SubscribeView.Input.SetOnSubscribeFunc(table.subscribe)
	table.SubscribeView.Input.SetOnCloseFunc(func() {
		table.ToggleOverlay(subscribeOverlayId, true)
	})
	table.HelpView.View.AddItem(string(subscribeKey), "Subscribe to the topic", nil)

	table.AddRuneToggleOverlay(unsubscribeOverlayId, table.UnsubscribeView, unsubscribeKey, false)
	table.UnsubscribeView.Input.SetOnShowFunc(table.prepareUnsubscribe)
	table.UnsubscribeView.Input.SetOnConfirmFunc(func() {
		table.ToggleOverlay(unsubscribeOverlayId, true)
		table.unsubscribe(table.toUnsubscribe)
	})
	table.UnsubscribeView.Input.SetOnCancelFunc(func() {
		table.ToggleOverlay(unsubscribeOverlayId, true)
	})
	table.HelpView.View.AddItem(string(unsubscribeKey), "Unsubscribe the selected subscription", nil)

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
}

// RefreshSubscriptions loads the subscriptions of the topic. Without a topic
// the table is cleared.
func (inst *SubscriptionsTable) RefreshSubscriptions(topicArn string) {
	inst.topicArn = topicArn
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

// subscribe subscribes the endpoint to the topic of the table without blocking
// the UI, then reloads the subscriptions.
func (inst *SubscriptionsTable) subscribe(protocol string, endpoint string) {
	var topicArn = inst.topicArn
	var view = inst.SubscribeView.Input

	if topicArn == "" {
		view.SetStatusMessage("No topic selected")
		return
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		view.SetStatusMessage("Endpoint is empty")
		return
	}

	go func() {
		var subscriptionArn, err = inst.serviceCtx.Api.Subscribe(inst.serviceCtx.Ctx, topicArn, protocol, endpoint)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				view.SetStatusMessage(err.Error())
				return
			}

			view.SetStatusMessage("Subscription created: " + subscriptionArn)
			if topicArn == inst.topicArn {
				inst.RefreshSubscriptions(topicArn)
			}
		})
	}()
}

// prepareUnsubscribe notes the selected subscription and asks about it. It is
// kept so the table changing underneath can't change what is deleted.
func (inst *SubscriptionsTable) prepareUnsubscribe() {
	var row, _ = inst.GetTable().GetSelection()
	inst.toUnsubscribe = subscriptionRef{}

	if row < 1 || row >= inst.GetTable().GetRowCount() {
		inst.UnsubscribeView.Input.SetMessage("No subscription selected")
		return
	}

	inst.toUnsubscribe = subscriptionRef{
		topicArn: inst.topicArn,
		arn:      inst.GetCellText(row, 3),
		protocol: inst.GetCellText(row, 0),
		endpoint: inst.GetCellText(row, 1),
	}

	var message = fmt.Sprintf("Unsubscribe %s %s from %s?", inst.toUnsubscribe.protocol,
		inst.toUnsubscribe.endpoint, topicName(inst.topicArn))
	if inst.toUnsubscribe.arn == pendingConfirmation {
		message += "\n\nIt is pending confirmation and can't be deleted until it is confirmed."
	}
	inst.UnsubscribeView.Input.SetMessage(message)
}

// unsubscribe deletes the subscription without blocking the UI, then reloads
// the subscriptions. Failures are shown as errors.
func (inst *SubscriptionsTable) unsubscribe(ref subscriptionRef) {
	if ref.arn == "" {
		inst.ErrorMessageCallback("No subscription selected")
		return
	}

	go func() {
		var err = inst.serviceCtx.Api.Unsubscribe(inst.serviceCtx.Ctx, ref.arn)

		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
				return
			}

			if ref.topicArn == inst.topicArn {
				inst.RefreshSubscriptions(ref.topicArn)
			}
		})
	}()
}
