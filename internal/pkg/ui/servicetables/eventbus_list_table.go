package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/gdamore/tcell/v2"
)

type EventBusListTable struct {
	*core.SelectableTable[types.EventBus]
	data             []types.EventBus
	filtered         []types.EventBus
	selectedEventBus types.EventBus
	serviceCtx       *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewEventBusListTable(
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *EventBusListTable {

	var view = &EventBusListTable{
		SelectableTable: core.NewSelectableTable[types.EventBus](
			"EventBuses",
			core.TableRow{
				"Name",
				"Description",
				"LastModified",
			},
			serviceCtx.AppContext,
		),
		data:             nil,
		selectedEventBus: types.EventBus{},
		serviceCtx:       serviceCtx,
	}

	view.populateEventBusTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *EventBusListTable) populateEventBusTable(data []types.EventBus) {
	var tableData []core.TableRow
	var privateData []types.EventBus
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			aws.ToString(row.Description),
			aws.ToTime(row.LastModifiedTime).Format(time.DateTime),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *EventBusListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
			name,
			inst.data,
			func(f types.EventBus) string {
				return aws.ToString(f.Name)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateEventBusTable(inst.filtered)
	})
}

func (inst *EventBusListTable) RefreshEventBuss(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListEventBuses(reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateEventBusTable(inst.data)
	})
}

func (inst *EventBusListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedEventBus = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *EventBusListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedEventBus = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *EventBusListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshEventBuss(true)
		}
		return capture(event)
	})
}

func (inst *EventBusListTable) GetSeletedEventBus() types.EventBus {
	return inst.selectedEventBus
}
