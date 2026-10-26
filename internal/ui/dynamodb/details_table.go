package dynamodb

import (
	"fmt"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
)

type DynamoDBDetailsTable struct {
	*core.DetailsTable
	data          *types.TableDescription
	selectedTable string
	serviceCtx    *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBDetailsTable(
	serviceContext *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBDetailsTable {
	var table = &DynamoDBDetailsTable{
		DetailsTable:  core.NewDetailsTable("Table Details", serviceContext.AppContext),
		data:          nil,
		selectedTable: "",
		serviceCtx:    serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshDetails()
			return nil
		}
		return event
	})

	return table
}

// indexRows describe each GSI and then each LSI by its type, keys, projection
// and, for a GSI, its status.
func (inst *DynamoDBDetailsTable) indexRows() []core.TableRow {
	var rows []core.TableRow
	for _, gsi := range inst.data.GlobalSecondaryIndexes {
		rows = append(rows, core.TableRow{
			aws.ToString(gsi.IndexName),
			indexSummary("GSI", gsi.KeySchema, gsi.Projection, string(gsi.IndexStatus)),
		})
	}
	for _, lsi := range inst.data.LocalSecondaryIndexes {
		rows = append(rows, core.TableRow{
			aws.ToString(lsi.IndexName),
			indexSummary("LSI", lsi.KeySchema, lsi.Projection, ""),
		})
	}
	return rows
}

func indexSummary(
	indexType string, keySchema []types.KeySchemaElement, projection *types.Projection, status string,
) string {
	var keys []string
	for _, attr := range keySchema {
		keys = append(keys, aws.ToString(attr.AttributeName))
	}

	var parts = []string{indexType}
	if len(keys) > 0 {
		parts = append(parts, "Keys: "+strings.Join(keys, ", "))
	}
	if projection != nil && len(projection.ProjectionType) > 0 {
		parts = append(parts, "Projection: "+string(projection.ProjectionType))
	}
	if len(status) > 0 {
		parts = append(parts, status)
	}
	return strings.Join(parts, " · ")
}

func (inst *DynamoDBDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow

	if inst.data != nil {
		var deleteProtection = "DISABLED"
		if aws.ToBool(inst.data.DeletionProtectionEnabled) {
			deleteProtection = "ENABLED"
		}

		var billingMode = string(types.BillingModeProvisioned)
		if billing := inst.data.BillingModeSummary; billing != nil {
			billingMode = string(billing.BillingMode)
		}

		tableData = []core.TableRow{
			{"Name", aws.ToString(inst.data.TableName)},
			{"Status", fmt.Sprintf("%s", inst.data.TableStatus)},
			{"Created", core.FormatTime(inst.data.CreationDateTime)},
			{"Item Count", fmt.Sprintf("%d", aws.ToInt64(inst.data.ItemCount))},
			{"Billing Mode", billingMode},
			{"Delete Protection", deleteProtection},
			{"ARN", aws.ToString(inst.data.TableArn)},
		}

		var keyData = []core.TableRow{}
		for _, atter := range inst.data.KeySchema {
			switch atter.KeyType {
			case types.KeyTypeHash:
				keyData = append(keyData, core.TableRow{"Partition Key", aws.ToString(atter.AttributeName)})
			case types.KeyTypeRange:
				keyData = append(keyData, core.TableRow{"Sort Key", aws.ToString(atter.AttributeName)})
			}
		}
		tableData = append(tableData, core.DetailsSection("Keys", keyData...)...)
		tableData = append(tableData, core.DetailsSection("Indexes", inst.indexRows()...)...)

		if sse := inst.data.SSEDescription; sse != nil {
			tableData = append(tableData, core.DetailsSection("Encryption",
				core.TableRow{"Type", string(sse.SSEType)},
				core.TableRow{"Status", string(sse.Status)},
				core.TableRow{"KMS Key ARN", aws.ToString(sse.KMSMasterKeyArn)},
			)...)
		}
	}

	if inst.data != nil {
		inst.SetRawData(inst.data)
	} else {
		inst.SetRawData(nil)
	}
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *DynamoDBDetailsTable) RefreshDetails() {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var tableName = inst.selectedTable
	var description *types.TableDescription

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		description, err = inst.serviceCtx.Api.DescribeTable(inst.serviceCtx.Ctx, tableName)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = description
		inst.populateDetailsTable()
	})
}

func (inst *DynamoDBDetailsTable) SetSelectedTable(tableName string) {
	inst.selectedTable = tableName
}

func (inst *DynamoDBDetailsTable) GetSelectedTableArn() string {
	if inst.data == nil {
		return ""
	}
	return aws.ToString(inst.data.TableArn)
}
