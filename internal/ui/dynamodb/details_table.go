package dynamodb

import (
	"fmt"
	"time"

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

func (inst *DynamoDBDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var partitionKey = ""
	var sortKey = ""

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
			{"Creation Date", aws.ToTime(inst.data.CreationDateTime).Format(time.DateTime)},
			{"Item Count", fmt.Sprintf("%d", aws.ToInt64(inst.data.ItemCount))},
			{"Billing Mode", billingMode},
			{"Delete Protection", deleteProtection},
			{"Arn", aws.ToString(inst.data.TableArn)},
		}

		var keyData = []core.TableRow{}
		for _, atter := range inst.data.KeySchema {
			switch atter.KeyType {
			case types.KeyTypeHash:
				partitionKey = aws.ToString(atter.AttributeName)
				keyData = append(keyData, core.TableRow{"Partition Key", partitionKey})
			case types.KeyTypeRange:
				sortKey = aws.ToString(atter.AttributeName)
				keyData = append(keyData, core.TableRow{"Sort Key", sortKey})
			}
		}
		tableData = append(tableData, keyData...)

		var gsiList = []core.TableRow{}
		for _, gsi := range inst.data.GlobalSecondaryIndexes {
			gsiList = append(gsiList, core.TableRow{"GSI", fmt.Sprintf("%s", aws.ToString(gsi.IndexName))})
		}
		tableData = append(tableData, gsiList...)

		if sse := inst.data.SSEDescription; sse != nil {
			var sseData = []core.TableRow{
				{"Encryption Type", string(sse.SSEType)},
				{"Encryption Status", string(sse.Status)},
				{"KMS Key Arn", aws.ToString(sse.KMSMasterKeyArn)},
			}
			tableData = append(tableData, sseData...)
		}
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
