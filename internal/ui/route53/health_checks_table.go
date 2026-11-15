package route53

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/gdamore/tcell/v2"
)

// HealthChecksTable lists the health checks of the account. The list has all
// of each health check, so only their status is loaded for the one selected.
type HealthChecksTable struct {
	*core.SelectableTable[types.HealthCheck]
	data                []types.HealthCheck
	selectedHealthCheck types.HealthCheck
	serviceCtx          *core.ServiceContext[awsapi.Route53Api]
}

func NewHealthChecksTable(
	serviceContext *core.ServiceContext[awsapi.Route53Api],
) *HealthChecksTable {
	var table = &HealthChecksTable{
		SelectableTable: core.NewSelectableTable[types.HealthCheck](
			"Health Checks",
			core.TableRow{
				"Id",
				"Type",
				"Target",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(2, 1).SetFilterColumns(0, 1, 2),
		serviceCtx: serviceContext,
	}

	table.populateHealthChecksTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *HealthChecksTable) populateHealthChecksTable() {
	var tableData []core.TableRow
	for _, check := range inst.data {
		var checkType = ""
		if check.HealthCheckConfig != nil {
			checkType = string(check.HealthCheckConfig.Type)
		}
		tableData = append(tableData, core.TableRow{
			aws.ToString(check.Id),
			checkType,
			healthCheckTarget(check.HealthCheckConfig),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedHealthCheck = types.HealthCheck{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshHealthChecks loads the health checks
func (inst *HealthChecksTable) RefreshHealthChecks() {
	inst.refreshHealthChecks(func() {})
}

func (inst *HealthChecksTable) refreshHealthChecks(loaded func()) {
	var checks []types.HealthCheck
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		checks, err = inst.serviceCtx.Api.ListHealthChecks(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = checks
		inst.populateHealthChecksTable()
		loaded()
	})
}

// Open selects the health check of the id. The health checks are loaded
// again first if it isn't one of those loaded.
func (inst *HealthChecksTable) Open(healthCheckId string) {
	if inst.selectHealthCheck(healthCheckId) {
		return
	}

	inst.refreshHealthChecks(func() {
		if !inst.selectHealthCheck(healthCheckId) {
			inst.ErrorMessageCallback("Health check %s not found", healthCheckId)
		}
	})
}

// selectHealthCheck shows all the health checks, without a search, to select
// the one with the id
func (inst *HealthChecksTable) selectHealthCheck(healthCheckId string) bool {
	var matches = func(check types.HealthCheck) bool { return aws.ToString(check.Id) == healthCheckId }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateHealthChecksTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *HealthChecksTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedHealthCheck = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *HealthChecksTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshHealthChecks()
			return nil
		}
		return capture(event)
	})
}

func (inst *HealthChecksTable) GetSelectedHealthCheck() types.HealthCheck {
	return inst.selectedHealthCheck
}
