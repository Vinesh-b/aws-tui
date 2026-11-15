package eventbridge

import (
	"slices"
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/eventbridge/schedulerfake"
	"aws-tui/internal/ui/uitest"
)

func TestSchedulesPageView__MovingThroughSchedulesGetsOnlyTheScheduleItStopsOn(t *testing.T) {
	var f = newSchedulesFixture(t, schedulerfake.New(schedulerfake.WithSchedules(
		newSchedule("default", "billing"), newSchedule("default", "invoices"),
		newSchedule("default", "orders"), newSchedule("default", "refunds"),
	)))
	uitest.Eventually(t, f.app(), func() bool { return f.table.GetTable().GetRowCount() == 5 })

	var passed, last string
	uitest.OnUI(t, f.app(), func() {
		for row := 2; row <= 4; row++ {
			f.table.Select(row, 0)
			if row == 3 {
				passed = scheduleIdOf(f.table.GetSelectedSchedule()).String()
			}
		}
		last = scheduleIdOf(f.table.GetSelectedSchedule()).String()
	})

	uitest.Eventually(t, f.app(), func() bool {
		return slices.Contains(f.api.Calls(), "get "+last)
	}, "the schedule stopped on to be loaded")
	uitest.Consistently(t, f.app(), 2*core.SelectionLoadDelay, func() bool {
		return !slices.Contains(f.api.Calls(), "get "+passed)
	}, "the schedule moved through not to be loaded")
}
