package sns

import (
	"testing"

	"aws-tui/internal/ui/sns/snsfake"
)

var topicHeadings = []string{"Name", "Type"}

func topicArn(name string) string { return snsfake.TopicArn(name) }

func newTopicAttributes() map[string]string {
	return map[string]string{
		"DisplayName":             "Orders",
		"Owner":                   "123456789012",
		"SubscriptionsConfirmed":  "3",
		"SubscriptionsPending":    "1",
		"SubscriptionsDeleted":    "0",
		"KmsMasterKeyId":          "alias/aws/sns",
		"EffectiveDeliveryPolicy": `{"http":{}}`,
	}
}

func TestSnsFake__PendingConfirmationMatchesTheProductionConstant(t *testing.T) {
	if snsfake.PendingConfirmation != pendingConfirmation {
		t.Errorf("snsfake.PendingConfirmation = %q, want %q", snsfake.PendingConfirmation, pendingConfirmation)
	}
}
