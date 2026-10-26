package lambda

import (
	"fmt"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaDetailsTable struct {
	*core.DetailsTable
	data       types.FunctionConfiguration
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

func formatOptionalInt(value *int32) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
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
	var tableData []core.TableRow
	var loggingConfig = inst.data.LoggingConfig
	var logGroup = ""
	var sysLogLevel = ""
	if loggingConfig != nil {
		logGroup = aws.ToString(loggingConfig.LogGroup)
		sysLogLevel = string(loggingConfig.SystemLogLevel)

	}

	tableData = []core.TableRow{
		{"Description", aws.ToString(inst.data.Description)},
		{"ARN", aws.ToString(inst.data.FunctionArn)},
		{"Version", aws.ToString(inst.data.Version)},
		{"Memory Size (MB)", formatOptionalInt(inst.data.MemorySize)},
		{"Runtime", string(inst.data.Runtime)},
		{"Architectures", core.FormatList(inst.data.Architectures)},
		{"Timeout (s)", formatOptionalInt(inst.data.Timeout)},
		{"Log Group", logGroup},
		{"System Log Level", sysLogLevel},
		{"State", string(inst.data.State)},
		{"Last Modified", formatLastModified(aws.ToString(inst.data.LastModified))},
		{"Role", aws.ToString(inst.data.Role)},
	}

	inst.SetTitleExtra(aws.ToString(inst.data.FunctionName))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaDetailsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaDetailsTable()
	})
}
