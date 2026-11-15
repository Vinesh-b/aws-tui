package cloudtrail

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	"github.com/gdamore/tcell/v2"
)

type EventsTable struct {
	*core.SelectableTable[types.Event]
	selectedEvent types.Event
	serviceCtx    *core.ServiceContext[awsapi.CloudTrailApi]
}

func NewEventsTable(
	serviceContext *core.ServiceContext[awsapi.CloudTrailApi],
) *EventsTable {
	var table = &EventsTable{
		SelectableTable: core.NewSelectableTable[types.Event](
			"Event History",
			core.TableRow{
				"Time",
				"Event Name",
				"User",
				"Event Source",
				"Resource",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(4, 1).SetFilterColumns(1, 2, 3, 4),
		serviceCtx: serviceContext,
	}

	table.HighlightSearch = true
	table.populateEventsTable(nil, true)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetLoadMoreFunc(func() { table.RefreshEvents(false) })

	return table
}

// eventResource is the name of the first resource of the event, which is what
// most events are about
func eventResource(event types.Event) string {
	if len(event.Resources) == 0 {
		return ""
	}
	return aws.ToString(event.Resources[0].ResourceName)
}

func (inst *EventsTable) populateEventsTable(events []types.Event, reset bool) {
	var tableData []core.TableRow
	for _, event := range events {
		tableData = append(tableData, core.TableRow{
			core.FormatTime(event.EventTime),
			aws.ToString(event.EventName),
			aws.ToString(event.Username),
			aws.ToString(event.EventSource),
			eventResource(event),
		})
	}

	if !reset {
		inst.ExtendData(tableData, events)
		return
	}

	// Nothing is selected in an empty table
	if len(events) == 0 {
		inst.selectedEvent = types.Event{}
	}
	inst.SetData(tableData, events, 0)
	inst.Select(1, 0)
}

// RefreshEvents loads the newest events again, or the page after those shown
// when reset is false
func (inst *EventsTable) RefreshEvents(reset bool) {
	var events []types.Event
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		events, err = inst.serviceCtx.Api.LookupEvents(ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateEventsTable(events, reset)
	})
}

func (inst *EventsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedEvent = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *EventsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshEvents(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *EventsTable) GetSelectedEvent() types.Event {
	return inst.selectedEvent
}
