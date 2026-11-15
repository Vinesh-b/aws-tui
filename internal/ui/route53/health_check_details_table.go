package route53

import (
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// HealthCheckDetailsTable shows the health check selected, which the list has
// all of, with the settings of its type
type HealthCheckDetailsTable struct {
	*core.DetailsTable
}

func NewHealthCheckDetailsTable(
	serviceContext *core.ServiceContext[awsapi.Route53Api],
) *HealthCheckDetailsTable {
	var table = &HealthCheckDetailsTable{
		DetailsTable: core.NewDetailsTable("Health Check Details", serviceContext.AppContext),
	}
	table.SetHealthCheck(types.HealthCheck{})
	return table
}

// SetHealthCheck shows the health check
func (inst *HealthCheckDetailsTable) SetHealthCheck(check types.HealthCheck) {
	var tableData []core.TableRow
	var rawData any

	if check.Id != nil {
		var config = check.HealthCheckConfig
		if config == nil {
			config = &types.HealthCheckConfig{}
		}
		var healthCheckId = aws.ToString(check.Id)

		tableData = []core.TableRow{
			{"Id", healthCheckId},
			{"ARN", healthCheckArn(healthCheckId)},
			{"Type", string(config.Type)},
			{"Disabled", formatBool(config.Disabled)},
			{"Inverted", formatBool(config.Inverted)},
			{"Version", strconv.FormatInt(aws.ToInt64(check.HealthCheckVersion), 10)},
			{"Linked Service", linkedService(check.LinkedService)},
		}
		tableData = append(tableData, typeSettings(*config)...)
		rawData = check
	}

	inst.SetTitleExtra(aws.ToString(check.Id))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// typeSettings are the settings of a health check that its type uses
func typeSettings(config types.HealthCheckConfig) []core.TableRow {
	switch config.Type {
	case types.HealthCheckTypeCalculated:
		var rows = []core.TableRow{
			{"Healthy When", formatInt32(config.HealthThreshold) + " or more are healthy"},
		}
		// Each child opens on the health checks page
		for _, child := range config.ChildHealthChecks {
			rows = append(rows, core.DetailsLink("Child", child, healthCheckArn(child)))
		}
		return core.DetailsSection("Child Health Checks", rows...)

	case types.HealthCheckTypeCloudwatchMetric:
		var alarm = config.AlarmIdentifier
		if alarm == nil {
			alarm = &types.AlarmIdentifier{}
		}
		return core.DetailsSection("CloudWatch Alarm",
			core.TableRow{"Name", aws.ToString(alarm.Name)},
			core.TableRow{"Region", string(alarm.Region)},
			core.TableRow{"Insufficient Data", string(config.InsufficientDataHealthStatus)},
		)

	case types.HealthCheckTypeRecoveryControl:
		return core.DetailsSection("Recovery Control",
			core.TableRow{"Routing Control", aws.ToString(config.RoutingControlArn)},
		)
	}

	var regions []string
	for _, region := range config.Regions {
		regions = append(regions, string(region))
	}
	return core.DetailsSection("Endpoint",
		core.TableRow{"Target", healthCheckTarget(&config)},
		core.TableRow{"IP Address", aws.ToString(config.IPAddress)},
		core.TableRow{"Domain Name", aws.ToString(config.FullyQualifiedDomainName)},
		core.TableRow{"Port", formatInt32(config.Port)},
		core.TableRow{"Resource Path", aws.ToString(config.ResourcePath)},
		core.TableRow{"Search String", aws.ToString(config.SearchString)},
		core.TableRow{"Enable SNI", formatBool(config.EnableSNI)},
		core.TableRow{"Request Interval (s)", formatInt32(config.RequestInterval)},
		core.TableRow{"Failure Threshold", formatInt32(config.FailureThreshold)},
		core.TableRow{"Measure Latency", formatBool(config.MeasureLatency)},
		core.TableRow{"Checker Regions", strings.Join(regions, ", ")},
	)
}
