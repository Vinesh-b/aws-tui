package transfer

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"

	"github.com/gdamore/tcell/v2"
)

// ConnectorsTable lists the SFTP and AS2 connectors. The list only has their
// URLs, so the rest is loaded for the one selected.
type ConnectorsTable struct {
	*core.SelectableTable[types.ListedConnector]
	data              []types.ListedConnector
	selectedConnector types.ListedConnector
	serviceCtx        *core.ServiceContext[awsapi.TransferApi]
}

func NewConnectorsTable(
	serviceContext *core.ServiceContext[awsapi.TransferApi],
) *ConnectorsTable {
	var table = &ConnectorsTable{
		SelectableTable: core.NewSelectableTable[types.ListedConnector](
			"Connectors",
			core.TableRow{
				"Connector Id",
				"URL",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateConnectorsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *ConnectorsTable) populateConnectorsTable() {
	var tableData []core.TableRow
	for _, connector := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(connector.ConnectorId),
			aws.ToString(connector.Url),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedConnector = types.ListedConnector{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshConnectors loads the connectors
func (inst *ConnectorsTable) RefreshConnectors() {
	inst.refreshConnectors(func() {})
}

func (inst *ConnectorsTable) refreshConnectors(loaded func()) {
	var connectors []types.ListedConnector
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		connectors, err = inst.serviceCtx.Api.ListConnectors(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = connectors
		inst.populateConnectorsTable()
		loaded()
	})
}

// Open selects the connector of the ARN. The connectors are loaded again
// first if it isn't one of those loaded.
func (inst *ConnectorsTable) Open(connectorArn string) {
	if inst.selectConnector(connectorArn) {
		return
	}

	inst.refreshConnectors(func() {
		if !inst.selectConnector(connectorArn) {
			inst.ErrorMessageCallback("Connector %s not found", connectorArn)
		}
	})
}

// selectConnector shows all the connectors, without a search, to select the
// one with the ARN
func (inst *ConnectorsTable) selectConnector(connectorArn string) bool {
	var matches = func(connector types.ListedConnector) bool { return aws.ToString(connector.Arn) == connectorArn }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateConnectorsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *ConnectorsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedConnector = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *ConnectorsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshConnectors()
			return nil
		}
		return capture(event)
	})
}

func (inst *ConnectorsTable) GetSelectedConnector() types.ListedConnector {
	return inst.selectedConnector
}
