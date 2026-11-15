package cognito

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gdamore/tcell/v2"
)

// UsersTable shows the users of the selected user pool. A pool can have
// millions, so like the traces of X-Ray they are loaded a page at a time, the
// next when the cursor moves down from the last. Nothing is loaded until the
// tab is opened. The search only finds the users loaded.
type UsersTable struct {
	*core.SelectableTable[idpTypes.UserType]
	serviceCtx *core.ServiceContext[awsapi.CognitoApi]
	pool       idpTypes.UserPoolDescriptionType
	loadedFor  string
	visible    bool
	// loads counts the loads from the first page, so the pages of one that
	// was replaced by another before they came are not shown
	loads int
}

func NewUsersTable(serviceCtx *core.ServiceContext[awsapi.CognitoApi]) *UsersTable {
	var table = &UsersTable{
		SelectableTable: core.NewSelectableTable[idpTypes.UserType](
			"Users",
			core.TableRow{
				"Username",
				"Email",
				"Status",
				"Enabled",
				"Created",
				"Last Modified",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1, 2),
		serviceCtx: serviceCtx,
	}

	table.showUsers(nil, true)
	table.SetLoadMoreFunc(func() { table.refreshUsers(false) })
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

func (inst *UsersTable) showUsers(users []idpTypes.UserType, reset bool) {
	var tableData []core.TableRow
	for _, user := range users {
		tableData = append(tableData, core.TableRow{
			aws.ToString(user.Username),
			userAttribute(user, "email"),
			string(user.UserStatus),
			strconv.FormatBool(user.Enabled),
			core.FormatTime(user.UserCreateDate),
			core.FormatTime(user.UserLastModifiedDate),
		})
	}

	if !reset {
		inst.ExtendData(tableData, users)
		return
	}

	inst.SetTitleExtra(aws.ToString(inst.pool.Name))
	inst.SetData(tableData, users, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}

// SetUserPool selects the user pool whose users are shown, without loading
// them. Load loads them once the tab is open.
func (inst *UsersTable) SetUserPool(pool idpTypes.UserPoolDescriptionType) {
	if aws.ToString(pool.Id) == aws.ToString(inst.pool.Id) {
		return
	}
	inst.pool = pool
	inst.loadedFor = ""
	// The pages of the pool before aren't shown if they come
	inst.loads++
	inst.showUsers(nil, true)
}

func (inst *UsersTable) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the first page of the users of the user pool selected, if the
// tab is open and they aren't loaded yet
func (inst *UsersTable) Load() {
	var userPoolId = aws.ToString(inst.pool.Id)
	if !inst.visible || len(userPoolId) == 0 || inst.loadedFor == userPoolId {
		return
	}
	inst.loadedFor = userPoolId
	inst.refreshUsers(true)
}

// refreshUsers loads the first page of the users, or the page after those
// shown when reset is false
func (inst *UsersTable) refreshUsers(reset bool) {
	var userPoolId = aws.ToString(inst.pool.Id)
	// More can only be loaded after the first page
	if !reset && inst.loadedFor != userPoolId {
		inst.MoreLoaded()
		return
	}
	if reset {
		inst.loads++
	}
	var load = inst.loads

	var users []idpTypes.UserType
	var loadFailed = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		users, err = inst.serviceCtx.Api.ListUsers(ctx, userPoolId, reset)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another user pool was selected, or the users were loaded again,
		// while these were loading
		if load != inst.loads {
			inst.MoreLoaded()
			return
		}
		// Opening the tab again tries the first page again
		if loadFailed && reset {
			inst.loadedFor = ""
		}
		inst.showUsers(users, reset)
	})
}
