package ssm

import (
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type ParameterDetailsTable struct {
	*core.DetailsTable
	data       types.Parameter
	serviceCtx *core.ServiceContext[awsapi.SystemsManagerApi]
}

func NewParameterDetailsTable(
	serviceContext *core.ServiceContext[awsapi.SystemsManagerApi],
) *ParameterDetailsTable {
	var table = &ParameterDetailsTable{
		DetailsTable: core.NewDetailsTable("Parameter Details", serviceContext.AppContext),
		data:         types.Parameter{},
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()

	return table
}

func (inst *ParameterDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.Name)},
		{"ARN", aws.ToString(inst.data.ARN)},
		{"Type", string(inst.data.Type)},
		{"Version", fmt.Sprintf("%d", inst.data.Version)},
		{"Data Type", aws.ToString(inst.data.DataType)},
		{"Last Modified", core.FormatTime(inst.data.LastModifiedDate)},
	}

	// The value is shown on its own tab, not in the raw response
	var raw = inst.data
	raw.Value = nil
	inst.SetRawData(raw)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *ParameterDetailsTable) RefreshDetails(parameter types.Parameter) {
	inst.data = parameter
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDetailsTable()
	})
}
