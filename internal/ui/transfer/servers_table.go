package transfer

import (
	"context"
	"slices"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"

	"github.com/gdamore/tcell/v2"
)

// ServersTable lists the Transfer Family servers. The list has no protocols or
// endpoint details, so the rest is loaded for the one selected.
type ServersTable struct {
	*core.SelectableTable[types.ListedServer]
	data           []types.ListedServer
	selectedServer types.ListedServer
	serviceCtx     *core.ServiceContext[awsapi.TransferApi]
}

func NewServersTable(
	serviceContext *core.ServiceContext[awsapi.TransferApi],
) *ServersTable {
	var table = &ServersTable{
		SelectableTable: core.NewSelectableTable[types.ListedServer](
			"Servers",
			core.TableRow{
				"Server Id",
				"State",
				"Domain",
				"Endpoint Type",
				"Identity Provider",
				"Users",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1, 2, 3, 4),
		serviceCtx: serviceContext,
	}

	table.populateServersTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *ServersTable) populateServersTable() {
	var tableData []core.TableRow
	for _, server := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(server.ServerId),
			string(server.State),
			string(server.Domain),
			string(server.EndpointType),
			string(server.IdentityProviderType),
			strconv.Itoa(int(aws.ToInt32(server.UserCount))),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedServer = types.ListedServer{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshServers loads the servers
func (inst *ServersTable) RefreshServers() {
	inst.refreshServers(func() {})
}

func (inst *ServersTable) refreshServers(loaded func()) {
	var servers []types.ListedServer
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		servers, err = inst.serviceCtx.Api.ListServers(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = servers
		inst.populateServersTable()
		loaded()
	})
}

// Open selects the server of the ARN. The servers are loaded again first if
// it isn't one of those loaded.
func (inst *ServersTable) Open(serverArn string) {
	if inst.selectServer(serverArn) {
		return
	}

	inst.refreshServers(func() {
		if !inst.selectServer(serverArn) {
			inst.ErrorMessageCallback("Server %s not found", serverArn)
		}
	})
}

// selectServer shows all the servers, without a search, to select the one
// with the ARN
func (inst *ServersTable) selectServer(serverArn string) bool {
	var matches = func(server types.ListedServer) bool { return aws.ToString(server.Arn) == serverArn }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateServersTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *ServersTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedServer = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *ServersTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshServers()
			return nil
		}
		return capture(event)
	})
}

func (inst *ServersTable) GetSelectedServer() types.ListedServer {
	return inst.selectedServer
}
