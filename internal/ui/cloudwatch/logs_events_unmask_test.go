package cloudwatch

import (
	"testing"

	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func TestLogEventsTable__UnmaskReloadsTheEventsUnmaskedAndTogglesBack(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))
	var table = newFollowedEventsTable(t, api)

	uitest.OnUI(t, table.serviceCtx.App, func() {
		uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Unmask)
	})
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return len(api.CallsWithPrefix("unmask true")) == 1 && len(api.CallsWithPrefix("list ")) == 2
	}, "the events to be listed again unmasked")

	uitest.OnUI(t, table.serviceCtx.App, func() {
		uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Unmask)
	})
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return len(api.CallsWithPrefix("unmask false")) == 1 && len(api.CallsWithPrefix("list ")) == 3
	}, "the events to be listed again masked")
}
