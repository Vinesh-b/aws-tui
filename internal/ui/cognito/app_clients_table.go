package cognito

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gdamore/tcell/v2"
)

// AppClientsTable shows the app clients of the selected user pool. Like the
// users, they are only loaded once the tab is opened.
type AppClientsTable struct {
	*core.SelectableTable[idpTypes.UserPoolClientDescription]
	serviceCtx *core.ServiceContext[awsapi.CognitoApi]
	pool       idpTypes.UserPoolDescriptionType
	loadedFor  string
	visible    bool
}

func NewAppClientsTable(serviceCtx *core.ServiceContext[awsapi.CognitoApi]) *AppClientsTable {
	var table = &AppClientsTable{
		SelectableTable: core.NewSelectableTable[idpTypes.UserPoolClientDescription](
			"App Clients",
			core.TableRow{
				"Name",
				"Client Id",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceCtx,
	}

	table.showClients(nil)
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.loadedFor = ""
			table.Load()
			return nil
		}
		return event
	})

	return table
}

func (inst *AppClientsTable) showClients(clients []idpTypes.UserPoolClientDescription) {
	var tableData []core.TableRow
	for _, client := range clients {
		tableData = append(tableData, core.TableRow{
			aws.ToString(client.ClientName),
			aws.ToString(client.ClientId),
		})
	}

	inst.SetTitleExtra(aws.ToString(inst.pool.Name))
	inst.SetData(tableData, clients, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}

// SetUserPool selects the user pool whose app clients are shown, without
// loading them. Load loads them once the tab is open.
func (inst *AppClientsTable) SetUserPool(pool idpTypes.UserPoolDescriptionType) {
	if aws.ToString(pool.Id) == aws.ToString(inst.pool.Id) {
		return
	}
	inst.pool = pool
	inst.loadedFor = ""
	inst.showClients(nil)
}

func (inst *AppClientsTable) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the app clients of the user pool selected, if the tab is open
// and they aren't loaded yet
func (inst *AppClientsTable) Load() {
	var userPoolId = aws.ToString(inst.pool.Id)
	if !inst.visible || len(userPoolId) == 0 || inst.loadedFor == userPoolId {
		return
	}
	inst.loadedFor = userPoolId

	var clients []idpTypes.UserPoolClientDescription
	var loadFailed = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		clients, err = inst.serviceCtx.Api.ListUserPoolClients(ctx, userPoolId)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another user pool was selected while this one loaded
		if inst.loadedFor != userPoolId {
			return
		}
		// What was found before an error is shown, and opening the tab again
		// tries again
		if loadFailed {
			inst.loadedFor = ""
		}
		inst.showClients(clients)
	})
}
