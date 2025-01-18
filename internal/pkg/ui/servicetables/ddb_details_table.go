package servicetables

import (
	"fmt"
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBDetailsTable struct {
	*core.DetailsTable
	data *types.TableDescription

	selectedTable string
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.DynamoDBApi
}

func NewDynamoDBDetailsTable(
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBDetailsTable {
	var table = &DynamoDBDetailsTable{
		DetailsTable:  core.NewDetailsTable("Table Details"),
		data:          nil,
		selectedTable: "",
		logger:        logger,
		app:           app,
		api:           api,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshDetails()
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
		for _, atter := range inst.data.KeySchema {
			switch atter.KeyType {
			case types.KeyTypeHash:
				partitionKey = aws.ToString(atter.AttributeName)
			case types.KeyTypeRange:
				sortKey = aws.ToString(atter.AttributeName)
			}
		}

		var gsiList = []core.TableRow{}
		for _, gsi := range inst.data.GlobalSecondaryIndexes {
			gsiList = append(gsiList, core.TableRow{"GSI", fmt.Sprintf("%s", aws.ToString(gsi.IndexName))})
		}

		tableData = []core.TableRow{
			{"Name", aws.ToString(inst.data.TableName)},
			{"Status", fmt.Sprintf("%s", inst.data.TableStatus)},
			{"CreationDate", inst.data.CreationDateTime.Format(time.DateTime)},
			{"PartitionKey", partitionKey},
			{"SortKey", sortKey},
			{"ItemCount", fmt.Sprintf("%d", aws.ToInt64(inst.data.ItemCount))},
		}
		tableData = append(tableData, gsiList...)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *DynamoDBDetailsTable) RefreshDetails() {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.data, err = inst.api.DescribeTable(inst.selectedTable)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDetailsTable()
	})
}

func (inst *DynamoDBDetailsTable) SetSelectedTable(tableName string) {
	inst.selectedTable = tableName
}
