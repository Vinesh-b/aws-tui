package xray

import (
	"strconv"
	"strings"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

// ServiceDetailsTable shows a service of the service map with the statistics
// of the requests it handled, of its calls to other services and of their
// calls to it
type ServiceDetailsTable struct {
	*core.DetailsTable
	service  types.Service
	services []types.Service
}

func NewServiceDetailsTable(appCtx *core.AppContext) *ServiceDetailsTable {
	var table = &ServiceDetailsTable{
		DetailsTable: core.NewDetailsTable("Service Details", appCtx),
	}
	table.populateDetailsTable()
	return table
}

// SetService shows the service, whose calls are found in all the services of
// the map
func (inst *ServiceDetailsTable) SetService(service types.Service, services []types.Service) {
	inst.service = service
	inst.services = services
	inst.populateDetailsTable()
}

// serviceByReference is the name of the service of the reference of an edge
func serviceByReference(services []types.Service, referenceId *int32) string {
	for _, service := range services {
		if referenceId != nil && aws.ToInt32(service.ReferenceId) == *referenceId {
			return serviceName(service)
		}
	}
	return "Service " + strconv.Itoa(int(aws.ToInt32(referenceId)))
}

// callRows are the calls the service made to other services
func callRows(service types.Service, services []types.Service) []core.TableRow {
	var rows []core.TableRow
	for _, edge := range service.Edges {
		rows = append(rows, core.TableRow{serviceByReference(services, edge.ReferenceId), edgeStats(edge).summary()})
	}
	return rows
}

// calledByRows are the calls other services made to the service
func calledByRows(service types.Service, services []types.Service) []core.TableRow {
	var rows []core.TableRow
	if service.ReferenceId == nil {
		return rows
	}
	for _, caller := range services {
		for _, edge := range caller.Edges {
			if edge.ReferenceId != nil && *edge.ReferenceId == *service.ReferenceId {
				rows = append(rows, core.TableRow{serviceName(caller), edgeStats(edge).summary()})
			}
		}
	}
	return rows
}

func (inst *ServiceDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.service.ReferenceId != nil || inst.service.Name != nil {
		var service = inst.service
		var stats = serviceStats(service)
		var otherNames []string
		for _, name := range service.Names {
			if name != aws.ToString(service.Name) {
				otherNames = append(otherNames, name)
			}
		}

		tableData = []core.TableRow{
			{"Name", serviceName(service)},
			{"Other Names", strings.Join(otherNames, ", ")},
			{"Type", aws.ToString(service.Type)},
			{"Account", aws.ToString(service.AccountId)},
			{"State", aws.ToString(service.State)},
			{"Root", formatBool(service.Root)},
			{"First Segment", core.FormatTime(service.StartTime)},
			{"Last Segment", core.FormatTime(service.EndTime)},
		}

		if service.SummaryStatistics != nil {
			tableData = append(tableData, core.DetailsSection("Requests",
				core.TableRow{"Total", strconv.FormatInt(stats.total, 10)},
				core.TableRow{"OK", strconv.FormatInt(stats.ok, 10)},
				core.TableRow{"Errors", countAndPercent(stats, stats.errors)},
				core.TableRow{"Throttles", countAndPercent(stats, stats.throttles)},
				core.TableRow{"Faults", countAndPercent(stats, stats.faults)},
				core.TableRow{"Average", stats.average()},
				core.TableRow{"p50", stats.percentile(0.5)},
				core.TableRow{"p90", stats.percentile(0.9)},
				core.TableRow{"p99", stats.percentile(0.99)},
			)...)
		}
		tableData = append(tableData, core.DetailsSection("Calls", callRows(service, inst.services)...)...)
		tableData = append(tableData, core.DetailsSection("Called By", calledByRows(service, inst.services)...)...)
		rawData = service
	}

	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// countAndPercent is the count with its percentage of the requests
func countAndPercent(stats requestStats, count int64) string {
	var text = strconv.FormatInt(count, 10)
	if percent := stats.percentOf(count); percent != "" {
		text += " (" + percent + ")"
	}
	return text
}
