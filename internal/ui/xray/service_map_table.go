package xray

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"

	"github.com/gdamore/tcell/v2"
)

// ServiceMapTable lists the services of the service map of a time range that
// ends now, with the statistics of the requests they handled. The ranges are
// those of the traces, but for Custom.
type ServiceMapTable struct {
	*core.SelectableTable[types.Service]
	services        []types.Service
	selectedService types.Service
	serviceCtx      *core.ServiceContext[awsapi.XRayApi]
	rangeIdx        int
	now             func() time.Time
	// loads counts the loads, so the services of one that was replaced by
	// another before they came are not shown
	loads int
}

func NewServiceMapTable(
	serviceContext *core.ServiceContext[awsapi.XRayApi],
) *ServiceMapTable {
	var table = &ServiceMapTable{
		SelectableTable: core.NewSelectableTable[types.Service](
			"Service Map",
			core.TableRow{
				"Name",
				"Type",
				"Requests",
				"Errors",
				"Faults",
				"Average",
				"p99",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
		rangeIdx:   defaultTraceRangeIdx,
		now:        time.Now,
	}

	table.SetTitleExtra(traceRanges[table.rangeIdx].label)
	table.HelpView.View.
		AddRune(core.APP_KEY_BINDINGS.TimeRange, "Show the service map of the next time range")
	table.SetRowColourFunc(func(_ core.TableRow, service types.Service) (tcell.Color, bool) {
		return serviceStats(service).colour()
	})
	table.populateServiceMapTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

// serviceRow is the row of a service in the list
func serviceRow(service types.Service) core.TableRow {
	var stats = serviceStats(service)
	var requests = ""
	if service.SummaryStatistics != nil {
		requests = strconv.FormatInt(stats.total, 10)
	}
	return core.TableRow{
		serviceName(service),
		aws.ToString(service.Type),
		requests,
		stats.percentOf(stats.errors),
		stats.percentOf(stats.faults),
		stats.average(),
		stats.percentile(0.99),
	}
}

func (inst *ServiceMapTable) populateServiceMapTable() {
	var tableData []core.TableRow
	for _, service := range inst.services {
		tableData = append(tableData, serviceRow(service))
	}

	// Nothing is selected in an empty table
	if len(inst.services) == 0 {
		inst.selectedService = types.Service{}
	}
	inst.SetData(tableData, inst.services, 0)
	inst.Select(1, 0)
}

// RefreshServices loads the service map of the time range again
func (inst *ServiceMapTable) RefreshServices() {
	inst.loads++
	var load = inst.loads
	var end = inst.now()
	var start = end.Add(-traceRanges[inst.rangeIdx].span)
	inst.SetTitleExtra(traceRanges[inst.rangeIdx].label)
	inst.RefreshTitle(0)

	var services []types.Service
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		services, err = inst.serviceCtx.Api.GetServiceGraph(ctx, start, end)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The map was loaded again while this one was loading
		if load != inst.loads {
			return
		}
		inst.services = sortedServices(services)
		inst.populateServiceMapTable()
	})
}

// sortedServices are the services by name, with the clients that called them
// first
func sortedServices(services []types.Service) []types.Service {
	var sorted = slices.Clone(services)
	var isClient = func(service types.Service) bool { return aws.ToString(service.Type) == "client" }
	slices.SortStableFunc(sorted, func(a, b types.Service) int {
		if isClient(a) != isClient(b) {
			if isClient(a) {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(serviceName(a)), strings.ToLower(serviceName(b)))
	})
	return sorted
}

// NextTimeRange loads the service map of the next range ending now, going
// back to the shortest after the longest
func (inst *ServiceMapTable) NextTimeRange() {
	inst.rangeIdx = (inst.rangeIdx + 1) % customTraceRangeIdx
	inst.RefreshServices()
}

// Services are all the services of the map, which the calls between them are
// found in
func (inst *ServiceMapTable) Services() []types.Service {
	return inst.services
}

func (inst *ServiceMapTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedService = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *ServiceMapTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshServices()
			return nil
		case core.APP_KEY_BINDINGS.TimeRange:
			inst.NextTimeRange()
			return nil
		}
		return capture(event)
	})
}

func (inst *ServiceMapTable) GetSelectedService() types.Service {
	return inst.selectedService
}
