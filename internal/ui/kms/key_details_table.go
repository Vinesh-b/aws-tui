package kms

import (
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"github.com/gdamore/tcell/v2"
)

type KmsKeyDetailsTable struct {
	*core.DetailsTable
	data        *types.KeyMetadata
	selectedKey kmsKey
	tags        core.DetailsTags
	serviceCtx  *core.ServiceContext[awsapi.KmsApi]
}

func NewKmsKeyDetailsTable(
	serviceContext *core.ServiceContext[awsapi.KmsApi],
) *KmsKeyDetailsTable {
	var table = &KmsKeyDetailsTable{
		DetailsTable: core.NewDetailsTable("Key Details", serviceContext.AppContext),
		data:         nil,
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.tags.Clear()
			table.RefreshDetails()
			return nil
		}
		return event
	})

	return table
}

func (inst *KmsKeyDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow

	if inst.data != nil {
		tableData = []core.TableRow{
			{"Key ID", aws.ToString(inst.data.KeyId)},
			{"Aliases", inst.selectedKey.aliasText()},
			{"Description", aws.ToString(inst.data.Description)},
			{"State", string(inst.data.KeyState)},
			{"Enabled", fmt.Sprintf("%t", inst.data.Enabled)},
			{"Key Manager", string(inst.data.KeyManager)},
			{"Usage", string(inst.data.KeyUsage)},
			{"Key Spec", string(inst.data.KeySpec)},
			{"Origin", string(inst.data.Origin)},
			{"Multi-Region", fmt.Sprintf("%t", aws.ToBool(inst.data.MultiRegion))},
			{"Created", core.FormatTime(inst.data.CreationDate)},
			{"Scheduled Deletion", core.FormatTime(inst.data.DeletionDate)},
			{"ARN", aws.ToString(inst.data.Arn)},
		}

		tableData = append(tableData, core.DetailsSection("Algorithms",
			core.TableRow{"Encryption", core.FormatList(inst.data.EncryptionAlgorithms)},
			core.TableRow{"Signing", core.FormatList(inst.data.SigningAlgorithms)},
		)...)

		tableData = append(tableData, inst.tags.Section(inst.GetSelectedKeyArn())...)
	}

	inst.SetRawData(inst.data)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *KmsKeyDetailsTable) RefreshDetails() {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var keyId = inst.selectedKey.keyId
	var metadata *types.KeyMetadata

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		metadata, err = inst.serviceCtx.Api.DescribeKey(inst.serviceCtx.Ctx, keyId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another key was selected while this one loaded
		if inst.selectedKey.keyId != keyId {
			return
		}
		inst.data = metadata
		inst.populateDetailsTable()
		inst.loadTags(keyId)
	})
}

// loadTags loads the tags of the key once it is described, as its ARN is not
// known before then
func (inst *KmsKeyDetailsTable) loadTags(keyId string) {
	var keyArn = inst.GetSelectedKeyArn()
	inst.tags.Load(inst.serviceCtx.App, inst.Box, keyArn,
		func() string {
			if inst.selectedKey.keyId != keyId {
				return ""
			}
			return inst.GetSelectedKeyArn()
		},
		func() ([]core.TableRow, error) {
			var tags, err = inst.serviceCtx.Api.ListTags(inst.serviceCtx.Ctx, true, keyId)
			return core.TagRows(tags, func(t types.Tag) (string, string) {
				return aws.ToString(t.TagKey), aws.ToString(t.TagValue)
			}), err
		},
		inst.ErrorMessageCallback,
		inst.populateDetailsTable,
	)
}

func (inst *KmsKeyDetailsTable) SetSelectedKey(key kmsKey) {
	inst.selectedKey = key
}

func (inst *KmsKeyDetailsTable) GetSelectedKeyArn() string {
	if inst.data == nil {
		return ""
	}
	return aws.ToString(inst.data.Arn)
}
