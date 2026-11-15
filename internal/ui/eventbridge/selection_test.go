package eventbridge

import (
	"strings"
	"testing"

	"aws-tui/internal/ui/eventbridge/schedulerfake"
	"aws-tui/internal/ui/uitest"
)

func TestSchedulesPageView__MovingThroughSchedulesGetsOnlyTheScheduleItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(
			newSchedule("default", "billing"), newSchedule("default", "invoices"),
			newSchedule("default", "orders"), newSchedule("default", "refunds"),
		)))
		uitest.Eventually(t, f.app(), func() bool { return f.table.GetTable().GetRowCount() == 5 })

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, f.app(), uitest.MoveThrough[string]{
			Table: f.table, Passed: "default/orders", StopsOn: "default/refunds",
			Loaded: func() []string {
				var got []string
				for _, call := range f.api.Calls() {
					if id, ok := strings.CutPrefix(call, "get "); ok {
						got = append(got, id)
					}
				}
				return got
			},
		})
	})
}
