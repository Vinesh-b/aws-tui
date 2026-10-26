package lambda

import (
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaDetailsTable struct {
	*core.DetailsTable
	data       types.FunctionConfiguration
	tags       core.DetailsTags
	serviceCtx *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaDetailsTable {
	var table = &LambdaDetailsTable{
		DetailsTable: core.NewDetailsTable("Lambda Details", serviceCtx.AppContext),
		data:         types.FunctionConfiguration{},
		serviceCtx:   serviceCtx,
	}

	table.populateLambdaDetailsTable()

	return table
}

// lambdaTimeFormat is how the lambda api gives the last modified time
const lambdaTimeFormat = "2006-01-02T15:04:05.000-0700"

func formatLastModified(text string) string {
	var t, err = time.Parse(lambdaTimeFormat, text)
	if err != nil {
		return text
	}
	return core.FormatTime(&t)
}

func (inst *LambdaDetailsTable) populateLambdaDetailsTable() {
	var tableData = []core.TableRow{
		{"Description", aws.ToString(inst.data.Description)},
		{"ARN", aws.ToString(inst.data.FunctionArn)},
		{"Version", aws.ToString(inst.data.Version)},
		{"State", string(inst.data.State)},
		{"Last Modified", formatLastModified(aws.ToString(inst.data.LastModified))},
		{"Role", aws.ToString(inst.data.Role)},
	}

	tableData = append(tableData, core.DetailsSection("Configuration",
		core.TableRow{"Runtime", string(inst.data.Runtime)},
		core.TableRow{"Architectures", core.FormatList(inst.data.Architectures)},
		core.TableRow{"Memory Size (MB)", core.FormatOptionalInt(inst.data.MemorySize)},
		core.TableRow{"Timeout (s)", core.FormatOptionalInt(inst.data.Timeout)},
	)...)

	if loggingConfig := inst.data.LoggingConfig; loggingConfig != nil {
		var logGroup = aws.ToString(loggingConfig.LogGroup)
		var logGroupArn = ""
		if len(logGroup) > 0 {
			logGroupArn = core.ArnFor(aws.ToString(inst.data.FunctionArn), "logs", "log-group:"+logGroup)
		}
		tableData = append(tableData, core.DetailsSection("Logging",
			core.DetailsLink("Log Group", logGroup, logGroupArn),
			core.TableRow{"System Log Level", string(loggingConfig.SystemLogLevel)},
		)...)
	}

	tableData = append(tableData, core.DetailsSection("Environment", environmentRows(inst.data)...)...)
	tableData = append(tableData, core.DetailsSection("VPC", vpcRows(inst.data)...)...)
	tableData = append(tableData, inst.tags.Section(aws.ToString(inst.data.FunctionArn))...)

	inst.SetTitleExtra(aws.ToString(inst.data.FunctionName))
	inst.SetRawData(inst.data)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func environmentRows(config types.FunctionConfiguration) []core.TableRow {
	if config.Environment == nil {
		return nil
	}
	return core.SortedRows(config.Environment.Variables)
}

func vpcRows(config types.FunctionConfiguration) []core.TableRow {
	var vpc = config.VpcConfig
	if vpc == nil || len(aws.ToString(vpc.VpcId)) == 0 {
		return nil
	}

	var rows = []core.TableRow{{"VPC ID", aws.ToString(vpc.VpcId)}}
	for _, id := range vpc.SubnetIds {
		rows = append(rows, core.TableRow{"Subnet ID", id})
	}
	for _, id := range vpc.SecurityGroupIds {
		rows = append(rows, core.TableRow{"Security Group ID", id})
	}
	return rows
}

// RefreshDetails shows the lambda, and then its tags once they are loaded
func (inst *LambdaDetailsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config
	inst.populateLambdaDetailsTable()

	var lambdaArn = aws.ToString(config.FunctionArn)
	inst.tags.Load(inst.serviceCtx.App, inst.Box, lambdaArn,
		func() string { return aws.ToString(inst.data.FunctionArn) },
		func() ([]core.TableRow, error) {
			var tags, err = inst.serviceCtx.Api.ListTags(inst.serviceCtx.Ctx, lambdaArn)
			return core.SortedRows(tags), err
		},
		inst.ErrorMessageCallback,
		inst.populateLambdaDetailsTable,
	)
}
