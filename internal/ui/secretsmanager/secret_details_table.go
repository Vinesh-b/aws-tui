package secretsmanager

import (
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

type SecretDetailsTable struct {
	*core.DetailsTable
	data       types.SecretListEntry
	serviceCtx *core.ServiceContext[awsapi.SecretsManagerApi]
}

func NewSecretDetailsTable(
	serviceContext *core.ServiceContext[awsapi.SecretsManagerApi],
) *SecretDetailsTable {
	var table = &SecretDetailsTable{
		DetailsTable: core.NewDetailsTable("Secret Details", serviceContext.AppContext),
		data:         types.SecretListEntry{},
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()

	return table
}

func (inst *SecretDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.Name)},
		{"ARN", aws.ToString(inst.data.ARN)},
		{"Description", aws.ToString(inst.data.Description)},
		{"KMS Key", aws.ToString(inst.data.KmsKeyId)},
		{"Rotation Enabled", fmt.Sprintf("%t", aws.ToBool(inst.data.RotationEnabled))},
		{"Created", core.FormatTime(inst.data.CreatedDate)},
		{"Last Changed", core.FormatTime(inst.data.LastChangedDate)},
		{"Last Accessed", core.FormatTime(inst.data.LastAccessedDate)},
	}

	inst.SetRawData(inst.data)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *SecretDetailsTable) RefreshDetails(secret types.SecretListEntry) {
	inst.data = secret
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDetailsTable()
	})
}
