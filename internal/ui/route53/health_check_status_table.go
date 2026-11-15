package route53

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gdamore/tcell/v2"
)

// HealthCheckStatusTable shows what each checker last saw of the selected
// health check. It changes all the time, so it is loaded each time its tab is
// opened and the selection stops on a health check, and again with reset.
type HealthCheckStatusTable struct {
	*core.SelectableTable[types.HealthCheckObservation]
	serviceCtx  *core.ServiceContext[awsapi.Route53Api]
	healthCheck types.HealthCheck
	loadedFor   string
	visible     bool
}

func NewHealthCheckStatusTable(serviceCtx *core.ServiceContext[awsapi.Route53Api]) *HealthCheckStatusTable {
	var table = &HealthCheckStatusTable{
		SelectableTable: core.NewSelectableTable[types.HealthCheckObservation](
			"Status",
			core.TableRow{
				"Region",
				"IP Address",
				"Checked",
				"Status",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(3, 1).SetFilterColumns(0, 3),
		serviceCtx: serviceCtx,
	}

	table.showObservations(nil)
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.loadedFor = ""
			table.Load()
			return nil
		}
		return event
	})

	return table
}

func (inst *HealthCheckStatusTable) showObservations(observations []types.HealthCheckObservation) {
	var tableData []core.TableRow
	for _, observation := range observations {
		var checked, status = "", ""
		if report := observation.StatusReport; report != nil {
			checked = core.FormatTime(report.CheckedTime)
			status = aws.ToString(report.Status)
		}
		tableData = append(tableData, core.TableRow{
			string(observation.Region),
			aws.ToString(observation.IPAddress),
			checked,
			status,
		})
	}

	inst.SetTitleExtra(aws.ToString(inst.healthCheck.Id))
	inst.SetData(tableData, observations, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}

// SetHealthCheck selects the health check whose status is shown, without
// loading it. Load loads it once the tab is open.
func (inst *HealthCheckStatusTable) SetHealthCheck(check types.HealthCheck) {
	if aws.ToString(check.Id) == aws.ToString(inst.healthCheck.Id) {
		return
	}
	inst.healthCheck = check
	inst.loadedFor = ""
	inst.showObservations(nil)
}

func (inst *HealthCheckStatusTable) SetVisible(visible bool) {
	inst.visible = visible
	// Opening the tab again shows the status as it is now
	if !visible {
		inst.loadedFor = ""
	}
	inst.Load()
}

// hasCheckers is whether Route 53 has checkers for the health check. A
// calculated one only combines the status of others, so it has none to ask.
func hasCheckers(check types.HealthCheck) bool {
	return check.HealthCheckConfig == nil || check.HealthCheckConfig.Type != types.HealthCheckTypeCalculated
}

// Load loads the status of the health check selected, if the tab is open and
// it isn't loaded yet
func (inst *HealthCheckStatusTable) Load() {
	var healthCheckId = aws.ToString(inst.healthCheck.Id)
	if !inst.visible || len(healthCheckId) == 0 || inst.loadedFor == healthCheckId ||
		!hasCheckers(inst.healthCheck) {
		return
	}
	inst.loadedFor = healthCheckId

	var observations []types.HealthCheckObservation
	var loadFailed = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		observations, err = inst.serviceCtx.Api.GetHealthCheckStatus(ctx, healthCheckId)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another health check was selected while this one loaded
		if inst.loadedFor != healthCheckId {
			return
		}
		if loadFailed {
			inst.loadedFor = ""
		}
		inst.showObservations(observations)
	})
}
