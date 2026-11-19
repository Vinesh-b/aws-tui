package transfer

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"
	"github.com/gdamore/tcell/v2"
)

// ServerUsersTable shows the users of the selected server. Like the
// associated resources of a WAF web ACL, nothing is loaded until its tab is
// opened.
type ServerUsersTable struct {
	*core.SelectableTable[types.ListedUser]
	serviceCtx *core.ServiceContext[awsapi.TransferApi]
	serverId   string
	loadedFor  string
	visible    bool
}

func NewServerUsersTable(serviceCtx *core.ServiceContext[awsapi.TransferApi]) *ServerUsersTable {
	var table = &ServerUsersTable{
		SelectableTable: core.NewSelectableTable[types.ListedUser](
			"Users",
			core.TableRow{
				"Username",
				"Home Directory",
				"Home Directory Type",
				"Role",
				"SSH Keys",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1, 3),
		serviceCtx: serviceCtx,
	}

	table.showUsers(nil)
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

func (inst *ServerUsersTable) showUsers(users []types.ListedUser) {
	var tableData []core.TableRow
	for _, user := range users {
		tableData = append(tableData, core.TableRow{
			aws.ToString(user.UserName),
			aws.ToString(user.HomeDirectory),
			string(user.HomeDirectoryType),
			aws.ToString(user.Role),
			strconv.Itoa(int(aws.ToInt32(user.SshPublicKeyCount))),
		})
	}

	inst.SetTitleExtra(inst.serverId)
	inst.SetData(tableData, users, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}

// SetServer selects the server whose users are shown, without loading them.
// Load loads them once the tab is open.
func (inst *ServerUsersTable) SetServer(serverId string) {
	if serverId == inst.serverId {
		return
	}
	inst.serverId = serverId
	inst.loadedFor = ""
	inst.showUsers(nil)
}

func (inst *ServerUsersTable) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the users of the server selected, if the tab is open and they
// aren't loaded yet
func (inst *ServerUsersTable) Load() {
	var serverId = inst.serverId
	if !inst.visible || len(serverId) == 0 || inst.loadedFor == serverId {
		return
	}
	inst.loadedFor = serverId

	var users []types.ListedUser
	var loadFailed = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		users, err = inst.serviceCtx.Api.ListUsers(ctx, serverId)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another server was selected while this one loaded
		if inst.loadedFor != serverId {
			return
		}
		// What was found before an error is shown, and opening the tab again
		// tries again
		if loadFailed {
			inst.loadedFor = ""
		}
		inst.showUsers(users)
	})
}
