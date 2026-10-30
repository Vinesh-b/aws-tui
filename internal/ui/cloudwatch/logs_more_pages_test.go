package cloudwatch

import (
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/uitest"
)

const moreMarker = "⇣ more"

func TestLogEventsTable__TitleShowsThereAreMoreEventsToLoad(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)), cloudwatchfake.WithMoreLogEvents(true))

	var table = newFollowedEventsTable(t, api)

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return strings.Contains(table.GetTitle(), moreMarker)
	}, "the title to show there are more events")
}

func TestLogEventsTable__TitleHasNoMarkerAtTheEndOfTheEvents(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)))

	var table = newFollowedEventsTable(t, api)

	uitest.Consistently(t, table.serviceCtx.App, 100*time.Millisecond, func() bool {
		return !strings.Contains(table.GetTitle(), moreMarker)
	}, "the title to have no marker")
}

func TestLogEventsTable__MarkerGoesWhenTheLastEventsAreLoaded(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)), cloudwatchfake.WithMoreLogEvents(true))
	var table = newFollowedEventsTable(t, api)
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return strings.Contains(table.GetTitle(), moreMarker)
	}, "the marker")
	api.SetMoreLogEvents(false)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.RefreshLogEvents(false) })

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return !strings.Contains(table.GetTitle(), moreMarker)
	}, "the marker to go")
}

func TestLogEventsTable__FollowingHasNoMoreEventsToLoad(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents(newLogEvent("START", 0)), cloudwatchfake.WithMoreLogEvents(true))
	var table = newFollowedEventsTable(t, api)
	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return strings.Contains(table.GetTitle(), moreMarker)
	}, "the marker")

	pressFollow(t, table)

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return !strings.Contains(table.GetTitle(), moreMarker)
	}, "the marker to go while following")
}

func TestLogStreamsTable__TitleShowsThereAreMoreStreamsToLoad(t *testing.T) {
	var api = cloudwatchfake.New(
		cloudwatchfake.WithLogStreams(newLogStream("stream-a", 0)), cloudwatchfake.WithMoreLogStreams(true),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogStreamsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	uitest.OnUI(t, serviceCtx.App, func() { table.SetSelectedLogGroup("group") })

	table.RefreshStreams(true)

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return strings.Contains(table.GetTitle(), moreMarker)
	}, "the title to show there are more streams")
}
